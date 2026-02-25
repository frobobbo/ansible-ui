package runner

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	client *ssh.Client
}

func Connect(host string, port int, username, privateKeyPEM string) (*SSHClient, error) {
	pem := strings.TrimSpace(privateKeyPEM)
	signer, err := ssh.ParsePrivateKey([]byte(pem))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return &SSHClient{client: client}, nil
}

func (c *SSHClient) Close() error {
	return c.client.Close()
}

// UploadFile writes content to remotePath on the remote host using cat via SSH stdin.
func (c *SSHClient) UploadFile(content []byte, remotePath string) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	session.Stdin = bytes.NewReader(content)
	cmd := fmt.Sprintf("cat > %s", remotePath)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("upload to %s: %w", remotePath, err)
	}
	return nil
}

// RunCommand executes a simple command and returns combined output.
func (c *SSHClient) RunCommand(cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	return string(out), err
}
