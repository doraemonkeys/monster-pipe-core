package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/doraemonkeys/monster-pipe-core/internal/forwarder"
	"github.com/doraemonkeys/monster-pipe-core/pkg/protocol"
	"github.com/fatih/color"
	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var (
	sshHostCmd       *string = flag.String("ssh", "", "ssh host")
	verboseCmd       *bool   = flag.Bool("verbose", false, "verbose")
	sshIdentityCmd   *string = flag.String("ssh-i", "", "ssh identity file")
	sshConfigFileCmd *string = flag.String("ssh-config", "", "ssh config file")
	sshPortCmd       *int    = flag.Int("ssh-p", 0, "ssh port")
	sshPwdFileCmd    *string = flag.String("ssh-pwd-file", "", "ssh password file")
)

type SSHConfig struct {
	Host         string
	Port         int
	User         string
	IdentityFile string
	Password     string
	Passphrase   string
}

func parseNeedConnectSSH(inputStr string, outputsStr string) bool {
	if strings.HasPrefix(inputStr, "ssh:") {
		return true
	}
	outputsStr = strings.TrimSpace(outputsStr)
	parts := strings.Split(outputsStr, ",")
	for _, part := range parts {
		if strings.HasPrefix(part, "ssh:") {
			return true
		}
	}
	return false
}

func parseSSHCmdConfig() (SSHConfig, error) {
	sshHost := strings.TrimSpace(*sshHostCmd)
	port, err := parseSSHPort(sshHost)
	if err != nil {
		return SSHConfig{}, err
	}
	// fmt.Println("sshHost: ", sshHost, port)
	sshHost = strings.Split(sshHost, ":")[0]
	if sshHost == "" {
		return SSHConfig{}, fmt.Errorf("invalid ssh host: %s", sshHost)
	}
	parts := strings.Split(sshHost, "@")
	var (
		user      string
		host      string
		hostAlias string
	)
	if len(parts) == 1 {
		hostAlias = parts[0]
	} else {
		user = parts[0]
		host = parts[1]
	}
	cfg, err := parseSSHConfigFile(*sshConfigFileCmd)
	if err != nil {
		return SSHConfig{}, err
	}

	// identityFile, identityFileErr := cfg.Get(host, "IdentityFile")
	// password, passwordErr := cfg.Get(host, "Password")
	// hostname, hostnameErr := cfg.Get(host, "Hostname")
	var (
		identityFile string
		password     string
	)
	if hostAlias != "" {
		identityFile, _ = cfg.Get(hostAlias, "IdentityFile")
		password, _ = cfg.Get(hostAlias, "Password")
		host, _ = cfg.Get(hostAlias, "Hostname")
		user, _ = cfg.Get(hostAlias, "User")
	}
	if *sshPwdFileCmd != "" {
		passwordBytes, err := os.ReadFile(*sshPwdFileCmd)
		if err != nil {
			return SSHConfig{}, fmt.Errorf("failed to read password file: %s", err)
		}
		password = string(passwordBytes)
	}
	if password == "" && identityFile == "" {
		fmt.Printf("Enter password for %s:", host)
		pwd, err := readline.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return SSHConfig{}, fmt.Errorf("failed to input password: %s", err)
		}
		password = string(pwd)
	}
	// password = ""
	// fmt.Println("identityFile: ", identityFile, identityFileErr)
	// fmt.Println("password: ", password, passwordErr)
	if port == 0 {
		if cfg != nil {
			portStr, _ := cfg.Get(hostAlias, "Port")
			port, _ = strconv.Atoi(portStr)
			// fmt.Println("port: ", port)
		}
		if port == 0 {
			port = *sshPortCmd
		}
		if port == 0 {
			port = 22
		}
	}
	if sshIdentityCmd != nil && *sshIdentityCmd != "" {
		identityFile = *sshIdentityCmd
	}
	return SSHConfig{
		Host:         host,
		Port:         port,
		User:         user,
		IdentityFile: identityFile,
		Password:     password,
	}, nil
}

