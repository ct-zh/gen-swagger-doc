package gin

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"gen-swagger-doc/pkg/driver"
)

func TestGinDriver_CheckNode_and_ParseRouter(t *testing.T) {
	// 1. 构造一段伪代码
	src := `
package main
func main() {
	r.GET("/ping", Pong)
	r.POST("/users", CreateUser)
	http.Get("http://google.com") // 这不应该被识别
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	d := NewDriver()

	// 2. 遍历 AST 查找 CallExpr
	ast.Inspect(f, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		ctx := &driver.Context{
			FileSet: fset,
			Node:    n,
			PkgName: "main",
		}

		handler, isRoute := d.CheckNode(ctx)

		// 3. 验证结果
		// 获取当前调用的函数名
		var funcName string
		if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			funcName = sel.Sel.Name
		} else if id, ok := callExpr.Fun.(*ast.Ident); ok {
			funcName = id.Name // e.g. http.Get 的情况比较复杂，这里简化
		}

		if funcName == "Get" {
			// http.Get 不应该被识别 (虽然它是 Get，但在我们的逻辑里，我们只检查 method name)
			// 等等，我们的逻辑只检查了 Name="GET"。http.Get 的 Name 是 "Get"。
			// Gin 是 r.GET (全大写)。所以 http.Get 应该返回 false。
			if isRoute {
				t.Errorf("Expected http.Get to NOT be a route")
			}
		}

		if funcName == "GET" {
			if !isRoute {
				t.Errorf("Expected r.GET to be a route")
			}
			// 验证 ParseRouter
			if routerParser, ok := handler.(driver.RouterParser); ok {
				info := routerParser.ParseRouter(ctx)
				if info.Method != "GET" {
					t.Errorf("Expected method GET, got %s", info.Method)
				}
				if info.Path != "/ping" {
					t.Errorf("Expected path /ping, got %s", info.Path)
				}
			} else {
				t.Errorf("Handler should implement RouterParser")
			}
		}

		return true
	})
}
