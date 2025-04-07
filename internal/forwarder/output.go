package forwarder

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/doraemonkeys/monster-pipe-core/pkg/protocol"
	"golang.org/x/sync/singleflight"
)

type ForwardOutputConfig struct {
	Readable bool
	Writable bool
	Host     string
	Port     int
	Protocol protocol.NetProtocol
}

func (f ForwardOutputConfig) Target() string {
	return f.Host + ":" + strconv.Itoa(f.Port)
}

type ForwardOutput struct {
	config              ForwardOutputConfig
	conn                net.Conn
	connectSingleflight singleflight.Group
	dialer              func(ctx context.Context, network string, address string) (net.Conn, error)
}

func NewForwardOutput(config ForwardOutputConfig, dialer func(ctx context.Context, network string, address string) (net.Conn, error)) *ForwardOutput {
	if dialer == nil {
		dialer = func(ctx context.Context, network string, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, network, address)
		}
	}
	return &ForwardOutput{
		config: config,
		dialer: dialer,
	}
}

func (f *ForwardOutput) GetConfig() ForwardOutputConfig {
	return f.config
}

func (f *ForwardOutput) Copy() *ForwardOutput {
	var newOutput ForwardOutput
	newOutput.config = f.config
	newOutput.dialer = f.dialer
	return &newOutput
}

func (f *ForwardOutput) Write(ctx context.Context, buf []byte) (int, error) {
	if !f.config.Writable {
		return 0, nil
	}
	if f.conn == nil {
		if err := f.Dial(ctx); err != nil {
			return 0, fmt.Errorf("dail output error: %w", err)
		}
	}
	n, err := f.conn.Write(buf)
	if err != nil {
		// f.conn = nil
		return 0, fmt.Errorf("write to output error: %w", err)
	}
	return n, nil
}

func (f *ForwardOutput) Read(ctx context.Context, buf []byte) (int, error) {
	if f.conn == nil {
		if err := f.Dial(ctx); err != nil {
			return 0, fmt.Errorf("dail output error: %w", err)
		}
	}
	n, err := f.conn.Read(buf)
	if err != nil {
		// f.conn = nil
		return 0, fmt.Errorf("read from output error: %w", err)
	}
	return n, nil
}

func (f *ForwardOutput) Close() error {
	if f.conn == nil {
		return nil
	}
	return f.conn.Close()
}

func (f *ForwardOutput) Dial(ctx context.Context) error {
	_, err, _ := f.connectSingleflight.Do("", func() (interface{}, error) {
		host := f.config.Host
		if len(host) == 0 {
			host = "127.0.0.1"
		}
		conn, err := f.dialer(ctx, string(f.config.Protocol), host+":"+strconv.Itoa(f.config.Port))
		if err != nil {
			return nil, err
		}
		f.conn = conn
		return nil, nil
	})
	return err
}

func (f *ForwardOutput) Target() string {
	if f.conn == nil {
		return f.config.Host + ":" + strconv.Itoa(f.config.Port)
	}
	return f.conn.RemoteAddr().String()
}

func (f *ForwardOutput) ConnAddr() net.Addr {
	if f.conn == nil {
		return nil
	}
	addr := f.conn.RemoteAddr()
	addrString := addr.String()
	if strings.HasSuffix(addrString, ":0") {
		return nil
	}
	return addr
}
