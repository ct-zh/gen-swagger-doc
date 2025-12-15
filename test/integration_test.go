package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gen-swagger-doc/pkg/driver/gin"
	"gen-swagger-doc/pkg/processor"
)

// setupTestDir 创建临时测试目录和文件
func setupTestDir(t *testing.T, files map[string]string) string {
	dir, err := os.MkdirTemp("", "swagger-gen-test-*")
	if err != nil {
		t.Fatal(err)
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestProcessor_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		files         map[string]string
		expectMatches []string // 期望在结果文件中出现的字符串
		expectNoMatch []string // 期望不出现的字符串
	}{
		{
			name: "匿名函数 Handler",
			files: map[string]string{
				"main.go": `
package main
import "github.com/gin-gonic/gin"
func main() {
	r := gin.Default()
	// 匿名函数，目前不支持提取名字，但不能 panic
	r.GET("/anon", func(c *gin.Context) {}) 
}
`,
			},
			expectNoMatch: []string{"@Router"}, // 应该跳过，不报错
		},
		{
			name: "Handler 是变量方法 (SelectorExpr)",
			files: map[string]string{
				"main.go": `
package main
import "github.com/gin-gonic/gin"
type UserAPI struct{}
func (u *UserAPI) Create(c *gin.Context) {}
func main() {
	api := &UserAPI{}
	r := gin.Default()
	r.POST("/users", api.Create)
}
`,
			},
			// 目前逻辑: GetHandlerName 支持 SelectorExpr (api.Create -> Create)
			// 但是 injectComments 需要匹配 FuncDecl，"Create" 对应 func (u *UserAPI) Create...
			// 我们的 Processor 只是简单匹配 FuncDecl.Name.Name，所以应该能匹配到 "Create"
			expectMatches: []string{
				"// @Summary POST /users",
				"// @Router /users [post]",
			},
		},
		{
			name: "单个 List 函数",
			files: map[string]string{
				"main.go": `
package main
import "github.com/gin-gonic/gin"
type A struct{}
func (a *A) List(c *gin.Context) {} 
func main() {
	a := &A{}
	r := gin.Default()
	r.GET("/a", a.List)
}
`,
			},
			expectMatches: []string{
				"// @Summary GET /a",
				"// @Router /a [get]",
			},
		},
		{
			name: "同名函数干扰 (不同 Receiver)",
			files: map[string]string{
				"main.go": `
package main
import "github.com/gin-gonic/gin"
type A struct{}
func (a *A) List(c *gin.Context) {} 

type B struct{}
func (b *B) List(c *gin.Context) {} 

func main() {
	a := &A{}
	r := gin.Default()
	r.GET("/a", a.List) // 应该只给 A.List 加注释
}
`,
			},
			// MVP 限制: 无法区分同名函数，为了安全起见，跳过注入
			expectNoMatch: []string{"@Router"},
		},
		{
			name: "缺少 Handler 参数",
			files: map[string]string{
				"main.go": `
package main
import "github.com/gin-gonic/gin"
func main() {
	r := gin.Default()
	r.GET("/broken") // 缺少 handler，不应该 panic
}
`,
			},
			expectNoMatch: []string{"@Router"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupTestDir(t, tt.files)
			defer os.RemoveAll(dir)

			opts := processor.Options{
				WorkDir: dir,
				Driver:  gin.NewDriver(),
			}

			// 运行 Processor
			if err := processor.Run(opts); err != nil {
				t.Fatalf("Run failed: %v", err)
			}

			// 验证结果
			content, err := os.ReadFile(filepath.Join(dir, "main.go"))
			if err != nil {
				t.Fatal(err)
			}
			strContent := string(content)

			for _, match := range tt.expectMatches {
				if !strings.Contains(strContent, match) {
					t.Errorf("Expected content to contain %q, but got:\n%s", match, strContent)
				}
			}

			for _, noMatch := range tt.expectNoMatch {
				if strings.Contains(strContent, noMatch) {
					t.Errorf("Expected content NOT to contain %q, but got:\n%s", noMatch, strContent)
				}
			}
		})
	}
}
