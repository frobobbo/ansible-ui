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

// varsToSkip are inventory connection vars that are either redundant in our
// app (ansible_connection is always ssh) or reference paths on the local
// control machine that won't be valid on the runner (ssh key file paths).
var varsToSkip = map[string]bool{
	"ansible_connection":          true,
	"ansible_ssh_private_key_file": true,
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

// parseAnsibleINI parses an Ansible INI inventory file.
//
// Name resolution:
//   - If a group has exactly one host and that host has no explicit alias
//     (the line is a bare IP/hostname), the group name is used as the host name.
//   - Otherwise the alias or bare address is used as-is.
//
// Vars:
//   - [group:vars] entries are merged into every host in that group.
//   - Inline host vars (key=value on the host line) take precedence.
//   - varsToSkip entries (e.g. ansible_connection, ssh key file paths) are dropped.
func parseAnsibleINI(content []byte) []importedHost {
	type groupData struct {
		hostIndices []int
		vars        map[string]string
	}

	type rawHost struct {
		name        string // alias or bare address
		address     string
		hasAlias    bool // true when name != address
		inlineVars  map[string]string
		groupName   string
	}

	var rawHosts []rawHost
	seenNames := map[string]bool{}
	groups := map[string]*groupData{}

	currentGroup := ""
	currentSection := "" // "hosts" | "vars" | "children"

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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
			switch {
			case strings.HasSuffix(lower, ":vars"):
				currentGroup = section[:len(section)-5]
				currentSection = "vars"
				if _, ok := groups[currentGroup]; !ok {
					groups[currentGroup] = &groupData{vars: map[string]string{}}
				}
			case strings.HasSuffix(lower, ":children"):
				currentGroup = ""
				currentSection = "children"
			default:
				currentGroup = section
				currentSection = "hosts"
				if _, ok := groups[currentGroup]; !ok {
					groups[currentGroup] = &groupData{vars: map[string]string{}}
				}
			}
			continue
		}

		// Group vars line.
		if currentSection == "vars" {
			idx := strings.IndexByte(line, '=')
			if idx < 0 {
				continue
			}
			k := strings.TrimSpace(line[:idx])
			v := unquote(strings.TrimSpace(line[idx+1:]))
			if !varsToSkip[k] {
				groups[currentGroup].vars[k] = v
			}
			continue
		}

		// Host line.
		if currentSection == "hosts" {
			tokens := tokenizeHostLine(line)
			if len(tokens) == 0 {
				continue
			}

			inlineVars := make(map[string]string)
			for _, tok := range tokens[1:] {
				idx := strings.IndexByte(tok, '=')
				if idx < 0 {
					continue
				}
				k := tok[:idx]
				v := unquote(tok[idx+1:])
				if !varsToSkip[k] {
					inlineVars[k] = v
				}
			}

			address := tokens[0]
			hasAlias := false
			if ah, ok := inlineVars["ansible_host"]; ok {
				address = ah
				hasAlias = true
				delete(inlineVars, "ansible_host")
			}

			name := tokens[0]
			if seenNames[name] {
				continue
			}
			seenNames[name] = true

			idx := len(rawHosts)
			rawHosts = append(rawHosts, rawHost{
				name:       name,
				address:    address,
				hasAlias:   hasAlias,
				inlineVars: inlineVars,
				groupName:  currentGroup,
			})
			g := groups[currentGroup]
			g.hostIndices = append(g.hostIndices, idx)
		}
	}

	// Build final host list: apply group name and merge group vars.
	hosts := make([]importedHost, 0, len(rawHosts))
	for i := range rawHosts {
		rh := &rawHosts[i]
		g := groups[rh.groupName]

		// Use the group name as the host name when the group has a single host
		// and the host has no explicit alias (bare IP or FQDN was the only token).
		finalName := rh.name
		if !rh.hasAlias && len(g.hostIndices) == 1 && rh.groupName != "" {
			finalName = rh.groupName
		}

		// Merge vars: inline takes precedence over group vars.
		merged := make(map[string]string, len(g.vars)+len(rh.inlineVars))
		for k, v := range g.vars {
			merged[k] = v
		}
		for k, v := range rh.inlineVars {
			merged[k] = v
		}

		hosts = append(hosts, importedHost{
			Name:    finalName,
			Address: rh.address,
			Vars:    merged,
		})
	}

	return hosts
}

// tokenizeHostLine splits a host line respecting single- and double-quoted values.
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
