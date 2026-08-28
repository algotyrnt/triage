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
	"path/filepath"
	"sort"
	"strings"
)

// ExtractedFunction represents a single Go function declaration with source coordinates.
type ExtractedFunction struct {
	Name         string
	ReceiverType string
	FilePath     string
	StartLine    int
	EndLine      int
	Snippet      string
	Node         *ast.FuncDecl
}

// ExtractedType represents a struct, interface, or type alias definition.
type ExtractedType struct {
	Name      string
	FilePath  string
	StartLine int
	EndLine   int
	Snippet   string
	Node      *ast.TypeSpec
}

// ExtractedVar represents a package-level variable or constant declaration.
type ExtractedVar struct {
	Name      string
	FilePath  string
	StartLine int
	EndLine   int
	Snippet   string
}

// PackageContext holds indexed AST symbols across multiple files in a Go package.
type PackageContext struct {
	PackageName string
	Fset        *token.FileSet
	Files       map[string]*ast.File
	Functions   []ExtractedFunction
	Types       map[string]ExtractedType
	Vars        map[string]ExtractedVar
}

// NewPackageContext builds a symbol table across all provided AST files.
func NewPackageContext(fset *token.FileSet, pkgName string, files map[string]*ast.File) *PackageContext {
	ctx := &PackageContext{
		PackageName: pkgName,
		Fset:        fset,
		Files:       files,
		Types:       make(map[string]ExtractedType),
		Vars:        make(map[string]ExtractedVar),
	}

	for filePath, fileNode := range files {
		for _, decl := range fileNode.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				startLine := fset.Position(d.Pos()).Line
				endLine := fset.Position(d.End()).Line

				var buf bytes.Buffer
				if err := printer.Fprint(&buf, fset, d); err != nil {
					continue
				}

				name := d.Name.Name
				recvType := ""
				if d.Recv != nil && len(d.Recv.List) > 0 {
					recvType = getTypeName(d.Recv.List[0].Type)
					var recvBuf bytes.Buffer
					if err := printer.Fprint(&recvBuf, fset, d.Recv.List[0].Type); err == nil {
						name = fmt.Sprintf("%s.%s", recvBuf.String(), name)
					} else {
						name = fmt.Sprintf("%s.%s", recvType, name)
					}
				}

				ctx.Functions = append(ctx.Functions, ExtractedFunction{
					Name:         name,
					ReceiverType: recvType,
					FilePath:     filePath,
					StartLine:    startLine,
					EndLine:      endLine,
					Snippet:      buf.String(),
					Node:         d,
				})

			case *ast.GenDecl:
				startLine := fset.Position(d.Pos()).Line
				endLine := fset.Position(d.End()).Line

				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						var buf bytes.Buffer
						if err := printer.Fprint(&buf, fset, d); err != nil {
							continue
						}
						ctx.Types[s.Name.Name] = ExtractedType{
							Name:      s.Name.Name,
							FilePath:  filePath,
							StartLine: startLine,
							EndLine:   endLine,
							Snippet:   buf.String(),
							Node:      s,
						}

					case *ast.ValueSpec:
						var buf bytes.Buffer
						if err := printer.Fprint(&buf, fset, d); err != nil {
							continue
						}
						for _, name := range s.Names {
							ctx.Vars[name.Name] = ExtractedVar{
								Name:      name.Name,
								FilePath:  filePath,
								StartLine: startLine,
								EndLine:   endLine,
								Snippet:   buf.String(),
							}
						}
					}
				}
			}
		}
	}

	return ctx
}

func getTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return getTypeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.ArrayType:
		return getTypeName(t.Elt)
	case *ast.MapType:
		// Check value type first, or key type
		if val := getTypeName(t.Value); val != "" {
			return val
		}
		return getTypeName(t.Key)
	case *ast.ChanType:
		return getTypeName(t.Value)
	case *ast.Ellipsis:
		return getTypeName(t.Elt)
	default:
		return ""
	}
}

// FindEnclosingFunction locates the function enclosing the given line in the specified file.
func (ctx *PackageContext) FindEnclosingFunction(filePath string, targetLine int) (*ExtractedFunction, error) {
	cleanTarget := filepath.ToSlash(filePath)

	for i := range ctx.Functions {
		fn := &ctx.Functions[i]
		cleanFnPath := filepath.ToSlash(fn.FilePath)

		matchesFile := cleanFnPath == cleanTarget ||
			filepath.Base(cleanFnPath) == filepath.Base(cleanTarget) ||
			strings.HasSuffix(cleanTarget, cleanFnPath) ||
			strings.HasSuffix(cleanFnPath, cleanTarget)

		if matchesFile && targetLine >= fn.StartLine && targetLine <= fn.EndLine {
			return fn, nil
		}
	}

	return nil, fmt.Errorf("no function declaration found surrounding line %d in %s", targetLine, filePath)
}