func parseSSHConfigFile(sshConfigFile string) (*ssh_config.Config, error) {
	if sshConfigFile == "" {
		sshConfigFile = GetDefaultSSHConfigPath()
	}
	if _, err := os.Stat(sshConfigFile); err != nil {
		return nil, fmt.Errorf("ssh config file not found: %s", sshConfigFile)
	}
	sshConfigFileReader, err := os.Open(sshConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open ssh config file: %s", err)
	}
	cfg, err := ssh_config.Decode(sshConfigFileReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ssh config file: %s", err)
	}
	return cfg, nil
}

// user@host:port
func parseSSHPort(sshHost string) (int, error) {
	if strings.Contains(sshHost, ":") {
		parts := strings.Split(sshHost, ":")
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return 22, fmt.Errorf("invalid ssh port: %s", parts[1])
		}
		return port, nil
	}
	return 0, nil
}

// host:port@protocol
func parseNetInputConfig(addr string) (*forwarder.ForwardInputConfig, error) {
	netAddrConfig, err := parseNetAddrConfig(addr, true)
	if err != nil {
		return nil, err
	}
	return &forwarder.ForwardInputConfig{
		NetAddrConfig: *netAddrConfig,
	}, nil
}

// host:port@protocol
func parseNetAddrConfig(addr string, isInput bool) (*forwarder.NetAddrConfig, error) {
	var pt string = "tcp"
	parts := strings.Split(addr, "@")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid addr format: %s", addr)
	} else if len(parts) == 2 {
		pt = strings.ToLower(parts[1])
	}

	// Parse address and protocol
	address := parts[0]

	// Split host and port
	hostPort := strings.Split(address, ":")
	host := hostPort[0]
	if host == "ssh" {
		switch len(hostPort) {
		case 2:
			// ssh:6789
			host = ""
		case 3:
			// ssh:127.0.0.1:6789
			host = hostPort[1]
		}
	}

	// Parse port
	port, err := strconv.Atoi(hostPort[len(hostPort)-1])
	if err != nil {
		return nil, fmt.Errorf("invalid port number: %s", hostPort[len(hostPort)-1])
	}
	protocol2, err := protocol.ParseNetProtocol(pt)
	if err != nil {
		return nil, fmt.Errorf("invalid protocol: %s", pt)
	}
	if host == "" && !isInput {
		host = "localhost"
	}
	return &forwarder.NetAddrConfig{
		Host:     host,
		Port:     port,
		Protocol: protocol2,
	}, nil
}

// host:port@protocol=
func parseNetOutputConfig(output string) (*forwarder.ForwardOutputConfig, error) {
	output = strings.TrimSpace(output)

	var cfg forwarder.ForwardOutputConfig
	cfg.Readable = true
	cfg.Writable = true
	hasSuffix := strings.HasSuffix(output, "<") || strings.HasSuffix(output, ">") || strings.HasSuffix(output, "=")
	switch {
	case strings.HasSuffix(output, "<"):
		cfg.Readable = false
	case strings.HasSuffix(output, ">"):
		cfg.Writable = false
	case strings.HasSuffix(output, "="):
		cfg.Readable = true
		cfg.Writable = true
	}
	if hasSuffix {
		output = output[:len(output)-1]
	}
	input, err := parseNetAddrConfig(output, false)
	if err != nil {
		return nil, err
	}
	cfg.Host = input.Host
	cfg.Port = input.Port
	cfg.Protocol = input.Protocol
	// return &cfg, nil
	return &cfg, nil
}

func parseNetOutputsConfig(outputs string) ([]forwarder.ForwardOutputConfig, error) {
	outputs = strings.TrimSpace(outputs)
	parts := strings.Split(outputs, ",")
	cfgs := make([]forwarder.ForwardOutputConfig, 0, len(parts))
	for _, output := range parts {
		cfg, err := parseNetOutputConfig(output)
		if err != nil {
			return nil, err
		}
		cfgs = append(cfgs, *cfg)
	}
	return cfgs, nil
}

