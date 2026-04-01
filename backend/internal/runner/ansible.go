package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
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
// script) are inherited.
// If vaultPassword is non-empty it is written to a temp file and passed via
// --vault-password-file; the temp file is cleaned up after the run.
// If vaultFileContent is non-nil it is uploaded to a temp file on the remote
// and passed via --extra-vars "@path" so ansible decrypts it automatically.
// Lines of output are sent to outputCh as they arrive.
// The caller must close outputCh after this returns.
func (c *SSHClient) RunPlaybook(ctx context.Context, playbookPath string, variables map[string]interface{}, inventoryTarget string, preCommand string, vaultPassword string, vaultFileContent []byte, vaultFileName string, sshCertContent []byte, outputCh chan<- string) RunResult {
	varJSON, err := json.Marshal(variables)
	if err != nil {
		return RunResult{Err: fmt.Errorf("marshal vars: %w", err)}
	}

	// Single-quote the JSON for shell safety; escape any embedded single quotes
	varStr := strings.ReplaceAll(string(varJSON), "'", `'"'"'`)
	ansibleCmd := fmt.Sprintf("ansible-playbook '%s' --extra-vars '%s'", playbookPath, varStr)
	if inventoryTarget != "" {
		// If an SSH cert is provided, upload it and inject the key path into the inventory.
		if len(sshCertContent) > 0 {
			certPath := strings.TrimSuffix(playbookPath, ".yml") + "-cert"
			if err := c.UploadFile(sshCertContent, certPath); err != nil {
				return RunResult{Err: fmt.Errorf("upload ssh cert: %w", err)}
			}
			c.RunCommand(fmt.Sprintf("chmod 600 '%s'", certPath))
			defer c.RunCommand(fmt.Sprintf("rm -f '%s'", certPath))
			inventoryTarget = strings.TrimSuffix(inventoryTarget, "\n") + " ansible_ssh_private_key_file=" + certPath + "\n"
		}
		inventoryPath := strings.TrimSuffix(playbookPath, ".yml") + "-inventory"
		if err := c.UploadFile([]byte(inventoryTarget), inventoryPath); err != nil {
			return RunResult{Err: fmt.Errorf("upload inventory: %w", err)}
		}
		defer c.RunCommand(fmt.Sprintf("rm -f '%s'", inventoryPath))
		ansibleCmd = fmt.Sprintf("ansible-playbook '%s' -i '%s' --extra-vars '%s'", playbookPath, inventoryPath, varStr)
	}

	// Upload vault password to a temp file on remote and add the flag
	if vaultPassword != "" {
		vaultPassPath := strings.TrimSuffix(playbookPath, ".yml") + "-vault-pass"
		if err := c.UploadFile([]byte(vaultPassword), vaultPassPath); err != nil {
			return RunResult{Err: fmt.Errorf("upload vault pass: %w", err)}
		}
		defer c.RunCommand(fmt.Sprintf("rm -f '%s'", vaultPassPath))
		ansibleCmd += fmt.Sprintf(" --vault-password-file '%s'", vaultPassPath)
	}

	// Upload vault vars file to remote and pass as extra-vars.
	// Also place it at /tmp/<stem>/<filename> so playbooks using the
	// vars_files: ./creds/creds.yml (stem-as-dir) convention find it.
	if len(vaultFileContent) > 0 {
		vaultVarsPath := strings.TrimSuffix(playbookPath, ".yml") + "-vault-vars.yml"
		if err := c.UploadFile(vaultFileContent, vaultVarsPath); err != nil {
			return RunResult{Err: fmt.Errorf("upload vault vars: %w", err)}
		}
		defer c.RunCommand(fmt.Sprintf("rm -f '%s'", vaultVarsPath))
		ansibleCmd += fmt.Sprintf(" --extra-vars '@%s'", vaultVarsPath)

		if vaultFileName != "" {
			stem := strings.TrimSuffix(vaultFileName, filepath.Ext(vaultFileName))
			if stem != "" && stem != vaultFileName {
				stemPath := "/tmp/" + stem + "/" + vaultFileName
				c.RunCommand("mkdir -p '/tmp/" + stem + "'")
				if err := c.UploadFile(vaultFileContent, stemPath); err == nil {
					defer c.RunCommand(fmt.Sprintf("rm -rf '/tmp/%s'", stem))
				}
			}
		}
	}

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
	runDone := make(chan error, 1)
	go func() { runDone <- session.Run(cmd) }()
	var runErr error
	select {
	case runErr = <-runDone:
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		session.Close()
		runErr = fmt.Errorf("cancelled")
	}
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
