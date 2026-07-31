// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type TelemetryPayload struct {
	APIKey     string `json:"api_key"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	StackTrace string `json:"stack_trace"`
}

// Middleware returns an HTTP middleware that catches panics and asynchronously
// sends crash telemetry to the specified Triage engine URL.
func Middleware(apiKey string, engineURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rerr := recover(); rerr != nil {
					stack := debug.Stack()
					stackStr := string(stack)

					file, line := parseTopApplicationFrame(stackStr)

					// Fire async, non-blocking HTTP POST request to engineURL
					go sendTelemetry(engineURL, apiKey, file, line, stackStr)

					// Respond with 500 Internal Server Error
					http.Error(w, fmt.Sprintf("Internal Server Error: %v", rerr), http.StatusInternalServerError)
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
				strings.Contains(line, "triage/sdk") ||
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
