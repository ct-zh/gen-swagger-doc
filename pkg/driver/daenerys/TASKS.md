# Tasks: Daenerys Driver Implementation

> **Objective**: Implement a new driver to support `code.nexita.net`'s internal framework (Daenerys), which is similar to Gin but uses different package names and context types.

## 1. Context Analysis
- **Framework**: `code.nexita.net/infra/libaray/daenerys/http/server`
- **Router Pattern**:
    ```go
    api := s.GROUP("/api/v1")
    {
        book := api.GROUP("/book")
        book.GET("/info", handler)
    }
    ```
- **Handler Signature**: `func handler(c *httpserver.Context)`

## 2. Implementation Status (Completed)

### Step 1: Interface Update
- [x] Added `driver.FileParser` interface to support full-file analysis (critical for nested Groups).

### Step 2: Driver Implementation
- [x] Created `pkg/driver/daenerys/driver.go`.
- [x] Implemented `ParseFile` with variable tracking for Group paths.
- [x] Implemented `AnalyzeFunction` for `*httpserver.Context`.
    - Supports `QueryInt64`, `QueryString`.
    - Supports `JSON`, `JSONAbort`.

### Step 3: Verification
- [x] Created `test/daenerys_test.go` simulating the `yuban.novel.buz` project structure.
- [x] Verified correct path concatenation (`/api/novel/v1/book/book-info`).
- [x] Verified parameter and response extraction.
- [x] Performed QA testing (`test/qa_daenerys_test.go`) for edge cases:
    - Scope reuse and variable shadowing.
    - Deeply nested routing.
    - Complex type declarations (short `:=` and long `var`).

## 3. Future Improvements
- **More Methods**: Support more `Context` methods like `PostForm`, `Header`.
- **Cross-File Analysis**: Support analyzing handlers defined in different packages/files (requires broader scope analysis).
