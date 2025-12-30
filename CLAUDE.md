# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Gen-Swagger-Doc** is an intelligent Swagger documentation auto-generation tool for Go backend projects. Unlike traditional `swag` tools that require manual annotation, this tool deeply parses Go source code (AST), automatically understands routing structure, request parameters, and response types, then injects standard Swagger 2.0 comments.

## Common Commands

### Build and Run
```bash
# Build the tool
go build -o swagger-gen ./cmd/swagger-gen

# Run on current directory with Gin framework
go run ./cmd/swagger-gen -framework gin

# Run on specific directory with Daenerys framework
go run ./cmd/swagger-gen -dir ./path/to/project -framework daenerys
```

### Testing
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./pkg/driver/gin
go test ./pkg/driver/daenerys
go test ./test

# Run specific test function
go test -run TestGinDriver ./pkg/driver/gin
go test -run TestDaenerysDriver ./test
```

### Development
```bash
# Check for syntax errors
go vet ./...

# Format code
go fmt ./...
```

## Architecture

This project uses a **Core-Driver** layered architecture that separates framework-agnostic processing logic from framework-specific parsing logic.

### Core Components

1. **Processor** (`pkg/processor/`): The orchestration engine
   - Two-phase execution: Collection phase → Injection phase
   - **Collection Phase**: Walks all `.go` files, calls driver to identify route registrations, builds a map of `HandlerKey → RouteInfo`
   - **Injection Phase**: Re-parses files, matches handler functions with collected routes, injects Swagger comments
   - HandlerKey format: `PkgName.ReceiverType.FunctionName` (see `makeHandlerKey()` in processor/helper.go:11)
   - Uses separate `token.FileSet` for each phase to avoid position conflicts

2. **Generator** (`pkg/generator/`): Swagger comment formatting
   - Transforms `RouteInfo`, `ParamInfo`, `ResponseInfo` into standard Swagger annotation lines
   - Generates `@Summary`, `@Router`, `@Param`, `@Success` tags
   - Does NOT include `//` prefix in output (caller's responsibility)

3. **Driver Interface** (`pkg/driver/interface.go`): Plugin system for framework adapters
   - **Required**: `Driver.CheckNode(ctx)` - Identifies if an AST node is a route registration
   - **Optional Mix-ins**: Drivers implement capabilities via additional interfaces:
     - `FileParser`: Override per-node scanning with whole-file analysis (for group routes)
     - `FuncBodyAnalyzer`: Extract params/responses from function body AST
     - `HandlerInfoProvider`: Get detailed handler location (package, receiver, function)
     - `RouterParser`: Extract route path and HTTP method

### Framework Drivers

1. **Gin Driver** (`pkg/driver/gin/`)
   - Pattern: `r.GET("/path", handlerFunc)`
   - Uses `CheckNode()` to find `CallExpr` with HTTP method selectors (GET/POST/etc)
   - `AnalyzeFunction()` walks function body to find:
     - Params: `c.Query()`, `c.Param()`, `c.BindJSON()`
     - Responses: `c.JSON(code, obj)` - extracts status code and response type
   - Variable type resolution: Traces variable assignments to resolve struct types from `c.JSON(myVar)` calls

2. **Daenerys Driver** (`pkg/driver/daenerys/`)
   - Internal framework from `code.nexita.net`
   - Pattern: `s.GROUP("/api").GET("/users", handler)` - supports nested route groups
   - Implements `FileParser` to analyze entire file instead of per-node
   - Maintains `varPathMap` to track route group variable names → path prefixes
   - Example: `api := s.GROUP("/api")` → `varPathMap["api"] = "/api"`
   - Joins parent paths when detecting route calls: `api.GET("/users", h)` → `/api/users`
   - Context methods: `c.QueryInt64()`, `c.QueryString()`, `c.JSON(data, err)`

### Key Design Patterns

**Two-Phase Processing**: Required because AST modification during traversal is unsafe. Collection phase is read-only, injection phase modifies and writes back.

**Driver Plugin System**: Allows adding new framework support without modifying core. New drivers only need to implement `Driver` interface + optional mix-ins.

**Variable Type Resolution**: Both drivers implement basic type tracking by scanning assignments and declarations to resolve types like:
```go
data := UserResponse{}  // varTypeMap["data"] = "UserResponse"
c.JSON(200, data)       // Resolves to "@Success 200 {object} UserResponse"
```

**Comment Injection Safeguards** (processor.go:198-242):
- Checks for existing `@Router` tag to avoid duplicate injection
- Creates `fn.Doc` if missing and registers it in `file.Comments`
- Clears `token.Pos` on function nodes to force `go/format` to re-layout comments

## Important Implementation Notes

- **AST Comment Handling**: When creating new `CommentGroup`, must append to `file.Comments` or `go/format` will ignore it (processor.go:209)
- **Position Clearing**: Set `token.NoPos` on function nodes after adding comments to ensure proper formatting (processor.go:231-239)
- **HandlerKey Matching**: Route collection and injection must use identical key format. If adding receiver type tracking, update both phases.
- **PathFilter**: `Options.PathFilter` allows processing only specific routes (useful for debugging)

## Adding New Framework Support

1. Create new driver package: `pkg/driver/yourframework/`
2. Implement `Driver.CheckNode()` to identify route registrations
3. Optionally implement:
   - `FileParser.ParseFile()` if routes use complex patterns (groups, middleware)
   - `FuncBodyAnalyzer.AnalyzeFunction()` to extract params/responses from handler body
   - `HandlerInfoProvider.GetHandlerInfo()` for accurate handler identification
4. Register in `cmd/swagger-gen/main.go` switch statement
5. Add tests following patterns in `test/integration_test.go`

## Testing Strategy

- **Unit tests**: `pkg/driver/gin/driver_test.go` - test driver logic in isolation
- **Integration tests**: `test/integration_test.go`, `test/daenerys_test.go` - end-to-end with real code samples
- **QA tests**: `test/qa_daenerys_test.go` - edge cases and regression tests
- Use `Options.PathFilter` in tests to isolate specific routes

## Module Information

- Module path: `gen-swagger-doc`
- Go version: 1.24.5
- Main dependency: `github.com/gin-gonic/gin` (for examples/tests only, not required for core)
