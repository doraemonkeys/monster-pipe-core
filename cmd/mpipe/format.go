package main

import (
	"fmt"
	"net"
	"strconv"
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
	white   = color.New(color.FgWhite).SprintFunc()
)

func formatData(data []byte) string {
	if len(data) <= 100 {
		return fmt.Sprintf("%q", data)
	}
	return fmt.Sprintf("%q ... %q", data[:50], data[len(data)-50:])
}

// Helper for conditional printing inline
func iif(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}

// Simplified printMessage and printMessageVerbose based on your original code
func printMessage(message forwarder.ForwardMessage) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	switch message.MessageType {
	case forwarder.ForwardMsgTypeAccept:
		fmt.Printf("[%s] %s: %s %s\n",
			green(timestamp),
			green("Connection Accepted"),
			blue(message.ConnAddr.String()),
			iif(message.ConnBlocked, red("(Blocked)"), ""),
		)
	case forwarder.ForwardMsgTypeAcceptError:
		fmt.Printf("[%s] %s: %s\n", red(timestamp), red("Connection Accepted Error"), red(message.Err))
	case forwarder.ForwardMsgTypeTunnel:
		if message.TunnelMsg != nil {
			tunnelMsg := message.TunnelMsg
			switch tunnelMsg.MessageType {
			case forwarder.ForwardConnMsgTypeInputReadError:
				fmt.Printf("[%s] %s: %s | %s\n", red(timestamp), red("Read <- Input Error"), blue(message.ConnAddr.String()), red(tunnelMsg.Err))
			case forwarder.ForwardConnMsgTypeWriteToInputError:
				fmt.Printf("[%s] %s: %s -> %s | %s\n", red(timestamp), red("Write -> Input Error"), blue(message.ConnAddr.String()), yellow(tunnelMsg.Address()), red(tunnelMsg.Err))
			case forwarder.ForwardConnMsgTypeWriteToOutputError:
				fmt.Printf("[%s] %s: %s <- %s | %s\n", red(timestamp), red("Write -> Output Error"), blue(message.ConnAddr.String()), yellow(tunnelMsg.Address()), red(tunnelMsg.Err))
			case forwarder.ForwardConnMsgTypeOutputReadError:
				fmt.Printf("[%s] %s: %s <- %s | %s\n", red(timestamp), red("Read <- Output Error"), blue(message.ConnAddr.String()), yellow(tunnelMsg.Address()), red(tunnelMsg.Err))
			case forwarder.ForwardConnMsgTypeTunnelClosed:
				fmt.Printf("[%s] %s : %s by %s\n", yellow(timestamp), yellow("Tunnel Closed"), blue(message.ConnAddr.String()), closedBy(message.TunnelMsg.ClosedByOutput))
			}
		}
	case forwarder.ForwardMsgTypeCommonError:
		fmt.Printf("[%s] %s: %s\n", red(timestamp), red("Error"), red(message.Err))
	}
}

func printMessageVerbose(message forwarder.ForwardMessage) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	switch message.MessageType {
	case forwarder.ForwardMsgTypeAccept:
		fmt.Printf("[%s] %s: %s %s\n",
			green(timestamp),
			green("Connection Accepted"),
			blue(message.ConnAddr.String()),
			iif(message.ConnBlocked, red("(Blocked by rules)"), ""),
		)
	case forwarder.ForwardMsgTypeAcceptError:
		fmt.Printf("[%s] %s: %s\n",
			red(timestamp),
			red("Connection Accept Error"),
			red(message.Err),
		)
	case forwarder.ForwardMsgTypeTunnel:
		if message.TunnelMsg != nil {
			tunnelMsg := message.TunnelMsg
			connAddrStr := blue(message.ConnAddr.String())
			outputAddrStr := yellow(tunnelMsg.Address())

			switch tunnelMsg.MessageType {
			case forwarder.ForwardConnMsgTypeInputRead:
				fmt.Printf("[%s] %s: %s | %d bytes | Data: %s\n",
					cyan(timestamp), cyan("Read <- Input"), connAddrStr, len(tunnelMsg.Data), formatData(tunnelMsg.Data))
			case forwarder.ForwardConnMsgTypeInputReadError:
				fmt.Printf("[%s] %s: %s | %s\n",
					red(timestamp), red("Read <- Input Error"), connAddrStr, red(tunnelMsg.Err))
			case forwarder.ForwardConnMsgTypeWriteToInputOK:
				fmt.Printf("[%s] %s: %s <- %s | %d bytes\n",
					cyan(timestamp), cyan("Write -> Input OK"), connAddrStr, outputAddrStr, len(tunnelMsg.Data))
			case forwarder.ForwardConnMsgTypeWriteToInputError:
				fmt.Printf("[%s] %s: %s <- %s | %s\n",
					red(timestamp), red("Write -> Input Error"), connAddrStr, outputAddrStr, red(tunnelMsg.Err))
			case forwarder.ForwardConnMsgTypeOutputRead:
				fmt.Printf("[%s] %s: %s <- %s | %d bytes | Data: %s\n",
					magenta(timestamp), magenta("Read <- Output"), connAddrStr, outputAddrStr, len(tunnelMsg.Data), formatData(tunnelMsg.Data))
			case forwarder.ForwardConnMsgTypeOutputReadError:
				fmt.Printf("[%s] %s: %s <- %s | %s\n",
					red(timestamp), red("Read <- Output Error"), connAddrStr, outputAddrStr, red(tunnelMsg.Err))
			case forwarder.ForwardConnMsgTypeWriteToOutputOK:
				fmt.Printf("[%s] %s: %s -> %s | %d bytes\n",
					magenta(timestamp), magenta("Write -> Output OK"), connAddrStr, outputAddrStr, len(tunnelMsg.Data))
			case forwarder.ForwardConnMsgTypeWriteToOutputError:
				fmt.Printf("[%s] %s: %s -> %s | %s\n",
					red(timestamp), red("Write -> Output Error"), connAddrStr, outputAddrStr, red(tunnelMsg.Err))
			case forwarder.ForwardConnMsgTypeTunnelClosed:
				fmt.Printf("[%s] %s : %s by %s\n",
					yellow(timestamp), yellow("Tunnel Closed"), connAddrStr, closedBy(message.TunnelMsg.ClosedByOutput))
			}
		}
	case forwarder.ForwardMsgTypeCommonError:
		fmt.Printf("[%s] %s: %s\n",
			red(timestamp), red("Common Error"), red(message.Err))
	}
}

