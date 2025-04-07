package protocol

import (
	"fmt"
	"strings"
)

type NetProtocol string

const (
	NetProtocolTCP  NetProtocol = "tcp"
	NetProtocolTCP4 NetProtocol = "tcp4"
	NetProtocolTCP6 NetProtocol = "tcp6"
	NetProtocolUDP  NetProtocol = "udp"
	NetProtocolUDP4 NetProtocol = "udp4"
	NetProtocolUDP6 NetProtocol = "udp6"
)

func (n NetProtocol) String() string {
	return string(n)
}

func ParseNetProtocol(protocol string) (NetProtocol, error) {
	switch strings.ToLower(protocol) {
	case "tcp":
		return NetProtocolTCP, nil
	case "tcp4":
		return NetProtocolTCP4, nil
	case "tcp6":
		return NetProtocolTCP6, nil
	case "udp":
		return NetProtocolUDP, nil
	case "udp4":
		return NetProtocolUDP4, nil
	case "udp6":
		return NetProtocolUDP6, nil
	}
	return "", fmt.Errorf("invalid protocol: %s", protocol)
}