func GetDefaultSSHConfigPath() string {
	// if $HOME/.ssh/config exists, return it
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".ssh", "config")); err == nil {
		return filepath.Join(os.Getenv("HOME"), ".ssh", "config")
	}
	// if $HOME/.ssh/config.d/config exists, return it
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".ssh", "config.d", "config")); err == nil {
		return filepath.Join(os.Getenv("HOME"), ".ssh", "config.d", "config")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

func parseInputCmd(inputCmd string, sshClient *ssh.Client) (*forwarder.ForwardInput, error) {
	cfg, err := parseNetInputConfig(inputCmd)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(inputCmd, "ssh:") {
		input := forwarder.NewForwardInput(*cfg, nil)
		return input, nil
	}
	input := forwarder.NewForwardInput(*cfg, func(_ context.Context, network string, address string) (net.Listener, error) {
		// fmt.Println("sshClient.Listen(network, address): ", network, address)
		return sshClient.Listen(network, address)
	})
	return input, nil
}

func parseSSHCmdConfigAndConnectSSH() (*ssh.Client, error) {
	cfg, err := parseSSHCmdConfig()
	if err != nil {
		return nil, err
	}
	// fmt.Println("cfg: ", cfg)
	sshClient, err := connectSSH(cfg)
	if err != nil {
		return nil, err
	}
	return sshClient, nil
}

func parseKnownHostsFilePath() (string, error) {
	sshConfigFile := *sshConfigFileCmd
	if sshConfigFile == "" {
		sshConfigFile = GetDefaultSSHConfigPath()
	}
	knownHostsFile := filepath.Join(filepath.Dir(sshConfigFile), "known_hosts")
	if _, err := os.Stat(knownHostsFile); err != nil {
		return "", fmt.Errorf("known hosts file not found: %s", knownHostsFile)
	}
	return knownHostsFile, nil
}

func loadPrivateKey(file string) (ssh.Signer, error) {
	key, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key file: %w", err)
	}
	// 1. Try to parse without passphrase
	signer, err := ssh.ParsePrivateKey(key)
	if err == nil {
		return signer, nil
	}
	// 2. Check if it's a passphrase related error
	if _, ok := err.(*ssh.PassphraseMissingError); ok {
		const maxTry = 3
		var passphrase []byte
		for i := 0; i < maxTry; i++ {
			fmt.Println()
			// 3. Prompt user for passphrase
			passphrase, err = promptPassphrase()
			if err != nil {
				fmt.Println("failed to read passphrase: %w", err)
				continue
			}
			fmt.Println("passphrase:", string(passphrase))
			// 4. Try again with passphrase
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
			if err != nil {
				fmt.Println("failed to parse private key with passphrase: %w", err)
				continue
			}
			return signer, nil
		}
		return nil, fmt.Errorf("failed to read passphrase:%w", err)
	}

	return nil, fmt.Errorf("unable to parse private key: %w", err)
}

// Prompt user for passphrase
func promptPassphrase() ([]byte, error) {
	fmt.Print("Enter passphrase:")
	passphrase, err := readline.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("failed to read passphrase: %w", err)
	}
	if len(passphrase) > 0 {
		return bytes.TrimSpace(passphrase), nil
	}
	return nil, fmt.Errorf("failed to read passphrase")
}

func parseKnownHostsFile(knownHostsFile ...string) (ssh.HostKeyCallback, error) {
	if len(knownHostsFile) == 0 {
		// If the file doesn't exist, we'll just assume we haven't connected before.
		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			firstConnection := true
			if firstConnection {
				fmt.Println("Fingerprint is SHA256: ", ssh.FingerprintSHA256(key))
				fmt.Println("Are you sure you want to continue connecting (yes/no)? ")
				reader := bufio.NewReader(os.Stdin)
				response, _, err := reader.ReadLine() // ReadLine gives a []byte, error
				if err != nil {
					return fmt.Errorf("failed to read user input: %w", err)
				}

				responseStr := strings.TrimSpace(string(response))
				if responseStr != "yes" && responseStr != "y" {
					return fmt.Errorf("connection refused by user")
				}
			}
			return nil
		}, nil
	}

	// If file exists, use the known hosts callback, if not we'll do the above.
	callback, err := knownhosts.New(knownHostsFile...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse known hosts file: %w", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// fmt.Printf("hostname: %v, remote: %v, key: %v\n", hostname, remote, key)
		err := callback(hostname, remote, key)
		if w, ok := err.(*knownhosts.KeyError); ok {
			fmt.Println("w.Want: ", w.Want)
			fmt.Println("WARNING: Host key mismatch!")
			fmt.Println("Fingerprint is SHA256: ", ssh.FingerprintSHA256(key))
			fmt.Print("Are you sure you want to continue connecting (yes/no)? ")
			reader := bufio.NewReader(os.Stdin)
			response, _, err := reader.ReadLine()
			if err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}
			responseStr := strings.TrimSpace(string(response))
			if responseStr != "yes" && responseStr != "y" {
				return fmt.Errorf("connection refused by user")
			}

			// User confirmed - Update the known_hosts file with the new key
			return appendKnownHostsFile(knownHostsFile[0], hostname, key)
		}
		// fmt.Println("err: ", err)
		return err
	}, nil
}

