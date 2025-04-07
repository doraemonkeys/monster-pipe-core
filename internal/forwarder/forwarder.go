package forwarder

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"

	syncgmap "github.com/doraemonkeys/sync-gmap"
)

type MonsterPipeCoreForwarder struct {
	input            *ForwardInput
	outputs          []*ForwardOutput
	msgWatcher       func(message ForwardMessage)
	connectedClients *syncgmap.SyncMap[string, net.Addr]
}

type ForwardMessageType int

const (
	ForwardMsgTypeAccept      ForwardMessageType = 1
	ForwardMsgTypeTunnel      ForwardMessageType = 2
	ForwardMsgTypeCommonError ForwardMessageType = 3
	ForwardMsgTypeAcceptError ForwardMessageType = 4
)

func (f ForwardMessageType) String() string {
	switch f {
	case ForwardMsgTypeAccept:
		return "Accept connection"
	case ForwardMsgTypeTunnel:
		return "Tunnel message"
	case ForwardMsgTypeCommonError:
		return "Common error"
	}
	return "Unknown"
}

type ForwardMessage struct {
	MessageType ForwardMessageType
	ConnAddr    net.Addr
	ConnBlocked bool
	TunnelMsg   *ForwardConnMessage
	Err         error
}

// func (f ForwardMessage) PrettyPrint() string {
// 	// return fmt.Sprintf("MessageType: %s, ConnAddr: %s, ConnBlocked: %t, TunnelMsg: %v, Err: %v", f.MessageType, f.ConnAddr, f.ConnBlocked, f.TunnelMsg, f.Err)
// 	o := fmt.Sprintf("MessageType: %v", f.MessageType)
// 	if f.ConnBlocked {
// 		o += fmt.Sprintf(", ConnBlocked: %t", f.ConnBlocked)
// 	}
// 	if f.ConnAddr != nil {
// 		o += fmt.Sprintf(", ConnAddr: %s", f.ConnAddr)
// 	}
// 	if f.Err != nil {
// 		o += fmt.Sprintf(", Err: %v", f.Err)
// 	}
// 	if f.TunnelMsg != nil {
// 		o += fmt.Sprintf("\n%s", f.TunnelMsg.PrettyPrint())
// 	}
// 	return o
// }

func NewForwarder(input *ForwardInput, outputs []*ForwardOutput, msgWatcher func(message ForwardMessage)) *MonsterPipeCoreForwarder {
	return &MonsterPipeCoreForwarder{
		input:            input,
		outputs:          outputs,
		msgWatcher:       msgWatcher,
		connectedClients: syncgmap.NewSyncMap[string, net.Addr](),
	}
}

// Concurrent not safe
func (f *MonsterPipeCoreForwarder) AddOutput(output *ForwardOutput) {
	f.outputs = append(f.outputs, output)
}

func (f *MonsterPipeCoreForwarder) Run(ctx context.Context) error {
	listener, err := f.input.Listen(ctx)
	if err != nil {
		return err
	}
	fmt.Println("mpipe forwarder start, listen on input", listener.Addr().String())
	defer listener.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		// fmt.Println("debug: 111")
		conn, err := listener.Accept()
		if err != nil {
			f.msgWatcher(ForwardMessage{
				MessageType: ForwardMsgTypeAcceptError,
				Err:         err,
			})
			continue
		}
		var connBlocked bool
		if !f.input.CheckConn(conn) {
			connBlocked = true
			_ = conn.Close()
		}
		f.msgWatcher(ForwardMessage{
			MessageType: ForwardMsgTypeAccept,
			ConnAddr:    conn.RemoteAddr(),
			ConnBlocked: connBlocked,
		})
		if connBlocked {
			continue
		}
		go f.handleConn(ctx, conn)
	}
}

func (f *MonsterPipeCoreForwarder) handleConn(ctx context.Context, conn net.Conn) {
	connAddr := conn.RemoteAddr()
	f.connectedClients.Store(connAddr.String(), connAddr)
	defer f.connectedClients.Delete(connAddr.String())

	MsgWatcher := func(message ForwardConnMessage) {
		f.msgWatcher(ForwardMessage{
			MessageType: ForwardMsgTypeTunnel,
			ConnAddr:    connAddr,
			TunnelMsg:   &message,
		})
	}
	var outputs []*ForwardOutput = make([]*ForwardOutput, 0, len(f.outputs))
	for _, output := range f.outputs {
		outputs = append(outputs, output.Copy())
	}
	tunnel := NewForwardTunnel(conn, outputs, MsgWatcher)
	tunnel.Run(ctx)
}

func matchAddress(pattern, address string) bool {
	// 分割 IP 和端口
	patternParts := strings.Split(pattern, ":")
	addressParts := strings.Split(address, ":")

	// 检查 IP 部分
	if !matchIP(patternParts[0], addressParts[0]) {
		return false
	}

	if len(patternParts) == 1 {
		// If pattern has no port, then it is considered a match
		return true
	}

	// 检查端口部分
	if len(patternParts) > 1 && len(addressParts) > 1 {
		pattern := strings.Replace(patternParts[1], "*", "[0-9]+", -1)
		pattern = fmt.Sprintf("^%s$", pattern)
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		return regex.MatchString(addressParts[len(addressParts)-1])
	}

	return true
}

func matchIP(pattern, ip string) bool {
	if pattern == "*" {
		return true
	}

	// 处理 IPv4
	if strings.ContainsRune(pattern, '.') {
		return matchIPv4(pattern, ip)
	}

	// 处理 IPv6
	return matchIPv6(pattern, ip)
}

func matchIPv4(pattern, ip string) bool {
	pattern = strings.Replace(pattern, ".", "\\.", -1)
	pattern = fmt.Sprintf("^%s$", strings.Replace(pattern, "*", "[0-9.]+", -1))
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return regex.MatchString(ip)
}

func matchIPv6(pattern, ip string) bool {
	// 移除方括号（如果有）
	pattern = strings.Trim(pattern, "[]")
	ip = strings.Trim(ip, "[]")

	// 展开 IPv6 地址
	patternIP := net.ParseIP(strings.Replace(pattern, "*", "0", -1))
	actualIP := net.ParseIP(ip)

	if patternIP == nil || actualIP == nil {
		return false
	}

	patternParts := strings.Split(pattern, ":")
	actualParts := strings.Split(ip, ":")

	for i := 0; i < len(patternParts); i++ {
		if patternParts[i] == "*" {
			continue
		}
		if patternParts[i] != actualParts[i] {
			return false
		}
	}

	return true
}
