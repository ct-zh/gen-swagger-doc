package test

import (
	"gen-swagger-doc/pkg/driver/daenerys"
	"gen-swagger-doc/pkg/processor"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// QA Test: Edge cases and destructive testing for DaenerysDriver
func TestDaenerysDriver_QA(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "daenerys_qa")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	files := map[string]string{
		"qa_router.go": `
package http

import (
	httpserver "code.nexita.net/infra/libaray/daenerys/http/server"
)

// Scenario 1: Variable Reuse / Scope Leak?
// If scope is not handled, 'g' from scopeA might leak to scopeB if not redefined.
func scopeA(s httpserver.Server) {
	g := s.GROUP("/group-a")
	g.GET("/route-a", handlerA)
}

func scopeB(s httpserver.Server) {
	// In valid Go, 'g' must be defined. 
	// But let's say we define a DIFFERENT 'g' here.
	g := s.GROUP("/group-b")
	g.GET("/route-b", handlerB)
}

// Scenario 2: Deep Nesting
func deepNest(s httpserver.Server) {
	l1 := s.GROUP("/l1")
	l2 := l1.GROUP("/l2")
	l3 := l2.GROUP("/l3")
	l3.GET("/deep", handlerDeep)
}

// Scenario 3: Complex Type Resolution
func typeTest(s httpserver.Server) {
	s.GET("/types", complexTypes)
}
`,
		"qa_handlers.go": `
package http

import (
	httpserver "code.nexita.net/infra/libaray/daenerys/http/server"
)

type SomePayload struct {
	ID string
}

func handlerA(c *httpserver.Context) {
	// Add body to avoid empty function formatting issues
	_ = c.QueryInt("id")
}

func handlerB(c *httpserver.Context) {
	_ = c.QueryString("name")
}

func handlerDeep(c *httpserver.Context) {
	c.JSON(nil, nil)
}

// var declaration instead of :=
func complexTypes(c *httpserver.Context) {
	var payload SomePayload
	c.JSON(payload, nil)
}
`,
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	opts := processor.Options{
		WorkDir: tmpDir,
		Driver:  daenerys.New(),
	}

	if err := processor.Run(opts); err != nil {
		t.Fatalf("Processor failed: %v", err)
	}

	// Verify Results
	// Router comments are injected into the HANDLER functions, which are in qa_handlers.go
	handlersContent, err := os.ReadFile(filepath.Join(tmpDir, "qa_handlers.go"))
	if err != nil {
		t.Fatal(err)
	}
	handlersCode := string(handlersContent)

	// Check Scenario 1
	if !strings.Contains(handlersCode, "@Router /group-a/route-a [get]") {
		t.Errorf("Scenario 1 Failed: /group-a/route-a not found in handlers. \n%s", handlersCode)
	}
	if !strings.Contains(handlersCode, "@Router /group-b/route-b [get]") {
		t.Errorf("Scenario 1 Failed: /group-b/route-b not found in handlers. \n%s", handlersCode)
	}
	// Ensure no cross-contamination (e.g. /group-a/route-b)
	if strings.Contains(handlersCode, "/group-a/route-b") {
		t.Error("Scenario 1 Failed: Scope leak detected! Found /group-a/route-b")
	}

	// Check Scenario 2
	if !strings.Contains(handlersCode, "@Router /l1/l2/l3/deep [get]") {
		t.Error("Scenario 2 Failed: Deep nesting path incorrect")
	}

	// Check Scenario 3 (Handler Analysis)
	if !strings.Contains(handlersCode, "@Success 200 {object} SomePayload") {
		t.Errorf("Scenario 3 Failed: Failed to resolve 'var payload SomePayload'. Got:\n%s", handlersCode)
	}
}
