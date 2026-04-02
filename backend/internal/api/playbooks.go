package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/brettjrea/ansible-frontend/internal/store"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type PlaybooksHandler struct {
	playbooks *store.PlaybookStore
	audit     *store.AuditStore
}

func newPlaybooksHandler(playbooks *store.PlaybookStore, audit *store.AuditStore) *PlaybooksHandler {
	return &PlaybooksHandler{playbooks: playbooks, audit: audit}
}

func (h *PlaybooksHandler) List(c *gin.Context) {
	list, err := h.playbooks.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*models.Playbook{}
	}
	c.JSON(http.StatusOK, list)
}

func (h *PlaybooksHandler) Get(c *gin.Context) {
	p, err := h.playbooks.Get(c.Param("id"))
	if err != nil || p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playbook source not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

type playbookBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RepoURL     string `json:"repo_url"`
	Branch      string `json:"branch"`
	Token       string `json:"token"`
}

func (h *PlaybooksHandler) Create(c *gin.Context) {
	var body playbookBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Name == "" || body.RepoURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and repo_url are required"})
		return
	}
	if body.Branch == "" {
		body.Branch = "main"
	}

	p, err := h.playbooks.Create(body.Name, body.Description, body.RepoURL, body.Branch, body.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "create", "playbook_source", p.ID, "", c.ClientIP())
	c.JSON(http.StatusCreated, p)
}

func (h *PlaybooksHandler) Update(c *gin.Context) {
	var body playbookBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Name == "" || body.RepoURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and repo_url are required"})
		return
	}
	if body.Branch == "" {
		body.Branch = "main"
	}

	p, err := h.playbooks.Update(c.Param("id"), body.Name, body.Description, body.RepoURL, body.Branch, body.Token)
	if err != nil || p == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "update", "playbook_source", p.ID, "", c.ClientIP())
	c.JSON(http.StatusOK, p)
}

func (h *PlaybooksHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.playbooks.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid, uname := auditUser(c)
	h.audit.Log(uid, uname, "delete", "playbook_source", id, "", c.ClientIP())
	c.Status(http.StatusNoContent)
}

// Files clones the source repo and returns a sorted list of .yml/.yaml files.
func (h *PlaybooksHandler) Files(c *gin.Context) {
	p, err := h.playbooks.Get(c.Param("id"))
	if err != nil || p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playbook source not found"})
		return
	}

	dir, err := os.MkdirTemp("", "ansible-clone-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tempdir: " + err.Error()})
		return
	}
	defer os.RemoveAll(dir)

	if err := cloneShallow(c.Request.Context(), p, dir); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var files []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yml" || ext == ".yaml" {
			rel, _ := filepath.Rel(dir, path)
			files = append(files, rel)
		}
		return nil
	})
	sort.Strings(files)
	if files == nil {
		files = []string{}
	}
	c.JSON(http.StatusOK, files)
}

// VarSuggestion is a proposed form field extracted from a playbook.
type VarSuggestion struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Type     string `json:"type"` // text | number | bool
	Default  string `json:"default"`
	Required bool   `json:"required"`
}

var jinja2VarRe = regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*[\|}\s]`)

// Scan clones the source repo, reads the specified playbook file, and returns
// suggested form fields inferred from vars:, vars_prompt:, and {{ }} patterns.
func (h *PlaybooksHandler) Scan(c *gin.Context) {
	pbPath := c.Query("path")
	if pbPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path query param is required"})
		return
	}

	p, err := h.playbooks.Get(c.Param("id"))
	if err != nil || p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playbook source not found"})
		return
	}

	dir, err := os.MkdirTemp("", "ansible-clone-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer os.RemoveAll(dir)

	if err := cloneShallow(c.Request.Context(), p, dir); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	content, err := os.ReadFile(filepath.Join(dir, pbPath))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("read %s: %v", pbPath, err)})
		return
	}

	c.JSON(http.StatusOK, extractVarSuggestions(content))
}

func extractVarSuggestions(content []byte) []VarSuggestion {
	seen := map[string]*VarSuggestion{}

	// Structured pass: parse plays as generic YAML nodes.
	var plays []map[string]interface{}
	if err := yaml.Unmarshal(content, &plays); err == nil {
		for _, play := range plays {
			if vars, ok := play["vars"].(map[string]interface{}); ok {
				for name, val := range vars {
					s := &VarSuggestion{
						Name:    name,
						Label:   varToLabel(name),
						Type:    inferVarType(val),
						Default: fmt.Sprintf("%v", val),
					}
					seen[name] = s
				}
			}
			if prompts, ok := play["vars_prompt"].([]interface{}); ok {
				for _, item := range prompts {
					if pm, ok := item.(map[string]interface{}); ok {
						name := fmt.Sprintf("%v", pm["name"])
						label := varToLabel(name)
						if pr, ok := pm["prompt"].(string); ok && pr != "" {
							label = pr
						}
						seen[name] = &VarSuggestion{
							Name:     name,
							Label:    label,
							Type:     "text",
							Required: true,
						}
					}
				}
			}
		}
	}

	// Regex pass: catch {{ var }} patterns not found above.
	for _, m := range jinja2VarRe.FindAllSubmatch(content, -1) {
		name := string(m[1])
		if _, exists := seen[name]; !exists {
			seen[name] = &VarSuggestion{
				Name:  name,
				Label: varToLabel(name),
				Type:  "text",
			}
		}
	}

	suggestions := make([]VarSuggestion, 0, len(seen))
	for _, s := range seen {
		suggestions = append(suggestions, *s)
	}
	sort.Slice(suggestions, func(i, j int) bool { return suggestions[i].Name < suggestions[j].Name })
	return suggestions
}

func inferVarType(val interface{}) string {
	switch val.(type) {
	case bool:
		return "bool"
	case int, int64, float64:
		return "number"
	default:
		return "text"
	}
}

func varToLabel(name string) string {
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
