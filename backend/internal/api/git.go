package api

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/brettjrea/ansible-frontend/internal/models"
)

// cloneShallow does a depth-1 clone of a playbook source repo into dir.
// If a token is set it is injected into the HTTPS URL. Token strings are
// redacted from any error output.
func cloneShallow(ctx context.Context, p *models.Playbook, dir string) error {
	cloneURL := p.RepoURL
	if p.Token != "" {
		if u, uerr := url.Parse(cloneURL); uerr == nil && (u.Scheme == "https" || u.Scheme == "http") {
			u.User = url.UserPassword("oauth2", p.Token)
			cloneURL = u.String()
		}
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", p.Branch, "--single-branch", cloneURL, dir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(out)
		if p.Token != "" {
			msg = strings.ReplaceAll(msg, p.Token, "***")
		}
		return fmt.Errorf("git clone: %w\n%s", err, strings.TrimSpace(msg))
	}
	return nil
}
