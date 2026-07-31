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
