package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// EEEditorHandler talks to the GitHub Contents API to read/write EE definition files.
type EEEditorHandler struct{}

// NewEEEditorHandler creates an EEEditorHandler.
func NewEEEditorHandler() *EEEditorHandler {
	return &EEEditorHandler{}
}

// eeFileMap maps JSON keys to GitHub file paths.
var eeFileMap = map[string]string{
	"execution_environment_yml": "execution-environment/execution-environment.yml",
	"requirements_yml":          "execution-environment/requirements.yml",
	"requirements_txt":          "execution-environment/requirements.txt",
	"bindep_txt":                "execution-environment/bindep.txt",
}

// eeFileOrder defines the canonical order of keys in responses.
var eeFileOrder = []string{
	"execution_environment_yml",
	"requirements_yml",
	"requirements_txt",
	"bindep_txt",
}

type eeFileContent struct {
	Content string `json:"content"`
	SHA     string `json:"sha"`
}

type eeGetResponse map[string]eeFileContent

type eePutRequest struct {
	Message string                   `json:"message"`
	Files   map[string]eeFileContent `json:"files"`
}

// githubConfig holds the resolved GitHub env vars.
type githubConfig struct {
	Token  string
	Repo   string
	Branch string
}

func getGitHubConfig() (githubConfig, bool) {
	token := os.Getenv("GITHUB_TOKEN")
	repo := os.Getenv("GITHUB_REPO")
	branch := os.Getenv("GITHUB_BRANCH")
	if branch == "" {
		branch = "main"
	}
	if token == "" || repo == "" {
		return githubConfig{}, false
	}
	return githubConfig{Token: token, Repo: repo, Branch: branch}, true
}

// fetchGitHubFile fetches a single file from the GitHub Contents API.
// Returns content (decoded), sha, and any error. On 404, returns empty strings with no error.
func fetchGitHubFile(cfg githubConfig, path string) (content, sha string, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s", cfg.Repo, path, cfg.Branch)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", "", nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("github API error %d: %s", resp.StatusCode, string(body))
	}

	var ghResp struct {
		Content string `json:"content"`
		SHA     string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghResp); err != nil {
		return "", "", err
	}

	// GitHub base64 content includes embedded newlines — strip them before decoding.
	cleaned := strings.ReplaceAll(ghResp.Content, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return "", "", fmt.Errorf("base64 decode error: %w", err)
	}
	return string(decoded), ghResp.SHA, nil
}

// putGitHubFile creates or updates a file via the GitHub Contents API.
func putGitHubFile(cfg githubConfig, path, message, content, sha string) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(content))

	payload := map[string]interface{}{
		"message": message,
		"content": encoded,
		"branch":  cfg.Branch,
	}
	if sha != "" {
		payload["sha"] = sha
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", cfg.Repo, path)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// Get fetches all four EE definition files from GitHub.
// GET /api/ee
func (h *EEEditorHandler) Get(c *gin.Context) {
	cfg, ok := getGitHubConfig()
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GITHUB_TOKEN and GITHUB_REPO env vars are required"})
		return
	}

	result := make(eeGetResponse, len(eeFileOrder))
	for _, key := range eeFileOrder {
		path := eeFileMap[key]
		content, sha, err := fetchGitHubFile(cfg, path)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to fetch %s: %v", path, err)})
			return
		}
		result[key] = eeFileContent{Content: content, SHA: sha}
	}

	c.JSON(http.StatusOK, result)
}

// Update commits changes to EE definition files in GitHub.
// PUT /api/ee
func (h *EEEditorHandler) Update(c *gin.Context) {
	cfg, ok := getGitHubConfig()
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GITHUB_TOKEN and GITHUB_REPO env vars are required"})
		return
	}

	var req eePutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "commit message is required"})
		return
	}

	for _, key := range eeFileOrder {
		file, exists := req.Files[key]
		if !exists || file.Content == "" {
			continue
		}
		path := eeFileMap[key]
		if err := putGitHubFile(cfg, path, req.Message, file.Content, file.SHA); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to commit %s: %v", path, err)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "committed"})
}
