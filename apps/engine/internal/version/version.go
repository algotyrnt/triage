// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"runtime/debug"
)

var (
	// Version is populated at build time via -ldflags "-X main.version=..."
	// or dynamically discovered via runtime/debug.ReadBuildInfo().
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		if Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
		if Commit == "none" {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					Commit = setting.Value
					if len(Commit) > 7 {
						Commit = Commit[:7]
					}
				}
				if setting.Key == "vcs.time" {
					Date = setting.Value
				}
			}
		}
	}
}

// Get returns the current version string.
func Get() string {
	return Version
}
