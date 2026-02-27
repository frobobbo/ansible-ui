package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/brettjrea/ansible-frontend/internal/auth"
	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/notify"
	"github.com/brettjrea/ansible-frontend/internal/runner"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
)

// liveRun holds in-progress output and SSE subscribers for a single run.
type liveRun struct {
	mu       sync.Mutex
	lines    []string
	subs     []chan string
	done     bool
	status   string
	cancelFn context.CancelFunc
}

type RunsHandler struct {
	runs      *store.RunStore
	forms     *store.FormStore
	servers   *store.ServerStore
	playbooks *store.PlaybookStore
	vaults    *store.VaultStore
	audit     *store.AuditStore
	jwtSvc    *auth.JWTService
	liveRuns  sync.Map // string -> *liveRun
}

func NewRunsHandler(
	runs *store.RunStore,
	forms *store.FormStore,
	servers *store.ServerStore,
	playbooks *store.PlaybookStore,
	vaults *store.VaultStore,
	audit *store.AuditStore,
	jwtSvc *auth.JWTService,
) *RunsHandler {
	return &RunsHandler{
		runs:      runs,
		forms:     forms,
		servers:   servers,
		playbooks: playbooks,
		vaults:    vaults,
		audit:     audit,
		jwtSvc:    jwtSvc,
	}
}

// ── REST handlers ─────────────────────────────────────────────────────────────

func (h *RunsHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	total, _ := h.runs.Count()
	c.Header("X-Total-Count", strconv.Itoa(total))

	list, err := h.runs.List(limit, offset)
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

	go h.executeRun(run.ID, form, req.Variables)

	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "create", "run", run.ID, "", c.ClientIP())
	c.JSON(http.StatusAccepted, gin.H{"run_id": run.ID, "status": "pending"})
}

// ── SSE streaming ─────────────────────────────────────────────────────────────

