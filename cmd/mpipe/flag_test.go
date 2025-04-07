package main

import (
	"testing"

	"github.com/doraemonkeys/monster-pipe-core/internal/forwarder"
	"github.com/doraemonkeys/monster-pipe-core/pkg/protocol"
)

func Test_parseNetInput(t *testing.T) {

	tests := []struct {
		name    string
		input   string
		want    *forwarder.ForwardInputConfig
		wantErr bool
	}{
		{"1", "192.168.1.1:6789", &forwarder.ForwardInputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP,
		}, false},
		{"2", "192.168.1.1:6789@tcp", &forwarder.ForwardInputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP,
		}, false},
		{"3", "192.168.1.1:6789@udp", &forwarder.ForwardInputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolUDP,
		}, false},
		{"4", "192.168.1.1:6789@udp@1234", &forwarder.ForwardInputConfig{}, true},
		{"5", "192.168.1.1:6789@tcp4", &forwarder.ForwardInputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP4,
		}, false},
		{"6", "192.168.1.1:6789@tcp6", &forwarder.ForwardInputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP6,
		}, false},
		{"7", ":6789@udp", &forwarder.ForwardInputConfig{
			Host:     "",
			Port:     6789,
			Protocol: protocol.NetProtocolUDP,
		}, false},
		{"8", "ssh:6789", &forwarder.ForwardInputConfig{
			Host:     "",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP,
		}, false},
		{"9", "ssh:6789@tcp", &forwarder.ForwardInputConfig{
			Host:     "",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP,
		}, false},
		{"10", "ssh:127.0.0.1:6789", &forwarder.ForwardInputConfig{
			Host:     "127.0.0.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP,
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNetInputConfig(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNetInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got.Host != tt.want.Host || got.Port != tt.want.Port || got.Protocol != tt.want.Protocol {
				t.Errorf("parseNetInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseNetOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    *forwarder.ForwardOutputConfig
		wantErr bool
	}{
		{"1", "192.168.1.1:6789", &forwarder.ForwardOutputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP,
			Readable: true,
			Writable: true,
		}, false},
		{"2", "192.168.1.1:6789@tcp", &forwarder.ForwardOutputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP,
			Readable: true,
			Writable: true,
		}, false},
		{"3", "192.168.1.1:6789@udp", &forwarder.ForwardOutputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolUDP,
			Readable: true,
			Writable: true,
		}, false},
		{"4", "192.168.1.1:6789@udp@1234", &forwarder.ForwardOutputConfig{}, true},
		{"5", "192.168.1.1:6789@tcp4", &forwarder.ForwardOutputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP4,
			Readable: true,
			Writable: true,
		}, false},
		{"6", "192.168.1.1:6789@tcp6", &forwarder.ForwardOutputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP6,
			Readable: true,
			Writable: true,
		}, false},
		{"7", "192.168.1.1:6789@tcp=1234", &forwarder.ForwardOutputConfig{}, true},
		{"8", "192.168.1.1:6789@tcp<", &forwarder.ForwardOutputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP,
			Readable: false,
			Writable: true,
		}, false},
		{"9", "192.168.1.1:6789@tcp>", &forwarder.ForwardOutputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP,
			Readable: true,
			Writable: false,
		}, false},
		{"10", "192.168.1.1:6789@tcp=", &forwarder.ForwardOutputConfig{
			Host:     "192.168.1.1",
			Port:     6789,
			Protocol: protocol.NetProtocolTCP,
			Readable: true,
			Writable: true,
		}, false},
		{"11", "ssh:7890@tcp", &forwarder.ForwardOutputConfig{
			Host:     "", //dial
			Port:     7890,
			Protocol: protocol.NetProtocolTCP,
			Readable: true,
			Writable: true,
		}, false},
		{"12", "ssh:127.0.0.1:7890@tcp", &forwarder.ForwardOutputConfig{
			Host:     "127.0.0.1",
			Port:     7890,
			Protocol: protocol.NetProtocolTCP,
			Readable: true,
			Writable: true,
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNetOutputConfig(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNetOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got.Host != tt.want.Host || got.Port != tt.want.Port || got.Protocol != tt.want.Protocol {
				t.Errorf("parseNetOutput() = %v, want %v", got, tt.want)
			}
			if got.Readable != tt.want.Readable || got.Writable != tt.want.Writable {
				t.Errorf("parseNetOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}
