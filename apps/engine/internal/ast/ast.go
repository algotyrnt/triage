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

type ExtractedFunction struct {
	Name      string
	StartLine int
	EndLine   int
	Snippet   string
}

func parseGoFile(filePath string) (*token.FileSet, *ast.File, error) {
	if filePath == "" {
		return nil, nil, fmt.Errorf("file path cannot be empty")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	return fset, node, nil
}

func extractFunctions(fset *token.FileSet, node *ast.File) []ExtractedFunction {
	var result []ExtractedFunction
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		startLine := fset.Position(fn.Pos()).Line
		endLine := fset.Position(fn.End()).Line

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, fn); err != nil {
			continue
		}

		name := fn.Name.Name
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			var recvBuf bytes.Buffer
			if err := printer.Fprint(&recvBuf, fset, fn.Recv.List[0].Type); err == nil {
				name = fmt.Sprintf("%s.%s", recvBuf.String(), name)
			} else {
				name = fmt.Sprintf("%v.%s", fn.Recv.List[0].Type, name)
			}
		}

		result = append(result, ExtractedFunction{
			Name:      name,
			StartLine: startLine,
			EndLine:   endLine,
			Snippet:   buf.String(),
		})
	}
	return result
}

// ExtractFuncAST parses a local .go file and extracts the string representation
// of the *ast.FuncDecl enclosing the specified targetLine.
func ExtractFuncAST(filePath string, targetLine int) (string, error) {
	fset, node, err := parseGoFile(filePath)
	if err != nil {
		return "", err
	}

	for _, fn := range extractFunctions(fset, node) {
		if targetLine >= fn.StartLine && targetLine <= fn.EndLine {
			return fn.Snippet, nil
		}
	}

	return "", fmt.Errorf("no function declaration found surrounding line %d in %s", targetLine, filePath)
}
