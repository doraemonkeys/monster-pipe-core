package forwarder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

type MonsterPipeCoreForwardTunnel struct {
	//  We need to close the input when the tunnel exits, otherwise the reading of the input will be blocked
	input         net.Conn
	outputs       []*ForwardOutput
	closed        atomic.Bool
	tunnelWatcher func(message ForwardConnMessage)
}

func NewForwardTunnel(input net.Conn, outputs []*ForwardOutput, tunnelWatcher func(message ForwardConnMessage)) *MonsterPipeCoreForwardTunnel {
	return &MonsterPipeCoreForwardTunnel{
		input:         input,
		outputs:       outputs,
		tunnelWatcher: tunnelWatcher,
	}
}

func (m *MonsterPipeCoreForwardTunnel) Close() error {
	if m.closed.Load() {
		return nil
	}
	m.closed.Store(true)
	var err error
	for _, output := range m.outputs {
		if e := output.Close(); e != nil {
			err = e
		}
	}
	if e := m.input.Close(); e != nil {
		err = e
	}
	return err
}

type ForwardConnMessageType int

const (
	ForwardConnMsgTypeInputRead          ForwardConnMessageType = 1
	ForwardConnMsgTypeInputReadError     ForwardConnMessageType = 2
	ForwardConnMsgTypeWriteToInputError  ForwardConnMessageType = 3
	ForwardConnMsgTypeOutputRead         ForwardConnMessageType = 4
	ForwardConnMsgTypeWriteToOutputOK    ForwardConnMessageType = 5
	ForwardConnMsgTypeWriteToOutputError ForwardConnMessageType = 6
	ForwardConnMsgTypeOutputReadError    ForwardConnMessageType = 7
	ForwardConnMsgTypeTunnelClosed       ForwardConnMessageType = 8
	ForwardConnMsgTypeWriteToInputOK     ForwardConnMessageType = 9
)

type ForwardConnMessage struct {
	MessageType    ForwardConnMessageType
	Output         ForwardOutputConfig
	OutputAddr     net.Addr
	ClosedByOutput bool
	Data           []byte
	Err            error
}

func (f ForwardConnMessage) Address() string {
	if f.OutputAddr != nil {
		fmt.Println("OutputAddr", f.OutputAddr.String())
		return f.OutputAddr.String()
	}
	return f.Output.Target()
}

// func (f ForwardConnMessage) PrettyPrint() string {
// 	// return fmt.Sprintf("MessageType: %s, Output: %s, Data: %s, Err: %v", f.messageType, f.output, f.data, f.err)
// 	if f.err != nil {
// 		return fmt.Sprintf("Tunnel Message: %v, Output: %+v, Err: %v", f.messageType, f.output, f.err)
// 	}
// 	o := fmt.Sprintf("Tunnel Message: %v", f.messageType)
// 	if f.output.Host != "" && f.output.Port != 0 {
// 		o += fmt.Sprintf(", Output: %+v", f.output)
// 	}
// 	if len(f.data) > 0 {
// 		if len(f.data) > 100 {
// 			return fmt.Sprintf("%s\nData: |%s......%s|", o, string(f.data[:50]), string(f.data[len(f.data)-50:]))
// 		}
// 		return fmt.Sprintf("%s\nData: |%s|", o, string(f.data))
// 	}
// 	return o
// }

