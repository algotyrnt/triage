// Copyright 2026 Punjitha Bandara (algotyrnt) <https://algotyrnt.com>
// SPDX-License-Identifier: Apache-2.0

package ast

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

// ExtractFuncAST parses a local .go file and extracts the string representation
// of the *ast.FuncDecl enclosing the specified targetLine.
func ExtractFuncAST(filePath string, targetLine int) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	var targetFunc *ast.FuncDecl
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		startLine := fset.Position(fn.Pos()).Line
		endLine := fset.Position(fn.End()).Line

		if targetLine >= startLine && targetLine <= endLine {
			targetFunc = fn
			break
		}
	}

	if targetFunc == nil {
		return "", fmt.Errorf("no function declaration found surrounding line %d in %s", targetLine, filePath)
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, targetFunc); err != nil {
		return "", fmt.Errorf("failed to print AST node: %w", err)
	}

	return buf.String(), nil
}
