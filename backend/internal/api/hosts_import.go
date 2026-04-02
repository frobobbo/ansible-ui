package api

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

type importedHost struct {
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Vars    map[string]string `json:"vars"`
}

type importResult struct {
	Created []string `json:"created"`
	Skipped []string `json:"skipped"`
	Errors  []string `json:"errors"`
}

// Import parses an uploaded Ansible INI inventory file and bulk-creates hosts.
func (h *HostsHandler) Import(c *gin.Context) {
	f, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer f.Close()

	content, err := io.ReadAll(io.LimitReader(f, 1<<20)) // 1 MB max
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read file"})
		return
	}

	parsed := parseAnsibleINI(content)

	uid, uname := auditUser(c)
	result := importResult{
		Created: []string{},
		Skipped: []string{},
		Errors:  []string{},
	}

	existing, _ := h.hosts.List()
	existingNames := make(map[string]bool, len(existing))
	for _, host := range existing {
		existingNames[host.Name] = true
	}

	for _, ph := range parsed {
		if existingNames[ph.Name] {
			result.Skipped = append(result.Skipped, ph.Name)
			continue
		}
		host, cerr := h.hosts.Create(ph.Name, ph.Address, "", nil, ph.Vars)
		if cerr != nil {
			result.Errors = append(result.Errors, ph.Name+": "+cerr.Error())
			continue
		}
		h.audit.Log(uid, uname, "create", "host", host.ID, "imported from inventory file", c.ClientIP())
		result.Created = append(result.Created, ph.Name)
		existingNames[ph.Name] = true
	}

	c.JSON(http.StatusOK, result)
}

// parseAnsibleINI parses an Ansible INI inventory file and returns the
// discovered hosts. Groups are ignored (hosts from all groups are merged).
// ansible_host is used as the Address and stripped from Vars.
func parseAnsibleINI(content []byte) []importedHost {
	var hosts []importedHost
	seen := map[string]bool{}

	// inHostSection is false for :vars and :children sections.
	inHostSection := true

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments.
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}

		// Section header.
		if line[0] == '[' {
			end := strings.IndexByte(line, ']')
			if end < 0 {
				continue
			}
			section := line[1:end]
			lower := strings.ToLower(section)
			inHostSection = !strings.HasSuffix(lower, ":vars") &&
				!strings.HasSuffix(lower, ":children")
			continue
		}

		if !inHostSection {
			continue
		}

		// Host line: first token is the alias/hostname.
		tokens := tokenizeHostLine(line)
		if len(tokens) == 0 {
			continue
		}
		name := tokens[0]
		if seen[name] {
			continue
		}

		vars := make(map[string]string)
		for _, tok := range tokens[1:] {
			idx := strings.IndexByte(tok, '=')
			if idx < 0 {
				continue
			}
			k := tok[:idx]
			v := unquote(tok[idx+1:])
			vars[k] = v
		}

		address := name
		if ah, ok := vars["ansible_host"]; ok {
			address = ah
			delete(vars, "ansible_host")
		}
		if address == "" {
			address = name
		}

		hosts = append(hosts, importedHost{Name: name, Address: address, Vars: vars})
		seen[name] = true
	}

	return hosts
}

// tokenizeHostLine splits a host line respecting quoted values.
// e.g. `web1 ansible_host=10.0.0.1 ansible_user="ubuntu user"` →
//
//	["web1", "ansible_host=10.0.0.1", `ansible_user="ubuntu user"`]
func tokenizeHostLine(line string) []string {
	var tokens []string
	var cur strings.Builder
	inQ := rune(0)

	for _, ch := range line {
		switch {
		case inQ != 0:
			cur.WriteRune(ch)
			if ch == inQ {
				inQ = 0
			}
		case ch == '"' || ch == '\'':
			cur.WriteRune(ch)
			inQ = ch
		case unicode.IsSpace(ch):
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// unquote strips surrounding single or double quotes from a string.
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
