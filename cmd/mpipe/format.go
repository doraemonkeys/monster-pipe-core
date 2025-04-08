package main

import (
	"fmt"
	"time"

	"github.com/doraemonkeys/monster-pipe-core/internal/forwarder"
	"github.com/fatih/color"
)

var (
	green   = color.New(color.FgGreen).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
)

func formatData(data []byte) string {
	if len(data) <= 100 {
		return fmt.Sprintf("%q", data)
	}
	return fmt.Sprintf("%q ... %q", data[:50], data[len(data)-50:])
}

func printMessage(message forwarder.ForwardMessage) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	switch message.MessageType {
	case forwarder.ForwardMsgTypeAccept:
		fmt.Printf("[%s] %s: %s\n",
			green(timestamp),
			green("Connection Accepted"),
			blue(message.ConnAddr.String()),
		)
		if message.ConnBlocked {
			fmt.Printf("  %s\n", red("Connection is blocked by rules"))
		}
	case forwarder.ForwardMsgTypeAcceptError:
		fmt.Printf("[%s] %s: %s\n",
			red(timestamp),
			red("Connection Accepted Error"),
			red(message.Err),
		)
	case forwarder.ForwardMsgTypeTunnel:
		if message.TunnelMsg != nil {
			tunnelMsg := message.TunnelMsg
			switch tunnelMsg.MessageType {
			case forwarder.ForwardConnMsgTypeInputRead:
			case forwarder.ForwardConnMsgTypeInputReadError:
				fmt.Printf("[%s] %s: %s | %s\n",
					red(timestamp),
					red("Read from input Error"),
					blue(message.ConnAddr.String()),
					red(tunnelMsg.Err),
				)
			case forwarder.ForwardConnMsgTypeWriteToInputError:
				fmt.Printf("[%s] %s: %s -> %s | %s\n",
					red(timestamp),
					red("Write to input Error"),
					blue(message.ConnAddr.String()),
					yellow(tunnelMsg.Address()),
					red(tunnelMsg.Err),
				)
			case forwarder.ForwardConnMsgTypeWriteToInputOK:
			case forwarder.ForwardConnMsgTypeOutputRead:
			case forwarder.ForwardConnMsgTypeWriteToOutputOK:
			case forwarder.ForwardConnMsgTypeWriteToOutputError:
				fmt.Printf("[%s] %s: %s <- %s | %s\n",
					red(timestamp),
					red("Write to output Error"),
					blue(message.ConnAddr.String()),
					yellow(tunnelMsg.Address()),
					red(tunnelMsg.Err),
				)
			case forwarder.ForwardConnMsgTypeOutputReadError:
				fmt.Printf("[%s] %s: %s <- %s | %s\n",
					red(timestamp),
					red("Read from output Error"),
					blue(message.ConnAddr.String()),
					yellow(tunnelMsg.Address()),
					red(tunnelMsg.Err),
				)
			case forwarder.ForwardConnMsgTypeTunnelClosed:
				fmt.Printf("[%s] %s : %s by %s\n",
					yellow(timestamp),
					yellow("Tunnel closed"),
					blue(message.ConnAddr.String()),
					closedBy(message.TunnelMsg.ClosedByOutput),
				)
			}
		}
	case forwarder.ForwardMsgTypeCommonError:
		fmt.Printf("[%s] %s: %s\n",
			red(timestamp),
			red("Common Error"),
			red(message.Err),
		)
	}
}

func printMessageVerbose(message forwarder.ForwardMessage) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	switch message.MessageType {
	case forwarder.ForwardMsgTypeAccept:
		fmt.Printf("[%s] %s: %s\n",
			green(timestamp),
			green("Connection Accepted"),
			blue(message.ConnAddr.String()),
		)
		if message.ConnBlocked {
			fmt.Printf("  %s\n", red("Connection is blocked by rules"))
		}
	case forwarder.ForwardMsgTypeTunnel:
		if message.TunnelMsg != nil {
			tunnelMsg := message.TunnelMsg
			switch tunnelMsg.MessageType {
			case forwarder.ForwardConnMsgTypeInputRead:
				fmt.Printf("[%s] %s: %s | %s\n",
					cyan(timestamp),
					cyan("Read from input"),
					blue(message.ConnAddr.String()),
					formatData(tunnelMsg.Data),
				)
			case forwarder.ForwardConnMsgTypeInputReadError:
				fmt.Printf("[%s] %s: %s | %s\n",
					red(timestamp),
					red("Read from input Error"),
					blue(message.ConnAddr.String()),
					red(tunnelMsg.Err),
				)
			case forwarder.ForwardConnMsgTypeWriteToInputError:
				fmt.Printf("[%s] %s: %s -> %s | %s\n",
					red(timestamp),
					red("Write to input Error"),
					blue(message.ConnAddr.String()),
					yellow(tunnelMsg.Address()),
					red(tunnelMsg.Err),
				)
			case forwarder.ForwardConnMsgTypeWriteToInputOK:
				fmt.Printf("[%s] %s: %s <- %s | %d bytes\n",
					cyan(timestamp),
					cyan("Write to input OK"),
					blue(message.ConnAddr.String()),
					yellow(tunnelMsg.Address()),
					len(tunnelMsg.Data),
				)
			case forwarder.ForwardConnMsgTypeOutputRead:
				fmt.Printf("[%s] %s: %s | %s\n",
					magenta(timestamp),
					magenta("Read from output"),
					yellow(tunnelMsg.Address()),
					formatData(tunnelMsg.Data),
				)
			case forwarder.ForwardConnMsgTypeWriteToOutputOK:
				fmt.Printf("[%s] %s: %s -> %s | %d bytes\n",
					magenta(timestamp),
					magenta("Write to output OK"),
					blue(message.ConnAddr.String()),
					yellow(tunnelMsg.Address()),
					len(tunnelMsg.Data),
				)
			case forwarder.ForwardConnMsgTypeWriteToOutputError:
				fmt.Printf("[%s] %s: %s <- %s | %s\n",
					red(timestamp),
					red("Write to output Error"),
					blue(message.ConnAddr.String()),
					yellow(tunnelMsg.Address()),
					red(tunnelMsg.Err),
				)
			case forwarder.ForwardConnMsgTypeOutputReadError:
				fmt.Printf("[%s] %s: %s <- %s | %s\n",
					red(timestamp),
					red("Read from output Error"),
					blue(message.ConnAddr.String()),
					yellow(tunnelMsg.Address()),
					red(tunnelMsg.Err),
				)
			case forwarder.ForwardConnMsgTypeTunnelClosed:
				fmt.Printf("[%s] %s : %s by %s\n",
					yellow(timestamp),
					yellow("Tunnel closed"),
					blue(message.ConnAddr.String()),
					closedBy(message.TunnelMsg.ClosedByOutput),
				)
			}
		}
	case forwarder.ForwardMsgTypeCommonError:
		fmt.Printf("[%s] %s: %s\n",
			red(timestamp),
			red("Common Error"),
			red(message.Err),
		)
	}
}

func closedBy(byOutput bool) string {
	if byOutput {
		return yellow("output")
	}
	return blue("input")
}
