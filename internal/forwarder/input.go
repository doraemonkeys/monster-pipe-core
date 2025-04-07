package forwarder

import (
	"context"
	"net"
	"strconv"

	"github.com/Monster-Pipe/monster-pipe-core/pkg/protocol"
	syncgmap "github.com/doraemonkeys/sync-gmap"
)

type ForwardInputConfig struct {
	Host     string
	Port     int
	Protocol protocol.NetProtocol
	// Blacklist and Whitelist can only exist one, the other is nil
	//
	// for example, "192.0.2.1:25", "[2001:db8::1]:80" , "192.0.2.1:*"
	Blacklist []MatchHostConfig
	// Blacklist and Whitelist can only exist one, the other is nil.
	//
	// for example, "192.0.2.1:25", "[2001:db8::1]:80" , "192.0.2.1:*"
	Whitelist []MatchHostConfig
}

type MatchHostConfig struct {
	Match    string
	AnyProto bool
	Protocol protocol.NetProtocol
}

type ForwardInput struct {
	Config   ForwardInputConfig
	listener func(ctx context.Context, network string, address string) (net.Listener, error)
}

func NewForwardInput(config ForwardInputConfig, listener func(ctx context.Context, network string, address string) (net.Listener, error)) *ForwardInput {
	if listener == nil {
		listener = func(ctx context.Context, network string, address string) (net.Listener, error) {
			// l := net.ListenConfig{}
			// l.ListenPacket()
			// return l.Listen(ctx, network, address)
			switch network {
			case "udp", "udp4", "udp6":
				udpAddr, err := net.ResolveUDPAddr(network, address)
				if err != nil {
					return nil, err
				}
				udpConn, err := net.ListenUDP(network, udpAddr)
				if err != nil {
					return nil, err
				}
				return &UdpListener{listener: udpConn, dataChMap: syncgmap.NewSyncMap[string, chan []byte]()}, nil
			default:
				l, err := net.Listen(network, address)
				if err != nil {
					return nil, err
				}
				return l, nil
			}
		}
	}
	return &ForwardInput{
		Config:   config,
		listener: listener,
	}
}

// Check if the connection is allowed
func (f *ForwardInput) CheckConn(conn net.Conn) bool {
	if len(f.Config.Blacklist) > 0 {
		for _, addr := range f.Config.Blacklist {
			if matchAddress(addr.Match, conn.RemoteAddr().String()) {
				if addr.AnyProto {
					return false
				} else if addr.Protocol != f.Config.Protocol {
					return false
				}
			}
		}
		return true
	}
	if len(f.Config.Whitelist) > 0 {
		for _, addr := range f.Config.Whitelist {
			if matchAddress(addr.Match, conn.RemoteAddr().String()) {
				if addr.AnyProto {
					return true
				} else if addr.Protocol == f.Config.Protocol {
					return true
				}
			}
		}
		return false
	}
	return true
}

func (f *ForwardInput) Listen(ctx context.Context) (net.Listener, error) {
	// fmt.Printf("f.config: %+v\n", f.config)
	return f.listener(ctx, f.Config.Protocol.String(), f.Config.Host+":"+strconv.Itoa(f.Config.Port))
}
