package chat

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitContext holds information about the current git repository.
type GitContext struct {
	IsRepo       bool
	Branch       string
	RemoteURL    string
	RepoOwner    string
	RepoName     string
	RecentCommits []string // last 5 commit summaries
	WorkDir      string
}

// DetectGitContext gathers git info from the current working directory.
func DetectGitContext() *GitContext {
	ctx := &GitContext{}

	// Check if we're in a git repo
	if err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		return ctx
	}
	ctx.IsRepo = true

	// Working directory (git root)
	if out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		ctx.WorkDir = strings.TrimSpace(string(out))
	}

	// Current branch
	if out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		ctx.Branch = strings.TrimSpace(string(out))
	}

	// Remote URL (origin)
	if out, err := exec.Command("git", "config", "--get", "remote.origin.url").Output(); err == nil {
		ctx.RemoteURL = strings.TrimSpace(string(out))
		ctx.RepoOwner, ctx.RepoName = parseGitRemote(ctx.RemoteURL)
	}

	// Recent commits (last 5)
	if out, err := exec.Command("git", "log", "--oneline", "-5").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if line != "" {
				ctx.RecentCommits = append(ctx.RecentCommits, line)
			}
		}
	}

	return ctx
}

// BuildGitContextPrompt creates a system prompt section with git/project info.
func BuildGitContextPrompt(ctx *GitContext) string {
	if ctx == nil || !ctx.IsRepo {
		return ""
	}

	var b strings.Builder
	b.WriteString("GIT REPOSITORY INFO (for GitHub MCP tools and git-related questions ONLY):\n")

	if ctx.RepoOwner != "" && ctx.RepoName != "" {
		b.WriteString(fmt.Sprintf("- GitHub: %s/%s\n", ctx.RepoOwner, ctx.RepoName))
	}
	if ctx.Branch != "" {
		b.WriteString(fmt.Sprintf("- Branch: %s\n", ctx.Branch))
	}
	if ctx.WorkDir != "" {
		b.WriteString(fmt.Sprintf("- Directory: %s\n", ctx.WorkDir))
	}

	b.WriteString("\nUSAGE RULES:")
	if ctx.RepoOwner != "" && ctx.RepoName != "" {
		b.WriteString("\n- When using GitHub MCP tools, use owner=\"" + ctx.RepoOwner + "\" and repo=\"" + ctx.RepoName + "\".")
	}
	b.WriteString("\n- Use this info ONLY for git/GitHub questions: branch, commits, PRs, issues.")
	b.WriteString("\n- For architecture, code structure, or implementation questions — use semantic_search or filesystem tools, NOT this git info.")

	return b.String()
}

// parseGitRemote extracts owner and repo from a git remote URL.
// Supports: https://github.com/owner/repo.git, git@github.com:owner/repo.git
func parseGitRemote(url string) (owner, repo string) {
	url = strings.TrimSpace(url)
	url = strings.TrimSuffix(url, ".git")

	// SSH format: git@github.com:owner/repo
	if strings.HasPrefix(url, "git@") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			pathParts := strings.Split(parts[1], "/")
			if len(pathParts) >= 2 {
				return pathParts[len(pathParts)-2], pathParts[len(pathParts)-1]
			}
		}
		return "", ""
	}

	// HTTPS format: https://github.com/owner/repo
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	parts := strings.Split(url, "/")
	// github.com/owner/repo → parts[0]=github.com, [1]=owner, [2]=repo
	if len(parts) >= 3 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}

	return "", ""
}