// ResolveDependencies finds structs, constructors, receiver methods, helper functions,
// and package variables referenced by the target function.
func (ctx *PackageContext) ResolveDependencies(targetFn *ExtractedFunction) (types []ExtractedType, helpers []ExtractedFunction, vars []ExtractedVar) {
	referencedTypeNames := make(map[string]bool)
	referencedFuncNames := make(map[string]bool)
	referencedVarNames := make(map[string]bool)

	// 1. Include the receiver struct if it's a method
	if targetFn.ReceiverType != "" {
		referencedTypeNames[targetFn.ReceiverType] = true
	}

	// 2. Walk the target function AST to collect referenced identifiers
	ast.Inspect(targetFn.Node, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		switch expr := n.(type) {
		case *ast.Ident:
			name := expr.Name
			if _, exists := ctx.Types[name]; exists {
				referencedTypeNames[name] = true
			}
			if _, exists := ctx.Vars[name]; exists {
				referencedVarNames[name] = true
			}

		case *ast.CallExpr:
			if ident, ok := expr.Fun.(*ast.Ident); ok {
				referencedFuncNames[ident.Name] = true
			}
		}
		return true
	})

	// 3. Transitively resolve struct fields for referenced types (depth 2)
	expandedTypes := make(map[string]bool)
	for tName := range referencedTypeNames {
		expandedTypes[tName] = true
	}

	for tName := range referencedTypeNames {
		if typeDef, ok := ctx.Types[tName]; ok && typeDef.Node != nil {
			if structType, isStruct := typeDef.Node.Type.(*ast.StructType); isStruct && structType.Fields != nil {
				for _, field := range structType.Fields.List {
					fieldTypeName := getTypeName(field.Type)
					if fieldTypeName != "" {
						if _, exists := ctx.Types[fieldTypeName]; exists {
							expandedTypes[fieldTypeName] = true
						}
					}
				}
			}
		}
	}
	referencedTypeNames = expandedTypes

	// 4. Collect matched types (sorted for deterministic output)
	var typeList []ExtractedType
	for tName := range referencedTypeNames {
		if typeDef, ok := ctx.Types[tName]; ok {
			typeList = append(typeList, typeDef)
		}
	}
	sort.Slice(typeList, func(i, j int) bool {
		return typeList[i].Name < typeList[j].Name
	})

	// 5. Collect related constructors and helper functions
	constructorPrefixes := make(map[string]bool)
	if targetFn.ReceiverType != "" {
		constructorPrefixes["New"+targetFn.ReceiverType] = true
	}
	for tName := range referencedTypeNames {
		constructorPrefixes["New"+tName] = true
	}

	seenFuncs := make(map[string]bool)
	var helperList []ExtractedFunction

	for _, fn := range ctx.Functions {
		// Skip the target function itself
		if fn.FilePath == targetFn.FilePath && fn.StartLine == targetFn.StartLine {
			continue
		}

		baseName := fn.Node.Name.Name
		isHelper := referencedFuncNames[baseName]
		isConstructor := constructorPrefixes[baseName]

		if (isHelper || isConstructor) && !seenFuncs[fn.Name] {
			seenFuncs[fn.Name] = true
			helperList = append(helperList, fn)
		}
	}
	sort.Slice(helperList, func(i, j int) bool {
		return helperList[i].Name < helperList[j].Name
	})

	// 6. Collect matched package variables/constants
	var varList []ExtractedVar
	for vName := range referencedVarNames {
		if vDef, ok := ctx.Vars[vName]; ok {
			varList = append(varList, vDef)
		}
	}
	sort.Slice(varList, func(i, j int) bool {
		return varList[i].Name < varList[j].Name
	})

	return typeList, helperList, varList
}

