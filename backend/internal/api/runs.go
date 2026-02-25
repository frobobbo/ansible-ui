package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/runner"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

type RunsHandler struct {
	runs      *store.RunStore
	forms     *store.FormStore
	servers   *store.ServerStore
	playbooks *store.PlaybookStore
	vaults    *store.VaultStore
}

func newRunsHandler(runs *store.RunStore, forms *store.FormStore, servers *store.ServerStore, playbooks *store.PlaybookStore, vaults *store.VaultStore) *RunsHandler {
	return &RunsHandler{runs: runs, forms: forms, servers: servers, playbooks: playbooks, vaults: vaults}
}

func (h *RunsHandler) List(c *gin.Context) {
	list, err := h.runs.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*models.Run{}
	}
	c.JSON(http.StatusOK, list)
}

func (h *RunsHandler) Get(c *gin.Context) {
	r, err := h.runs.Get(c.Param("id"))
	if err != nil || r == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *RunsHandler) Create(c *gin.Context) {
	var req struct {
		FormID    string                 `json:"form_id" binding:"required"`
		Variables map[string]interface{} `json:"variables"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	form, err := h.forms.Get(req.FormID)
	if err != nil || form == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "form not found"})
		return
	}

	varJSON, _ := json.Marshal(req.Variables)

	fid := req.FormID
	run, err := h.runs.Create(&fid, form.PlaybookID, form.ServerID, string(varJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Execute asynchronously
	go h.executeRun(run.ID, form, req.Variables)

	c.JSON(http.StatusAccepted, gin.H{"run_id": run.ID, "status": "pending"})
}

func (h *RunsHandler) executeRun(runID string, form *models.Form, variables map[string]interface{}) {
	ctx := context.Background()

	h.runs.SetRunning(runID)

	// Load server with SSH key
	server, err := h.servers.Get(form.ServerID)
	if err != nil || server == nil {
		h.runs.Finish(runID, "failed", fmt.Sprintf("server not found: %v", err))
		return
	}

	// Load playbook file
	playbook, err := h.playbooks.Get(form.PlaybookID)
	if err != nil || playbook == nil {
		h.runs.Finish(runID, "failed", fmt.Sprintf("playbook not found: %v", err))
		return
	}

	playbookContent, err := os.ReadFile(playbook.FilePath)
	if err != nil {
		h.runs.Finish(runID, "failed", fmt.Sprintf("read playbook: %v", err))
		return
	}

	// Connect via SSH
	client, err := runner.Connect(server.Host, server.Port, server.Username, server.SSHPrivateKey)
	if err != nil {
		h.runs.Finish(runID, "failed", fmt.Sprintf("SSH connect failed: %v", err))
		return
	}
	defer client.Close()

	// Upload playbook to remote /tmp/
	remotePath := fmt.Sprintf("/tmp/ansible-run-%s.yml", runID)
	if err := client.UploadFile(playbookContent, remotePath); err != nil {
		h.runs.Finish(runID, "failed", fmt.Sprintf("upload playbook: %v", err))
		return
	}
	defer client.RunCommand(fmt.Sprintf("rm -f '%s'", remotePath))

	// Decrypt vault password + read vault file if the form references a vault
	var vaultPassword string
	var vaultFileContent []byte
	if form.VaultID != nil {
		vaultPassword, err = h.vaults.GetDecryptedPassword(*form.VaultID)
		if err != nil {
			h.runs.Finish(runID, "failed", fmt.Sprintf("decrypt vault: %v", err))
			return
		}

		filePath, err := h.vaults.GetVaultFilePath(*form.VaultID)
		if err != nil {
			h.runs.Finish(runID, "failed", fmt.Sprintf("get vault file path: %v", err))
			return
		}
		if filePath != "" {
			vaultFileContent, err = os.ReadFile(filePath)
			if err != nil {
				h.runs.Finish(runID, "failed", fmt.Sprintf("read vault file: %v", err))
				return
			}
		}
	}

	// Stream output
	outputCh := make(chan string, 256)
	var outputBuilder strings.Builder

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		for line := range outputCh {
			outputBuilder.WriteString(line + "\n")
		}
	}()

	result := client.RunPlaybook(ctx, remotePath, variables, server.PreCommand, vaultPassword, vaultFileContent, outputCh)
	close(outputCh)
	<-doneCh

	fullOutput := outputBuilder.String()
	if result.Err != nil {
		fullOutput += "\nRunner error: " + result.Err.Error()
	}

	status := "success"
	if result.ExitCode != 0 || result.Err != nil {
		status = "failed"
	}

	h.runs.Finish(runID, status, fullOutput)
}
