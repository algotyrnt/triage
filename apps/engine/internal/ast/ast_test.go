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

package ast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractFuncAST(t *testing.T) {
	tempDir := t.TempDir()
	sampleFile := filepath.Join(tempDir, "sample.go")

	sampleCode := `package sample

import "fmt"

func HelperFunc() string {
	return "hello"
}

func TargetFunc(val *int) int {
	if val == nil {
		panic("nil pointer")
	}
	return *val
}
`

	err := os.WriteFile(sampleFile, []byte(sampleCode), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Line 11 is inside TargetFunc
	astStr, err := ExtractFuncAST(sampleFile, 11)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(astStr, "func TargetFunc(val *int) int") {
		t.Errorf("expected astStr to contain TargetFunc, got:\n%s", astStr)
	}

	if strings.Contains(astStr, "HelperFunc") {
		t.Errorf("expected astStr NOT to contain HelperFunc, got:\n%s", astStr)
	}
}