func closedBy(byOutput bool) string {
	if byOutput {
		return yellow("output")
	}
	return blue("input")
}

// prettyPrintConfig formats and prints the input and output configurations.
// It assumes ForwardInputConfig and ForwardOutputConfig have fields like
// Host, Port, Protocol, Blacklist, Whitelist, Readable, Writable.
func prettyPrintConfig(input forwarder.ForwardInputConfig, outputs []forwarder.ForwardOutputConfig) {
	fmt.Println(green("--- Configuration Summary ---"))

	// --- Input Configuration ---
	fmt.Printf("%s\n", yellow("Input (Listen):"))

	listenHost := input.Host
	if listenHost == "" {
		listenHost = "*" // Represent listening on all interfaces
	}
	listenAddr := net.JoinHostPort(listenHost, strconv.Itoa(input.Port))
	fmt.Printf("  %-12s %s (%s)\n", white("Address:"), blue(listenAddr), cyan(input.Protocol.String()))

	// Print Blacklist/Whitelist if they exist and are non-empty
	// Assuming Blacklist/Whitelist are slices of a type with a String() method, or just []string
	printAccessList := func(label string, list []forwarder.MatchHostConfig) { // Adjust MatchHostConfig type if needed
		if len(list) > 0 {
			fmt.Printf("  %-12s\n", white(label+":"))
			for _, item := range list {
				// Assuming item has String() or is a string itself
				fmt.Printf("    - %s\n", magenta(fmt.Sprintf("%v", item))) // Use %v as a fallback
			}
		} else {
			// Optional: Print "None" if list is empty
			// fmt.Printf("  %-12s %s\n", white(label+":"), yellow("None"))
		}
	}

	printAccessList("Blacklist", input.Blacklist)
	printAccessList("Whitelist", input.Whitelist)

	// --- Output Configuration ---
	fmt.Printf("\n%s\n", yellow("Outputs (Forward To):"))
	if len(outputs) == 0 {
		fmt.Printf("  %s\n", red("No output destinations configured!"))
		return
	}

	for i, output := range outputs {

		targetHost := output.Host
		targetDesc := ""
		if targetHost == "ssh" {
			targetHost = "localhost (via SSH)" // Clarify SSH forwarding
			targetDesc = yellow(" (SSH Tunnel)")
		}
		targetAddr := net.JoinHostPort(targetHost, strconv.Itoa(output.Port))

		// Use Target() method if available and preferred, otherwise build manually
		// targetAddrStr := output.Target() // If Target() exists and gives the desired string
		targetAddrStr := blue(targetAddr) // Use manually built string otherwise

		fmt.Printf("  %s %d:%s\n", white("Output"), i+1, targetDesc)
		fmt.Printf("    %-10s %s (%s)\n", white("Target:"), targetAddrStr, cyan(output.Protocol.String()))

		readableStr := iif(output.Readable, green("Yes"), red("No"))
		writableStr := iif(output.Writable, green("Yes"), red("No"))
		fmt.Printf("    %-10s %s\n", white("Readable:"), readableStr)
		fmt.Printf("    %-10s %s\n", white("Writable:"), writableStr)
	}

	fmt.Println(green("-----------------------------")) // Footer
}