// Run is the main loop of the tunnel, it will read data from the input and write to the outputs.
//
// Input will be closed when the tunnel exits.
func (m *MonsterPipeCoreForwardTunnel) Run(ctx context.Context) {
	closedByOutput := false
	defer func() {
		_ = m.Close()
		m.tunnelWatcher(ForwardConnMessage{
			MessageType:    ForwardConnMsgTypeTunnelClosed,
			ClosedByOutput: closedByOutput,
		})
	}()
	if m.tunnelWatcher == nil {
		m.tunnelWatcher = func(ForwardConnMessage) {}
	}
	var closedOutputCount atomic.Int32
	for _, output := range m.outputs {
		go func() {
			defer func() {
				closedOutputCount.Add(1)
				if closedOutputCount.Load() == int32(len(m.outputs)) && !m.closed.Load() {
					closedByOutput = true
					_ = m.Close() // Close input to make input.Read() not blocked
				}
			}()
			const maxBufferSize = 1024 * 1024 * 2
			var readBuffer [maxBufferSize]byte
			for {
				n, err := output.Read(ctx, readBuffer[:])
				if !output.config.Readable {
					// Read the received data, even if you may not process them immediately.
					continue
				}
				if err != nil {
					if m.closed.Load() || errors.Is(err, io.EOF) {
						return
					}
					m.tunnelWatcher(ForwardConnMessage{
						MessageType: ForwardConnMsgTypeOutputReadError,
						Err:         err,
						Output:      output.config,
						OutputAddr:  output.ConnAddr(),
					})
					return
				}
				m.tunnelWatcher(ForwardConnMessage{
					MessageType: ForwardConnMsgTypeOutputRead,
					Data:        readBuffer[:n],
					Output:      output.config,
					OutputAddr:  output.ConnAddr(),
				})
				_, err = m.input.Write(readBuffer[:n])
				if err != nil {
					m.tunnelWatcher(ForwardConnMessage{
						MessageType: ForwardConnMsgTypeWriteToInputError,
						Err:         err,
						Output:      output.config,
						OutputAddr:  output.ConnAddr(),
					})
				} else {
					m.tunnelWatcher(ForwardConnMessage{
						MessageType: ForwardConnMsgTypeWriteToInputOK,
						Data:        readBuffer[:n],
						Output:      output.config,
						OutputAddr:  output.ConnAddr(),
					})
				}
			}
		}()
	}
	const maxBufferSize = 1024 * 1024 * 2
	var readBuffer [maxBufferSize]byte
	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := m.input.Read(readBuffer[:])
		if err != nil {
			if m.closed.Load() || errors.Is(err, io.EOF) {
				return
			}
			m.tunnelWatcher(ForwardConnMessage{
				MessageType: ForwardConnMsgTypeInputReadError,
				Err:         err,
			})
			return
		}
		m.tunnelWatcher(ForwardConnMessage{
			MessageType: ForwardConnMsgTypeInputRead,
			Data:        readBuffer[:n],
		})
		// quick path for single output
		if len(m.outputs) == 1 {
			wn, err := m.outputs[0].Write(ctx, readBuffer[:n])
			if err == nil && wn != n {
				m.tunnelWatcher(ForwardConnMessage{
					MessageType: ForwardConnMsgTypeWriteToOutputError,
					Err:         fmt.Errorf("write not match, want %d, got %d", n, wn),
					Output:      m.outputs[0].config,
					OutputAddr:  m.outputs[0].ConnAddr(),
				})
			}
			if err != nil {
				m.tunnelWatcher(ForwardConnMessage{
					MessageType: ForwardConnMsgTypeWriteToOutputError,
					Err:         err,
					Output:      m.outputs[0].config,
					OutputAddr:  m.outputs[0].ConnAddr(),
				})
			} else {
				m.tunnelWatcher(ForwardConnMessage{
					MessageType: ForwardConnMsgTypeWriteToOutputOK,
					Output:      m.outputs[0].config,
					Data:        readBuffer[:n],
					OutputAddr:  m.outputs[0].ConnAddr(),
				})
			}
			continue
		}
		for _, output := range m.outputs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				wn, err := output.Write(ctx, readBuffer[:n])
				if err == nil && wn != n {
					m.tunnelWatcher(ForwardConnMessage{
						MessageType: ForwardConnMsgTypeWriteToOutputError,
						Err:         fmt.Errorf("write not match, want %d, got %d", n, wn),
						Output:      output.config,
						OutputAddr:  output.ConnAddr(),
					})
				}
				if err != nil {
					m.tunnelWatcher(ForwardConnMessage{
						MessageType: ForwardConnMsgTypeWriteToOutputError,
						Err:         err,
						Output:      output.config,
						OutputAddr:  output.ConnAddr(),
					})
				} else {
					m.tunnelWatcher(ForwardConnMessage{
						MessageType: ForwardConnMsgTypeWriteToOutputOK,
						Output:      output.config,
						Data:        readBuffer[:n],
						OutputAddr:  output.ConnAddr(),
					})
				}
			}()
		}
		wg.Wait()
	}
}
