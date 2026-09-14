// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"runtime/debug"
	"testing"
)

func TestGet(t *testing.T) {
	v := Get()
	if v == "" {
		t.Fatalf("expected non-empty version")
	}
}

func TestPopulateFromBuildInfo(t *testing.T) {
	// Save globals to restore after test
	origVersion := Version
	origCommit := Commit
	origDate := Date
	defer func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	}()

	// 1. nil build info
	populateFromBuildInfo(nil)

	// 2. Build info with custom version
	Version = "dev"
	Commit = "none"
	Date = "unknown"

	info := &debug.BuildInfo{
		Main: debug.Module{
			Version: "v1.2.3",
		},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "0123456789abcdef"},
			{Key: "vcs.time", Value: "2026-09-14T00:00:00Z"},
		},
	}

	populateFromBuildInfo(info)
	if Version != "v1.2.3" {
		t.Errorf("expected Version 'v1.2.3', got %s", Version)
	}
	if Commit != "0123456" {
		t.Errorf("expected shortened Commit '0123456', got %s", Commit)
	}
	if Date != "2026-09-14T00:00:00Z" {
		t.Errorf("expected Date '2026-09-14T00:00:00Z', got %s", Date)
	}

	// 3. Short commit <= 7 chars
	Commit = "none"
	shortInfo := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc"},
		},
	}
	populateFromBuildInfo(shortInfo)
	if Commit != "abc" {
		t.Errorf("expected Commit 'abc', got %s", Commit)
	}

	// 4. (devel) version does not overwrite dev
	Version = "dev"
	develInfo := &debug.BuildInfo{
		Main: debug.Module{
			Version: "(devel)",
		},
	}
	populateFromBuildInfo(develInfo)
	if Version != "dev" {
		t.Errorf("expected Version 'dev' to remain, got %s", Version)
	}
}
