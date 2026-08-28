// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"strings"
	"time"
)

type GitHubIssueMarkdownParams struct {
	IncidentID   string
	Owner        string
	Repo         string
	File         string
	Line         int
	PanicMessage string
	Severity     string
	Status       string
	CreatedAt    time.Time
	TraceID      string
	RootCause    string
	SuggestedFix string
	ASTSnippet   string
	StackTrace   string
	AppURL       string
}

type GitHubPRMarkdownParams struct {
	IncidentID    string
	Owner         string
	Repo          string
	File          string
	Line          int
	PanicMessage  string
	IssueNumber   int
	RootCause     string
	SuggestedFix  string
	PatchCode     string
	AppURL        string
	DefaultBranch string
}

// BuildGitHubIssueBody constructs a structured, high-clarity GitHub Issue description in Markdown.
func BuildGitHubIssueBody(p GitHubIssueMarkdownParams) string {
	var b strings.Builder

	b.WriteString("## Runtime Go Panic Intercepted\n\n")
	if p.AppURL != "" {
		b.WriteString(fmt.Sprintf("> An unhandled runtime panic was intercepted non-blockingly by **[Triage](%s)**. Root-cause isolation, cross-file AST context, and synthesized diagnostics are detailed below.\n\n", p.AppURL))
	} else {
		b.WriteString("> An unhandled runtime panic was intercepted non-blockingly by **Triage**. Root-cause isolation, cross-file AST context, and synthesized diagnostics are detailed below.\n\n")
	}

	b.WriteString("### Incident Summary\n\n")
	b.WriteString("| Attribute | Value |\n")
	b.WriteString("| :--- | :--- |\n")

	crashSite := fmt.Sprintf("`%s:%d`", p.File, p.Line)
	if p.Owner != "" && p.Repo != "" && p.File != "" && p.Line > 0 {
		cleanFile := strings.TrimPrefix(p.File, "./")
		cleanFile = strings.TrimPrefix(cleanFile, "/")
		crashSite = fmt.Sprintf("[`%s:%d`](https://github.com/%s/%s/blob/main/%s#L%d)", p.File, p.Line, p.Owner, p.Repo, cleanFile, p.Line)
	}
	b.WriteString(fmt.Sprintf("| **Crash Site** | %s |\n", crashSite))

	if p.IncidentID != "" {
		if p.AppURL != "" {
			b.WriteString(fmt.Sprintf("| **Incident ID** | [`%s`](%s/?incident=%s) |\n", p.IncidentID, p.AppURL, p.IncidentID))
		} else {
			b.WriteString(fmt.Sprintf("| **Incident ID** | `%s` |\n", p.IncidentID))
		}
	}

	if p.TraceID != "" && p.TraceID != p.IncidentID {
		b.WriteString(fmt.Sprintf("| **Trace ID** | `%s` |\n", p.TraceID))
	}

	ts := p.CreatedAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	b.WriteString(fmt.Sprintf("| **Timestamp** | `%s UTC` |\n", ts.UTC().Format("2006-01-02 15:04:05")))

	if p.Severity != "" {
		b.WriteString(fmt.Sprintf("| **Severity** | `%s` |\n", p.Severity))
	}

	status := p.Status
	if status == "" {
		status = "OPEN"
	}
	b.WriteString(fmt.Sprintf("| **Status** | `%s` |\n\n", status))

	if p.PanicMessage != "" {
		b.WriteString("---\n\n### Panic Message\n```\n")
		b.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(p.PanicMessage)))
		b.WriteString("```\n\n")
	}

	if p.RootCause != "" {
		b.WriteString("---\n\n### AI Root Cause Analysis\n")
		b.WriteString(fmt.Sprintf("> %s\n\n", strings.ReplaceAll(strings.TrimSpace(p.RootCause), "\n", "\n> ")))
	}

	if p.SuggestedFix != "" {
		b.WriteString("---\n\n### Recommended Resolution\n")
		b.WriteString(fmt.Sprintf("> %s\n\n", strings.ReplaceAll(strings.TrimSpace(p.SuggestedFix), "\n", "\n> ")))
		if p.AppURL != "" {
			fixURL := fmt.Sprintf("%s/?incident=%s", p.AppURL, p.IncidentID)
			b.WriteString(fmt.Sprintf("[**Generate Bugfix PR via Triage Studio**](%s)\n\n", fixURL))
		}
	}

	if p.ASTSnippet != "" {
		b.WriteString("---\n\n### Isolated Function AST Context\n")
		b.WriteString(fmt.Sprintf("<details open>\n<summary><b>View Isolated Code Context around line %d</b></summary>\n\n", p.Line))
		b.WriteString("```go\n")
		b.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(p.ASTSnippet)))
		b.WriteString("```\n</details>\n\n")
	}

	if p.StackTrace != "" {
		b.WriteString("---\n\n### Goroutine Stack Trace\n")
		b.WriteString("<details>\n<summary><b>View Complete Stack Trace</b></summary>\n\n")
		b.WriteString("```\n")
		b.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(p.StackTrace)))
		b.WriteString("```\n</details>\n\n")
	}

	b.WriteString("---\n")
	if p.AppURL != "" {
		b.WriteString(fmt.Sprintf("<sub>Intercepted and filed automatically by <a href=\"%s\"><b>Triage Engine</b></a> — Zero-overhead Go panic isolation, AST diagnostics and automated bugfix PRs.</sub>\n", p.AppURL))
	} else {
		b.WriteString("<sub>Intercepted and filed automatically by <b>Triage Engine</b> — Zero-overhead Go panic isolation, AST diagnostics and automated bugfix PRs.</sub>\n")
	}

	return b.String()
}