// append to the known_hosts files.
func appendKnownHostsFile(knownHostsFile string, hostname string, key ssh.PublicKey) error {
	f, err := os.OpenFile(knownHostsFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file to append: %w", err)
	}
	defer f.Close()
	fileInfo, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	var lastByte []byte = make([]byte, 1)
	if fileInfo.Size() > 0 {
		_, err = f.ReadAt(lastByte, fileInfo.Size()-1)
		if err != nil {
			return fmt.Errorf("failed to read last byte of known_hosts file: %w", err)
		}
	}
	if lastByte[0] != '\n' && fileInfo.Size() > 0 {
		_, err = f.Write([]byte("\n"))
		if err != nil {
			return fmt.Errorf("failed to write new line to known_hosts file: %w", err)
		}
	}
	newLine := knownhosts.Line([]string{hostname}, key)
	_, err = f.WriteString(newLine)
	if err != nil {
		return fmt.Errorf("failed to append new host key to known_hosts file: %w", err)
	}
	if newLine[len(newLine)-1] != '\n' {
		_, err = f.Write([]byte("\n"))
		if err != nil {
			return fmt.Errorf("failed to write new line to known_hosts file: %w", err)
		}
	}
	return nil
}

func connectSSH(cfg SSHConfig) (*ssh.Client, error) {
	knownHostsFile, err := parseKnownHostsFilePath()
	if err != nil {
		return nil, err
	}
	// fmt.Println("knownHostsFile: ", knownHostsFile)
	callback, err := parseKnownHostsFile(knownHostsFile)
	if err != nil {
		return nil, err
	}
	sshConfig := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{},
		// HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		HostKeyCallback: callback,
		Timeout:         10 * time.Second,
	}
	if cfg.IdentityFile != "" {
		// fmt.Println("cfg.IdentityFile: ", cfg.IdentityFile)
		signer, err := loadPrivateKey(cfg.IdentityFile)
		if err != nil {
			return nil, err
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	}
	if cfg.Password != "" {
		// fmt.Println("cfg.Password: ", cfg.Password)
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(cfg.Password))
	}
	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), sshConfig)
}

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

func PrintUsage() {
	Usages := strings.Join([]string{
		"Usage: mpipe [options...] [input] [output[,...]]",
		"\n",
		"Usage: mpipe 0.0.0.0:6777@udp '192.168.1.100:9090@tcp>,192.168.1.101:9090@udp<,192.168.1.102:9989@tcp'",
		"Usage: mpipe -ssh user@example.com:22 ssh:7890@tcp  local:7890@tcp=",
		"Usage: mpipe -ssh user@example.com:22 127.0.0.1:6379@tcp  ssh:6379@tcp",
		"\n",
		"Usage: mpipe :7890  192.168.1.100:7890",
		"Usage: mpipe -verbose localhost:7890@udp 192.168.1.100:7890@tcp",
		"Usage: mpipe -ssh user@example.com 127.0.0.1:6379@tcp  ssh:127.0.0.1:6379@tcp",
		"Usage(SSH MYSQL): mpipe -ssh sshName :6379 ssh:6379",
		"Usage(SSH PROXY): mpipe -ssh sshName ssh:7890 127.0.0.1:7890",
		"\n",
	}, "\n")
	fmt.Println(Usages)
	flag.PrintDefaults()
}

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
