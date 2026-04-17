package remote_deploy

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"time"
)

// SSHClient wrapper
type SSHClient struct {
	client *ssh.Client
}

// Connect establishes an SSH connection using either password or private key

func getKnownHostsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh", "known_hosts")
}

func getHostKeyCallback() ssh.HostKeyCallback {
	path := getKnownHostsPath()
	os.MkdirAll(filepath.Dir(path), 0700)
	
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.WriteFile(path, []byte(""), 0600)
	}
	
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		hkCallback, err := knownhosts.New(path)
		if err != nil {
			return fmt.Errorf("failed to load known_hosts: %v", err)
		}
		
		err = hkCallback(hostname, remote, key)
		if err != nil {
			keyErr, ok := err.(*knownhosts.KeyError)
			if ok && len(keyErr.Want) == 0 {
				f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
				if err != nil {
					return err
				}
				defer f.Close()
				
				knownHostLine := knownhosts.Line([]string{hostname}, key)
				_, err = f.WriteString(knownHostLine + "\n")
				return err
			}
			return err
		}
		return nil
	}
}

func Connect(host string, port int, user string, authType string, credential string) (*SSHClient, error) {
	var authMethod ssh.AuthMethod

	if authType == "password" {
		authMethod = ssh.Password(credential)
	} else if authType == "key" {
		signer, err := ssh.ParsePrivateKey([]byte(credential))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		authMethod = ssh.PublicKeys(signer)
	} else {
		return nil, fmt.Errorf("unsupported auth type: %s", authType)
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: getHostKeyCallback(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %v", err)
	}

	return &SSHClient{client: client}, nil
}

// RunCommand executes a command and returns stdout, stderr and error
func (c *SSHClient) RunCommand(cmd string) (string, string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	err = session.Run(cmd)
	return stdoutBuf.String(), stderrBuf.String(), err
}

// Close closes the SSH connection
func (c *SSHClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