// Stream serves a run's output as a Server-Sent Events stream.
// Authentication accepts Bearer token in the Authorization header OR a ?token= query param,
// because EventSource cannot set custom headers.
func (h *RunsHandler) Stream(c *gin.Context) {
	id := c.Param("id")

	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if token == "" {
		token = c.Query("token")
	}
	if _, err := h.jwtSvc.Verify(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	run, _ := h.runs.Get(id)
	if run == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := c.Writer
	sendLine := func(line string) {
		fmt.Fprintf(w, "data: %s\n\n", line)
		w.Flush()
	}
	sendDone := func(status string) {
		fmt.Fprintf(w, "event: done\ndata: %s\n\n", status)
		w.Flush()
	}

	if val, ok := h.liveRuns.Load(id); ok {
		lr := val.(*liveRun)
		lr.mu.Lock()
		buf := make([]string, len(lr.lines))
		copy(buf, lr.lines)

		if lr.done {
			status := lr.status
			lr.mu.Unlock()
			for _, line := range buf {
				sendLine(line)
			}
			sendDone(status)
			return
		}

		ch := make(chan string, 512)
		lr.subs = append(lr.subs, ch)
		lr.mu.Unlock()
		defer h.unsubFromRun(id, ch)

		for _, line := range buf {
			sendLine(line)
		}

		ctx := c.Request.Context()
		for {
			select {
			case line, ok := <-ch:
				if !ok {
					run2, _ := h.runs.Get(id)
					status := "failed"
					if run2 != nil {
						status = run2.Status
					}
					sendDone(status)
					return
				}
				sendLine(line)
			case <-ctx.Done():
				return
			}
		}
	}

	// Fall back: serve stored output from DB.
	for _, line := range strings.Split(run.Output, "\n") {
		if line != "" {
			sendLine(line)
		}
	}
	sendDone(run.Status)
}

// ── Live-run helpers ──────────────────────────────────────────────────────────

func (h *RunsHandler) unsubFromRun(runID string, ch chan string) {
	val, ok := h.liveRuns.Load(runID)
	if !ok {
		return
	}
	lr := val.(*liveRun)
	lr.mu.Lock()
	for i, s := range lr.subs {
		if s == ch {
			lr.subs = append(lr.subs[:i], lr.subs[i+1:]...)
			break
		}
	}
	lr.mu.Unlock()
}

func (h *RunsHandler) broadcastLine(runID, line string) {
	val, ok := h.liveRuns.Load(runID)
	if !ok {
		return
	}
	lr := val.(*liveRun)
	lr.mu.Lock()
	lr.lines = append(lr.lines, line)
	for _, ch := range lr.subs {
		select {
		case ch <- line:
		default:
		}
	}
	lr.mu.Unlock()
}

func (h *RunsHandler) finishLiveRun(runID, status string) {
	val, ok := h.liveRuns.Load(runID)
	if !ok {
		return
	}
	lr := val.(*liveRun)
	lr.mu.Lock()
	lr.done = true
	lr.status = status
	for _, ch := range lr.subs {
		close(ch)
	}
	lr.subs = nil
	lr.mu.Unlock()
}

// ── Execution ─────────────────────────────────────────────────────────────────

// Cancel stops an in-progress run by cancelling its context.
func (h *RunsHandler) Cancel(c *gin.Context) {
	val, ok := h.liveRuns.Load(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not in progress"})
		return
	}
	lr := val.(*liveRun)
	lr.mu.Lock()
	if lr.cancelFn != nil {
		lr.cancelFn()
	}
	lr.mu.Unlock()
	c.Status(http.StatusNoContent)
}

func (h *RunsHandler) executeRun(runID string, form *models.Form, variables map[string]interface{}) {
	ctx, cancel := context.WithCancel(context.Background())

	lr := &liveRun{cancelFn: cancel}
	h.liveRuns.Store(runID, lr)

	fail := func(msg string) {
		h.runs.Finish(runID, "failed", msg)
		h.finishLiveRun(runID, "failed")
	}

	h.runs.SetRunning(runID)

	server, err := h.servers.Get(form.ServerID)
	if err != nil || server == nil {
		fail(fmt.Sprintf("server not found: %v", err))
		return
	}

	playbook, err := h.playbooks.Get(form.PlaybookID)
	if err != nil || playbook == nil {
		fail(fmt.Sprintf("playbook not found: %v", err))
		return
	}

	playbookContent, err := os.ReadFile(playbook.FilePath)
	if err != nil {
		fail(fmt.Sprintf("read playbook: %v", err))
		return
	}

	client, err := runner.Connect(server.Host, server.Port, server.Username, server.SSHPrivateKey)
	if err != nil {
		fail(fmt.Sprintf("SSH connect failed: %v", err))
		return
	}
	defer client.Close()

	remotePath := fmt.Sprintf("/tmp/ansible-run-%s.yml", runID)
	if err := client.UploadFile(playbookContent, remotePath); err != nil {
		fail(fmt.Sprintf("upload playbook: %v", err))
		return
	}
	defer client.RunCommand(fmt.Sprintf("rm -f '%s'", remotePath))

	var vaultPassword string
	var vaultFileContent []byte
	if form.VaultID != nil {
		vaultPassword, err = h.vaults.GetDecryptedPassword(*form.VaultID)
		if err != nil {
			fail(fmt.Sprintf("decrypt vault: %v", err))
			return
		}
		filePath, err := h.vaults.GetVaultFilePath(*form.VaultID)
		if err != nil {
			fail(fmt.Sprintf("get vault file path: %v", err))
			return
		}
		if filePath != "" {
			vaultFileContent, err = os.ReadFile(filePath)
			if err != nil {
				fail(fmt.Sprintf("read vault file: %v", err))
				return
			}
		}
	}

	outputCh := make(chan string, 256)
	var outputBuilder strings.Builder
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		for line := range outputCh {
			outputBuilder.WriteString(line + "\n")
			h.broadcastLine(runID, line)
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
	h.finishLiveRun(runID, status)

	// Fire completion notifications (webhook + email) if configured on the form.
	if form.NotifyWebhook != "" || form.NotifyEmail != "" {
		go notify.Send(form.NotifyWebhook, form.NotifyEmail, runID, status, form.Name)
	}
}

// TriggerWebhook handles unauthenticated webhook triggers via a form's token.
// POST /api/webhook/forms/:token
func (h *RunsHandler) TriggerWebhook(c *gin.Context) {
	token := c.Param("token")
	form, err := h.forms.GetByWebhookToken(token)
	if err != nil || form == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid webhook token"})
		return
	}

	// Build variables from field defaults, allow overrides from the request body.
	variables := make(map[string]interface{})
	for _, f := range form.Fields {
		switch f.FieldType {
		case "bool":
			variables[f.Name] = f.DefaultValue == "true"
		default:
			variables[f.Name] = f.DefaultValue
		}
	}
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err == nil {
		for k, v := range body {
			variables[k] = v
		}
	}

	varJSON, _ := json.Marshal(variables)
	fid := form.ID
	run, err := h.runs.Create(&fid, form.PlaybookID, form.ServerID, string(varJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	go h.executeRun(run.ID, form, variables)
	h.audit.Log("", "webhook", "trigger", "run", run.ID, "", c.ClientIP())
	c.JSON(http.StatusAccepted, gin.H{"run_id": run.ID, "status": "pending"})
}

// TriggerScheduledRun is the callback invoked by the scheduler on each cron tick.
// It creates a run record and launches executeRun in a goroutine.
func (h *RunsHandler) TriggerScheduledRun(form *models.Form, variables map[string]interface{}) {
	varJSON, _ := json.Marshal(variables)
	fid := form.ID
	run, err := h.runs.Create(&fid, form.PlaybookID, form.ServerID, string(varJSON))
	if err != nil {
		log.Printf("[scheduler] failed to create run for form %s: %v", form.ID, err)
		return
	}
	go h.executeRun(run.ID, form, variables)
}