// BuildGitHubPRBody constructs a clean, comprehensive Pull Request description in Markdown.
func BuildGitHubPRBody(p GitHubPRMarkdownParams) string {
	var b strings.Builder

	b.WriteString("## Automated Bugfix Pull Request\n\n")
	if p.IssueNumber > 0 {
		b.WriteString(fmt.Sprintf("Closes #%d\n\n", p.IssueNumber))
	}

	if p.AppURL != "" {
		b.WriteString(fmt.Sprintf("This Pull Request was generated automatically by **[Triage](%s)** to fix a runtime panic intercepted in `%s:%d`.\n\n", p.AppURL, p.File, p.Line))
	} else {
		b.WriteString(fmt.Sprintf("This Pull Request was generated automatically by **Triage** to fix a runtime panic intercepted in `%s:%d`.\n\n", p.File, p.Line))
	}

	b.WriteString("---\n\n### Incident Details\n\n")
	b.WriteString("| Attribute | Value |\n")
	b.WriteString("| :--- | :--- |\n")

	crashSite := fmt.Sprintf("`%s:%d`", p.File, p.Line)
	if p.Owner != "" && p.Repo != "" && p.File != "" && p.Line > 0 {
		cleanFile := strings.TrimPrefix(p.File, "./")
		cleanFile = strings.TrimPrefix(cleanFile, "/")
		crashSite = fmt.Sprintf("[`%s:%d`](https://github.com/%s/%s/blob/main/%s#L%d)", p.File, p.Line, p.Owner, p.Repo, cleanFile, p.Line)
	}
	b.WriteString(fmt.Sprintf("| **Crash Site** | %s |\n", crashSite))

	if p.IncidentID != "" {
		if p.AppURL != "" {
			b.WriteString(fmt.Sprintf("| **Incident ID** | [`%s`](%s/?incident=%s) |\n", p.IncidentID, p.AppURL, p.IncidentID))
		} else {
			b.WriteString(fmt.Sprintf("| **Incident ID** | `%s` |\n", p.IncidentID))
		}
	}

	if p.PanicMessage != "" {
		cleanPanic := strings.ReplaceAll(strings.TrimSpace(p.PanicMessage), "\n", " ")
		b.WriteString(fmt.Sprintf("| **Panic Message** | `%s` |\n", cleanPanic))
	}

	if p.IssueNumber > 0 {
		b.WriteString(fmt.Sprintf("| **Linked Issue** | Closes #%d |\n", p.IssueNumber))
	}

	b.WriteString("\n")

	if p.RootCause != "" {
		b.WriteString("---\n\n### AI Root Cause Analysis\n")
		b.WriteString(fmt.Sprintf("> %s\n\n", strings.ReplaceAll(strings.TrimSpace(p.RootCause), "\n", "\n> ")))
	}

	if p.SuggestedFix != "" {
		b.WriteString("---\n\n### Recommended Fix Strategy\n")
		b.WriteString(fmt.Sprintf("> %s\n\n", strings.ReplaceAll(strings.TrimSpace(p.SuggestedFix), "\n", "\n> ")))
	}

	if p.PatchCode != "" {
		b.WriteString("---\n\n### Applied Patch Diff\n\n")
		b.WriteString("```diff\n")
		b.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(p.PatchCode)))
		b.WriteString("```\n\n")
	}

	b.WriteString("---\n\n### Reviewer Verification Checklist\n")
	b.WriteString("- [ ] Review code change against boundary and nullability edge cases\n")
	b.WriteString("- [ ] Verify local tests pass (`go test ./...`)\n")
	b.WriteString("- [ ] Merge to apply fix and resolve associated incident\n\n")

	b.WriteString("---\n")
	if p.AppURL != "" {
		b.WriteString(fmt.Sprintf("<sub>Generated automatically by <a href=\"%s\"><b>Triage Studio</b></a> — AI-powered panic isolation and automated patch generation.</sub>\n", p.AppURL))
	} else {
		b.WriteString("<sub>Generated automatically by <b>Triage Studio</b> — AI-powered panic isolation and automated patch generation.</sub>\n")
	}

	return b.String()
}
