// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package config

// Options contains startup configuration for the Triage server.
type Options struct {
	DataDir  string
	Port     string
	LogLevel string
}

// DefaultOptions returns the zero-config startup defaults for embedded SQLite storage.
func DefaultOptions() *Options {
	return &Options{
		DataDir:  "data",
		Port:     "8080",
		LogLevel: "info",
	}
}

// DatabasePath returns the path to the embedded SQLite database file.
func (o *Options) DatabasePath() string {
	if o.DataDir == "" {
		return "data/triage.db"
	}
	return o.DataDir + "/triage.db"
}
