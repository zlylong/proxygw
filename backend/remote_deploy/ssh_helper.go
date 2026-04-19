package remote_deploy

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient wrapper
type SSHClient struct {
	client *ssh.Client
}

func getStrictHostKeyCallback(expectedHostKey string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint := "SHA256:" + base64.RawStdEncoding.EncodeToString(func(k []byte) []byte { h := sha256.Sum256(k); return h[:] }(key.Marshal()))

		if expectedHostKey == "" {
			return fmt.Errorf("Strict Host Key checking failed. The server's fingerprint is %s. Please update the node configuration with this fingerprint to connect securely.", fingerprint)
		}

		if expectedHostKey == fingerprint || expectedHostKey == base64.StdEncoding.EncodeToString(key.Marshal()) {
			return nil
		}

		return fmt.Errorf("Host key verification failed! Expected %s but got %s", expectedHostKey, fingerprint)
	}
}

// Connect establishes an SSH connection using either password or private key
func Connect(host string, port int, user string, authType string, credential string, expectedHostKey string) (*SSHClient, error) {
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
		HostKeyCallback: getStrictHostKeyCallback(expectedHostKey),
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
