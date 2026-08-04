// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package triage

import (
	"bytes"
	"encoding/json"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type TelemetryPayload struct {
	APIKey     string `json:"api_key"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	StackTrace string `json:"stack_trace"`
}

type TelemetryJob struct {
	EngineURL  string
	APIKey     string
	File       string
	Line       int
	StackTrace string
}

var (
	telemetryQueue = make(chan TelemetryJob, 100)
	queuedCount    uint64
	processedCount uint64
	droppedCount   uint64
)

func init() {
	// Start fixed worker pool for async telemetry dispatch
	for i := 0; i < 4; i++ {
		go func() {
			for job := range telemetryQueue {
				atomic.AddUint64(&processedCount, 1)
				sendTelemetry(job.EngineURL, job.APIKey, job.File, job.Line, job.StackTrace)
			}
		}()
	}
}

// GetTelemetryMetrics returns current telemetry queue statistics
func GetTelemetryMetrics() (queued uint64, processed uint64, dropped uint64) {
	return atomic.LoadUint64(&queuedCount), atomic.LoadUint64(&processedCount), atomic.LoadUint64(&droppedCount)
}

func enqueueTelemetry(job TelemetryJob) {
	select {
	case telemetryQueue <- job:
		atomic.AddUint64(&queuedCount, 1)
	default:
		// Queue full: drop non-blockingly to protect application runtime performance
		atomic.AddUint64(&droppedCount, 1)
	}
}

// Middleware returns an HTTP middleware that catches panics and asynchronously
// queues crash telemetry for the specified Triage engine URL.
func Middleware(apiKey string, engineURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rerr := recover(); rerr != nil {
					stack := debug.Stack()
					stackStr := string(stack)

					file, line := parseTopApplicationFrame(stackStr)

					// Non-blocking enqueue to bounded telemetry worker pool
					enqueueTelemetry(TelemetryJob{
						EngineURL:  engineURL,
						APIKey:     apiKey,
						File:       file,
						Line:       line,
						StackTrace: stackStr,
					})

					// Respond with generic 500 Internal Server Error (no internal error leakage)
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

func sendTelemetry(engineURL string, apiKey string, file string, line int, stackTrace string) {
	payload := TelemetryPayload{
		APIKey:     apiKey,
		File:       file,
		Line:       line,
		StackTrace: stackTrace,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", engineURL, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err == nil && resp != nil {
		_ = resp.Body.Close()
	}
}
