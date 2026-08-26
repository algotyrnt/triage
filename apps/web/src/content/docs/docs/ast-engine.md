---
title: AST Engine & Package Context Slicing
description: How Triage parses Go packages and extracts cross-file struct definitions, receiver methods, constructors, and helpers
---

Triage's core architectural differentiator is **selective on-demand AST slicing with multi-file package context**. Rather than sending monolithic files or losing context with single-function fragments, Triage isolates the crash site and automatically resolves its cross-file type definitions, receiver structs, constructors, and package helpers.

## Why Multi-File AST Slicing?

When a Go runtime panic occurs (such as a `nil pointer dereference`, `nil map write`, or `slice bounds out of range`), passing only a single function declaration or an entire source file creates significant diagnostic problems:

1. **Single-Function Blindness:** If a crash happens on `user.Profile.Settings.Theme = "dark"` in `handler.go`, but `User`, `Profile`, and `Settings` are defined in `models.go`, an isolated function snippet gives the LLM zero visibility into which fields are pointers (`*Profile`), value types (`Profile`), or uninitialized structures.
2. **Constructor & Initializer Gaps:** If a method `func (s *Service) Process()` panics on `s.cache.Set(...)`, the root cause is almost always an uninitialized field in `NewService()` in another file (`service.go` or `factory.go`).
3. **Token Cost & Dilution of Full Files:** Ingesting 2,000-line source files wastes thousands of tokens per incident and dilutes the model's attention with unrelated business logic.

Triage solves this by combining Go's official `go/parser` and `go/ast` packages to extract **the exact crashing function plus its selectively pruned package-level dependencies**, achieving over **90% token reduction** while delivering 100% semantic clarity.

---

## AST Resolution Pipeline

```
1. Panic Telemetry arrives (file + line + commit_sha)
              │
              ▼
2. 3-Tier Layered Cache Lookup
   ├── Tier 1: In-Memory KV Cache     (< 1.5ms)
   ├── Tier 2: PostgreSQL ast_nodes   (< 5ms)
   └── Tier 3: GitHub Contents API    (< 25ms)
              │
              ▼
3. Package AST Symbol Resolver (go/parser & go/ast)
   ├── Parse all package files in directory (parser.ParseDir)
   ├── Locate crashing *ast.FuncDecl enclosing targetLine
   ├── Resolve receiver struct (e.g. type UserHandler struct)
   ├── Transitive struct field resolution (depth 2 across pointers/slices/maps)
   ├── Locate associated constructors (New<Receiver>, New<Type>)
   └── Extract called package-level helpers and variables
              │
              ▼
4. Formatted Multi-File AST Context Snippet
```

---

## The Package Context Internals

The Triage Engine parses source packages using standard Go AST tooling:

- `go/parser`: `parser.ParseDir(fset, dir, filter, parser.ParseComments)`
- `go/ast`: `ast.Inspect(node, func(n ast.Node) bool)`
- `go/printer`: `printer.Fprint(&buf, fset, decl)`

### Dependency Resolution Algorithm

When a panic occurs, Triage builds a `PackageContext` symbol table and selectively extracts referenced symbols:

```go
// 1. Receiver Struct Resolution
if targetFn.ReceiverType != "" {
    referencedTypeNames[targetFn.ReceiverType] = true
}

// 2. AST Identifier Traversal
ast.Inspect(targetFn.Node, func(n ast.Node) bool {
    switch expr := n.(type) {
    case *ast.Ident:
        if _, exists := ctx.Types[expr.Name]; exists {
            referencedTypeNames[expr.Name] = true
        }
        if _, exists := ctx.Vars[expr.Name]; exists {
            referencedVarNames[expr.Name] = true
        }
    case *ast.CallExpr:
        if ident, ok := expr.Fun.(*ast.Ident); ok {
            referencedFuncNames[ident.Name] = true
        }
    }
    return true
})

// 3. Transitive Struct Field Traversal (depth 2)
for tName := range referencedTypeNames {
    if typeDef, ok := ctx.Types[tName]; ok {
        if structType, ok := typeDef.Node.Type.(*ast.StructType); ok {
            for _, field := range structType.Fields.List {
                if fieldType := getTypeName(field.Type); fieldType != "" {
                    referencedTypeNames[fieldType] = true
                }
            }
        }
    }
}
```

---

## Formatted AST Context Output Example

Here is what the Triage Engine generates and passes to the configured AI model:

```go
// ==============================================================================
// Crash Site: user.go (lines 25-32)
// ==============================================================================
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
    if validateRequest("update") {
        h.Notifier.WebhookURL = "https://hooks.slack.com/services/..."
    }
}

// ==============================================================================
// Package Context: Struct & Type Definitions (handler)
// ==============================================================================
// File: types.go (lines 5-8)
type SlackNotifier struct {
    WebhookURL string
    Client     *http.Client
}

// File: types.go (lines 10-14)
type UserHandler struct {
    Notifier *SlackNotifier
    MaxUsers int
}

// ==============================================================================
// Package Context: Related Constructors & Helper Functions (handler)
// ==============================================================================
// File: init.go (lines 3-8)
func NewUserHandler() *UserHandler {
    return &UserHandler{
        Notifier: nil, // Root cause: Notifier is left uninitialized
        MaxUsers: 100,
    }
}

// File: helpers.go (lines 5-7)
func validateRequest(action string) bool {
    return len(action) > 0
}
```

With this complete context, the AI model immediately pinpoints that `UserHandler.Notifier` is `nil` because `NewUserHandler()` left it uninitialized, and proposes the exact fix in `NewUserHandler()` or adds a nil check before dereference.

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

This parses package directories once and stores rich multi-file AST context snippets in the `ast_nodes` PostgreSQL table for instant `< 5ms` lookups.
