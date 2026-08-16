---
title: AST Engine & Node Slicing
description: How Triage uses Go's AST parser to isolate function blocks on demand
---

Triage's core architectural differentiator is **on-demand AST slicing**. Rather than pre-indexing whole repositories or passing full source files to LLMs, Triage extracts only the enclosing `*ast.FuncDecl` node for the panicking line.

## Why AST Slicing?

When a Go panic occurs, passing the full source file to an LLM creates multiple problems:

1. **Token Cost:** A 1,500-line file uses ~3,000 tokens per incident.
2. **Context Dilution:** Unrelated functions in the same file confuse LLM inference.
3. **Speed:** Parsing only the relevant function block speeds up inference by 3x.

By extracting only the `*ast.FuncDecl` containing the panic, Triage achieves a **94% token reduction** (reducing payload size to 100–250 tokens) while preserving complete semantic context.

---

## AST Resolution Pipeline

```
1. Panic Telemetry arrives (file + line + commit_sha)
              │
              ▼
2. 3-Tier Layered Cache Lookup
   ├── Tier 1: In-Memory KV Cache     (<1.5ms)
   ├── Tier 2: PostgreSQL ast_nodes   (<5ms)
   └── Tier 3: GitHub Contents API    (<25ms)
              │
              ▼
3. go/parser & go/ast AST Walker
   └── Find *ast.FuncDecl where Pos <= Line <= End
              │
              ▼
4. Formatted AST Code Snippet (10-30 lines)
```

---

## The Go Parser Internals

The Triage Engine parses source files using Go's official standard packages:

- `go/parser`: `parser.ParseFile(fset, filename, src, parser.ParseComments)`
- `go/ast`: `ast.Inspect(node, func(n ast.Node) bool)`

### AST Node Traversal Algorithm

```go
func ExtractEnclosingFunc(src []byte, targetLine int) (*ast.FuncDecl, []string, error) {
    fset := token.NewFileSet()
    fileNode, err := parser.ParseFile(fset, "", src, parser.ParseComments)
    if err != nil {
        return nil, nil, err
    }

    var matchedFunc *ast.FuncDecl
    ast.Inspect(fileNode, func(n ast.Node) bool {
        if fn, ok := n.(*ast.FuncDecl); ok {
            startLine := fset.Position(fn.Pos()).Line
            endLine := fset.Position(fn.End()).Line

            if targetLine >= startLine && targetLine <= endLine {
                matchedFunc = fn
                return false // Found target function
            }
        }
        return true
    })

    return matchedFunc, extractLines(src, matchedFunc), nil
}
```

---

## Monorepo Subdirectory Resolution

In monorepos where Go backends live in subfolders (e.g. `backend`, `apps/api`), runtime panic stack traces may reference files either relative to the module root (e.g. `handlers/payment.go`) or the repository root (`backend/handlers/payment.go`).

Triage handles this transparently:

1. **Path Normalization:** The engine normalizes candidate paths using the project's registered `root_dir`, ensuring files are accurately fetched from the GitHub Contents API without path mismatch.
2. **Dual-Key AST Indexing:** When indexing monorepo AST nodes, Triage indexes both the repository-relative path and the module-relative path in PostgreSQL, guaranteeing $O(1)$ symbolication lookups regardless of how the runtime frame is formatted.

---

## Pre-Indexing (Optional)

While Triage resolves AST nodes dynamically on demand, you can also pre-index entire repositories via the API:

```bash
curl -X POST http://localhost:8080/api/v1/ast/index \
  -H "Authorization: Bearer $TRIAGE_SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "repo": "myorg/myrepo",
    "root_dir": "backend",
    "commit_sha": "main"
  }'
```

This populates the `ast_nodes` PostgreSQL table for instant `<5ms` queries.
