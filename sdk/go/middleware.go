// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package triage

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const DefaultEngineURL = "https://triage.algotyrnt.com/api/v1/telemetry"

type middlewareConfig struct {
	gatewayURL string
	commit     string
	owner      string
	repo       string
}

type Option func(*middlewareConfig)

// WithGatewayURL allows self-hosted deployments to specify a custom Manager Gateway URL.
func WithGatewayURL(url string) Option {
	return func(c *middlewareConfig) {
		if url != "" {
			c.gatewayURL = url
		}
	}
}

// WithCommit sets the Git commit SHA for on-demand source code resolution.
func WithCommit(commit string) Option {
	return func(c *middlewareConfig) {
		if commit != "" {
			c.commit = commit
		}
	}
}

// WithRepo sets the repository owner and name (e.g. "owner/repo" or ("owner", "repo")).
func WithRepo(ownerRepo string) Option {
	return func(c *middlewareConfig) {
		if ownerRepo != "" {
			parts := strings.Split(ownerRepo, "/")
			if len(parts) == 2 {
				c.owner = parts[0]
				c.repo = parts[1]
			}
		}
	}
}

func getVCSCommit() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return ""
}

type TelemetryPayload struct {
	APIKey     string `json:"api_key"`
	Owner      string `json:"owner,omitempty"`
	Repo       string `json:"repo,omitempty"`
	Commit     string `json:"commit,omitempty"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	StackTrace string `json:"stack_trace"`
	TraceID    string `json:"trace_id,omitempty"`
}

type TelemetryJob struct {
	EngineURL  string
	APIKey     string
	Owner      string
	Repo       string
	Commit     string
	File       string
	Line       int
	StackTrace string
	TraceID    string
}

var (
	telemetryQueue = make(chan TelemetryJob, 1000)
	queuedCount    uint64
	processedCount uint64
	droppedCount   uint64
)

func init() {
	// Fixed worker pool for async telemetry dispatch
	for i := 0; i < 4; i++ {
		go func() {
			for job := range telemetryQueue {
				atomic.AddUint64(&processedCount, 1)
				sendTelemetry(job.EngineURL, job.APIKey, job.Owner, job.Repo, job.Commit, job.File, job.Line, job.StackTrace, job.TraceID)
			}
		}()
	}
}

// GetTelemetryMetrics returns current telemetry queue statistics
func GetTelemetryMetrics() (queued uint64, processed uint64, dropped uint64) {
	return atomic.LoadUint64(&queuedCount), atomic.LoadUint64(&processedCount), atomic.LoadUint64(&droppedCount)
}

func generateTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func isValidTraceID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, ch := range id {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '_' || ch == '-') {
			return false
		}
	}
	return true
}

func parseOrGenerateTraceID(r *http.Request) string {
	if r != nil {
		if triageHeader := r.Header.Get("X-Triage-Trace-ID"); triageHeader != "" {
			if isValidTraceID(triageHeader) {
				return triageHeader
			}
		}
		if tp := r.Header.Get("traceparent"); tp != "" {
			parts := strings.Split(tp, "-")
			if len(parts) == 4 && len(parts[1]) == 32 {
				if isValidTraceID(parts[1]) {
					return parts[1]
				}
			}
		}
	}
	return generateTraceID()
}

func enqueueTelemetry(job TelemetryJob) {
	select {
	case telemetryQueue <- job:
		atomic.AddUint64(&queuedCount, 1)
	default:
		// Bounded queue full: drop non-blockingly to protect application runtime performance
		atomic.AddUint64(&droppedCount, 1)
	}
}

// Middleware returns an HTTP middleware wrapping standard net/http routers.
// Managed Usage: triage.Middleware(apiKey, triage.WithRepo("algotyrnt/triage"))
// Self-Hosted Usage: triage.Middleware(apiKey, triage.WithGatewayURL("https://triage.internal.com/api/telemetry"))
func Middleware(apiKey string, opts ...Option) func(http.Handler) http.Handler {
	cfg := &middlewareConfig{
		gatewayURL: DefaultEngineURL,
		commit:     getVCSCommit(),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rerr := recover(); rerr != nil {
					stack := debug.Stack()
					stackStr := string(stack)

					file, line := parseTopApplicationFrame(stackStr)
					traceID := parseOrGenerateTraceID(r)

					// Non-blocking enqueue to bounded telemetry worker pool
					enqueueTelemetry(TelemetryJob{
						EngineURL:  cfg.gatewayURL,
						APIKey:     apiKey,
						Owner:      cfg.owner,
						Repo:       cfg.repo,
						Commit:     cfg.commit,
						File:       file,
						Line:       line,
						StackTrace: stackStr,
						TraceID:    traceID,
					})

					// Respond with generic 500 Internal Server Error (no internal error leakage)
					w.Header().Set("X-Triage-Trace-ID", traceID)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func parseTopApplicationFrame(stackTrace string) (string, int) {
	lines := strings.Split(stackTrace, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "/") || strings.Contains(line, ".go:") {
			// Ignore runtime internal frames, standard library web server frames, and middleware frames
			if strings.Contains(line, "runtime/") ||
				strings.Contains(line, "net/http/") ||
				strings.Contains(line, "github.com/algotyrnt/triage/sdk/go") ||
				strings.Contains(line, "middleware.go") {
				continue
			}
			// Line format: /path/to/file.go:line +0x...
			parts := strings.Split(line, " ")
			if len(parts) > 0 {
				fileAndLine := parts[0]
				colonIdx := strings.LastIndex(fileAndLine, ":")
				if colonIdx != -1 {
					file := fileAndLine[:colonIdx]
					lineNumStr := fileAndLine[colonIdx+1:]
					if lineNum, err := strconv.Atoi(lineNumStr); err == nil {
						return file, lineNum
					}
				}
			}
		}
	}
	return "", 0
}

func sendTelemetry(engineURL string, apiKey string, owner string, repo string, commit string, file string, line int, stackTrace string, traceID string) {
	payload := TelemetryPayload{
		APIKey:     apiKey,
		Owner:      owner,
		Repo:       repo,
		Commit:     commit,
		File:       file,
		Line:       line,
		StackTrace: stackTrace,
		TraceID:    traceID,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodPost, engineURL, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if traceID != "" {
		req.Header.Set("X-Triage-Trace-ID", traceID)
		req.Header.Set("traceparent", fmt.Sprintf("00-%s-0000000000000001-01", traceID))
	}

	resp, err := client.Do(req)
	if err == nil && resp != nil {
		_ = resp.Body.Close()
	}
}
