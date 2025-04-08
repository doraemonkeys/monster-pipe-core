package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/doraemonkeys/monster-pipe-core/internal/forwarder"
	"golang.org/x/crypto/ssh"
)

func main() {
	flag.Parse()
	args := flag.Args()
	// for _, arg := range args {
	// 	fmt.Println(arg)
	// }
	if len(args) != 2 {
		PrintUsage()
		return
	}
	inputCmd := args[0]
	outputCmd := args[1]
	var sshClient *ssh.Client
	if parseNeedConnectSSH(inputCmd, outputCmd) {
		var err error
		sshClient, err = parseSSHCmdConfigAndConnectSSH()
		if err != nil {
			log.Fatal(err)
		}
	}
	// fmt.Println("ssh client: ", sshClient)

	input, err := parseInputCmd(inputCmd, sshClient)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("input: %#v\n", input.Config)
	outputs, err := parseNetOutputsConfig(outputCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	var ForwardOutputs []*forwarder.ForwardOutput = make([]*forwarder.ForwardOutput, 0, len(outputs))
	for i, output := range outputs {
		if output.Host == "ssh" && sshClient == nil {
			log.Fatal("ssh client config not found")
		}
		if output.Host == "ssh" && sshClient != nil {
			ForwardOutputs = append(ForwardOutputs,
				forwarder.NewForwardOutput(output, func(_ context.Context, network string, address string) (net.Conn, error) {
					// fmt.Println("sshClient.Dial(network, address): ", network, address)
					return sshClient.Dial(network, address)
				}))
		} else {
			ForwardOutputs = append(ForwardOutputs, forwarder.NewForwardOutput(output, nil))
		}
		fmt.Printf("Output(%d): %#v\n", i, output)
	}
	// i := 0
	f := forwarder.NewForwarder(input, ForwardOutputs, func(message forwarder.ForwardMessage) {
		// fmt.Println("message: ", message)
		// print message
		if *verboseCmd {
			printMessageVerbose(message)
		} else {
			printMessage(message)
		}
	})
	err = f.Run(context.Background())
	if err != nil {
		fmt.Println(err)
	}
}
