package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/ssh"
)

type RunResult struct {
	Output   string
	ExitCode int
	Err      error
}

// RunPlaybook executes ansible-playbook on the remote server.
// If preCommand is non-empty it is run before ansible-playbook in the same
// shell so its environment changes (e.g. PATH from a virtualenv activate
// script) are inherited. Example: ". /home/brett/ansible/bin/activate"
// Lines of output are sent to outputCh as they arrive.
// The caller must close outputCh after this returns.
func (c *SSHClient) RunPlaybook(ctx context.Context, playbookPath string, variables map[string]interface{}, preCommand string, outputCh chan<- string) RunResult {
	varJSON, err := json.Marshal(variables)
	if err != nil {
		return RunResult{Err: fmt.Errorf("marshal vars: %w", err)}
	}

	// Single-quote the JSON for shell safety; escape any embedded single quotes
	varStr := strings.ReplaceAll(string(varJSON), "'", `'"'"'`)
	ansibleCmd := fmt.Sprintf("ansible-playbook '%s' --extra-vars '%s'", playbookPath, varStr)

	var cmd string
	if preCommand != "" {
		// Run pre-command in the same shell so its environment changes
		// (e.g. PATH from virtualenv activate) are inherited by ansible-playbook.
		cmd = preCommand + " && " + ansibleCmd
	} else {
		cmd = ansibleCmd
	}

	session, err := c.client.NewSession()
	if err != nil {
		return RunResult{Err: fmt.Errorf("new session: %w", err)}
	}
	defer session.Close()

	pr, pw := io.Pipe()
	var buf bytes.Buffer
	mw := io.MultiWriter(&buf, pw)
	session.Stdout = mw
	session.Stderr = mw

	// Stream output lines to channel
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			select {
			case outputCh <- line:
			case <-ctx.Done():
				return
			}
		}
	}()

	exitCode := 0
	runErr := session.Run(cmd)
	pw.Close()
	<-done // wait for scanner to finish

	if runErr != nil {
		if exitErr, ok := runErr.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
			runErr = nil // non-zero exit is a playbook failure, not a runner error
		} else {
			return RunResult{Err: runErr, Output: buf.String()}
		}
	}

	return RunResult{
		Output:   buf.String(),
		ExitCode: exitCode,
	}
}