// FormatContext formats the target crash function along with its resolved multi-file package context.
func (ctx *PackageContext) FormatContext(targetFn *ExtractedFunction, types []ExtractedType, helpers []ExtractedFunction, vars []ExtractedVar) string {
	var sb strings.Builder

	// Target Crash Function
	sb.WriteString("// ==============================================================================\n")
	sb.WriteString(fmt.Sprintf("// Crash Site: %s (lines %d-%d)\n", filepath.Base(targetFn.FilePath), targetFn.StartLine, targetFn.EndLine))
	sb.WriteString("// ==============================================================================\n")
	sb.WriteString(targetFn.Snippet)
	sb.WriteString("\n")

	// Struct & Type Definitions
	if len(types) > 0 {
		sb.WriteString("\n// ==============================================================================\n")
		sb.WriteString(fmt.Sprintf("// Package Context: Struct & Type Definitions (%s)\n", ctx.PackageName))
		sb.WriteString("// ==============================================================================\n")
		for _, t := range types {
			sb.WriteString(fmt.Sprintf("// File: %s (lines %d-%d)\n", filepath.Base(t.FilePath), t.StartLine, t.EndLine))
			sb.WriteString(t.Snippet)
			sb.WriteString("\n\n")
		}
	}

	// Related Constructors & Helper Functions
	if len(helpers) > 0 {
		sb.WriteString("\n// ==============================================================================\n")
		sb.WriteString(fmt.Sprintf("// Package Context: Related Constructors & Helper Functions (%s)\n", ctx.PackageName))
		sb.WriteString("// ==============================================================================\n")
		for _, h := range helpers {
			sb.WriteString(fmt.Sprintf("// File: %s (lines %d-%d)\n", filepath.Base(h.FilePath), h.StartLine, h.EndLine))
			sb.WriteString(h.Snippet)
			sb.WriteString("\n\n")
		}
	}

	// Package-Level Variables & Constants
	if len(vars) > 0 {
		sb.WriteString("\n// ==============================================================================\n")
		sb.WriteString(fmt.Sprintf("// Package Context: Package Variables & Constants (%s)\n", ctx.PackageName))
		sb.WriteString("// ==============================================================================\n")
		for _, v := range vars {
			sb.WriteString(fmt.Sprintf("// File: %s (lines %d-%d)\n", filepath.Base(v.FilePath), v.StartLine, v.EndLine))
			sb.WriteString(v.Snippet)
			sb.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(sb.String())
}

// ExtractFuncAST parses the package surrounding filePath, extracts the target function
// at targetLine, and enriches it with cross-file struct definitions, receiver methods,
// constructors, and helper functions.
func ExtractFuncAST(filePath string, targetLine int) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	dir := filepath.Dir(filePath)
	fset := token.NewFileSet()

	isTestFile := strings.HasSuffix(filePath, "_test.go")
	filter := func(info os.FileInfo) bool {
		if !strings.HasSuffix(info.Name(), ".go") {
			return false
		}
		if !isTestFile && strings.HasSuffix(info.Name(), "_test.go") {
			return false
		}
		return true
	}

	pkgs, err := parser.ParseDir(fset, dir, filter, parser.ParseComments)
	if err != nil || len(pkgs) == 0 {
		// Fallback: parse single file if directory parsing fails
		return extractSingleFileAST(filePath, targetLine)
	}

	// Find the package that contains our target file
	var targetPkg *ast.Package
	cleanTarget := filepath.ToSlash(filePath)

	for _, pkg := range pkgs {
		for p := range pkg.Files {
			if filepath.ToSlash(p) == cleanTarget || filepath.Base(p) == filepath.Base(cleanTarget) {
				targetPkg = pkg
				break
			}
		}
		if targetPkg != nil {
			break
		}
	}

	if targetPkg == nil {
		for _, pkg := range pkgs {
			targetPkg = pkg
			break
		}
	}

	pkgCtx := NewPackageContext(fset, targetPkg.Name, targetPkg.Files)
	targetFn, err := pkgCtx.FindEnclosingFunction(filePath, targetLine)
	if err != nil {
		return "", err
	}

	types, helpers, vars := pkgCtx.ResolveDependencies(targetFn)
	return pkgCtx.FormatContext(targetFn, types, helpers, vars), nil
}

// ExtractFuncASTFromBytes parses raw .go source bytes in memory and extracts
// the target function along with type definitions and helpers defined in the same source.
func ExtractFuncASTFromBytes(content []byte, targetLine int) (string, error) {
	if len(content) == 0 {
		return "", fmt.Errorf("file content is empty")
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "src.go", content, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("failed to parse Go AST from bytes: %w", err)
	}

	files := map[string]*ast.File{
		"src.go": node,
	}

	pkgCtx := NewPackageContext(fset, node.Name.Name, files)
	targetFn, err := pkgCtx.FindEnclosingFunction("src.go", targetLine)
	if err != nil {
		return "", err
	}

	types, helpers, vars := pkgCtx.ResolveDependencies(targetFn)
	return pkgCtx.FormatContext(targetFn, types, helpers, vars), nil
}

// ExtractPackageContextASTFromBytes parses a collection of raw Go files in memory
// (e.g. from GitHub API or multi-file archives) and returns the enriched multi-file AST context.
func ExtractPackageContextASTFromBytes(files map[string][]byte, targetFile string, targetLine int) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files provided for package AST extraction")
	}

	fset := token.NewFileSet()
	astFiles := make(map[string]*ast.File)
	pkgName := ""

	for name, content := range files {
		node, err := parser.ParseFile(fset, name, content, parser.ParseComments)
		if err != nil {
			continue
		}
		if pkgName == "" {
			pkgName = node.Name.Name
		}
		astFiles[name] = node
	}

	if len(astFiles) == 0 {
		return "", fmt.Errorf("failed to parse any Go AST files from provided bytes")
	}

	pkgCtx := NewPackageContext(fset, pkgName, astFiles)
	targetFn, err := pkgCtx.FindEnclosingFunction(targetFile, targetLine)
	if err != nil {
		return "", err
	}

	types, helpers, vars := pkgCtx.ResolveDependencies(targetFn)
	return pkgCtx.FormatContext(targetFn, types, helpers, vars), nil
}

func extractSingleFileAST(filePath string, targetLine int) (string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	files := map[string]*ast.File{
		filePath: node,
	}

	pkgCtx := NewPackageContext(fset, node.Name.Name, files)
	targetFn, err := pkgCtx.FindEnclosingFunction(filePath, targetLine)
	if err != nil {
		return "", err
	}

	types, helpers, vars := pkgCtx.ResolveDependencies(targetFn)
	return pkgCtx.FormatContext(targetFn, types, helpers, vars), nil
}
