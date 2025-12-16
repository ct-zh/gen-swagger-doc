package test

import (
	"fmt"
	"gen-swagger-doc/pkg/driver/daenerys"
	"gen-swagger-doc/pkg/processor"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDaenerysDriver(t *testing.T) {
	// 1. Setup temp workspace
	tmpDir, err := os.MkdirTemp("", "daenerys_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 2. Create mock files
	files := map[string]string{
		"router.go": `
package http

import (
	httpserver "code.nexita.net/infra/libaray/daenerys/http/server"
)

func initRoute(s httpserver.Server) {
	api := s.GROUP("/api/novel/v1")
	{
		book := api.GROUP("/book")
		{
			book.GET("/book-info", bookInfo)         // 书籍信息
		}
	}
}
`,
		"book.go": `
package http

import (
	httpserver "code.nexita.net/infra/libaray/daenerys/http/server"
)

func bookInfo(c *httpserver.Context) {
	bookId := c.QueryInt64("book_id")
	if bookId <= 0 {
		c.JSONAbort(nil, nil)
		return
	}
	data := SomeStruct{}
	c.JSON(data, nil)
}
`,
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// 3. Run Processor with DaenerysDriver
	opts := processor.Options{
		WorkDir: tmpDir,
		Driver:  daenerys.New(),
	}

	if err := processor.Run(opts); err != nil {
		t.Fatalf("Processor failed: %v", err)
	}

	// 4. Verify results
	content, err := os.ReadFile(filepath.Join(tmpDir, "book.go"))
	if err != nil {
		t.Fatal(err)
	}
	codeStr := string(content)

	// Check Router path (must handle GROUP correctly)
	expectedRouter := "// @Router /api/novel/v1/book/book-info [get]"
	if !strings.Contains(codeStr, expectedRouter) {
		t.Errorf("Expected router path %q, but got:\n%s", expectedRouter, codeStr)
	}

	// Check Param
	if !strings.Contains(codeStr, "// @Param book_id query integer true") {
		t.Error("Missing expected param annotation")
	}

	// Check Success
	// Updated expectation: we now resolve "data" -> "SomeStruct"
	if !strings.Contains(codeStr, "// @Success 200 {object} SomeStruct") {
		t.Errorf("Missing expected success annotation with resolved type. Got:\n%s", codeStr)
	}

	fmt.Println(codeStr)
}
