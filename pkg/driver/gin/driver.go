package gin

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"gen-swagger-doc/pkg/driver"
)

// 支持的 HTTP 方法
var httpMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"OPTIONS": true,
	"HEAD":    true,
}

// GinDriver 实现了 driver.Driver 接口
type GinDriver struct{}

// NewDriver 创建一个新的 GinDriver 实例
func NewDriver() *GinDriver {
	return &GinDriver{}
}

// CheckNode 检查节点是否为 Gin 路由注册
// e.g. r.GET("/path", handler)
func (d *GinDriver) CheckNode(ctx *driver.Context) (driver.Handler, bool) {
	// 1. 必须是函数调用
	callExpr, ok := ctx.Node.(*ast.CallExpr)
	if !ok {
		return nil, false
	}

	// 2. 必须是 SelectorExpr (例如 r.GET)
	// 如果是直接调用 GET() 这种 Ident，我们这里暂不处理，假设都是通过 router 实例调用的
	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}

	// 3. 方法名必须是 HTTP 方法
	methodName := selExpr.Sel.Name
	if !httpMethods[methodName] {
		return nil, false
	}

	// 4. 参数检查：至少要有两个参数 (path, handler)
	// r.GET("/path", handler)
	if len(callExpr.Args) < 2 {
		return nil, false
	}

	return &GinHandler{
		ctx:  ctx,
		node: callExpr,
	}, true
}

// GinHandler 实现了 driver.Handler 和 driver.RouterParser
type GinHandler struct {
	ctx  *driver.Context
	node *ast.CallExpr
}

// ParseRouter 提取路由路径和方法
func (h *GinHandler) ParseRouter(ctx *driver.Context) driver.RouteInfo {
	// 方法名就是函数名 (e.g. "GET")
	selExpr := h.node.Fun.(*ast.SelectorExpr)
	method := selExpr.Sel.Name

	// 第一个参数通常是 Path
	// r.GET("/ping", ...)
	var path string
	if len(h.node.Args) > 0 {
		arg0 := h.node.Args[0]
		// 检查是否是字符串字面量
		if lit, ok := arg0.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			// 去除引号: "/ping" -> /ping
			unquoted, err := strconv.Unquote(lit.Value)
			if err == nil {
				path = unquoted
			} else {
				// 如果 unquote 失败（极少见），保留原值
				path = lit.Value
			}
		} else {
			// 如果路径是变量 (e.g. r.GET(pathVar, ...))，目前简单处理为空或占位符
			// TODO: 支持常量解析
			path = "unknown_path_var"
		}
	}

	return driver.RouteInfo{
		Method: strings.ToUpper(method),
		Path:   path,
	}
}

// GetHandlerName 提取路由处理函数名
func (h *GinHandler) GetHandlerName(ctx *driver.Context) string {
	// 最后一个参数通常是 Handler
	// r.GET("/path", handler) -> handler
	// r.GET("/path", middleware, handler) -> handler
	if len(h.node.Args) < 2 {
		return ""
	}

	lastArg := h.node.Args[len(h.node.Args)-1]

	// 情况 1: 直接是函数名 (Ident) -> handler
	if ident, ok := lastArg.(*ast.Ident); ok {
		return ident.Name
	}

	// 情况 2: 包名.函数名 (SelectorExpr) -> user.Create
	if sel, ok := lastArg.(*ast.SelectorExpr); ok {
		return sel.Sel.Name
	}

	return ""
}

// GetHandlerInfo 获取详细的 Handler 信息，包括 Receiver 类型
func (h *GinHandler) GetHandlerInfo(ctx *driver.Context) driver.HandlerInfo {
	info := driver.HandlerInfo{
		PkgName: ctx.PkgName,
	}

	if len(h.node.Args) < 2 {
		return info
	}

	lastArg := h.node.Args[len(h.node.Args)-1]

	switch expr := lastArg.(type) {
	case *ast.Ident:
		// Case 1: MyFunc
		info.FunctionName = expr.Name
	case *ast.SelectorExpr:
		// Case 2: api.List
		info.FunctionName = expr.Sel.Name
		// 尝试解析 Receiver 类型
		if xIdent, ok := expr.X.(*ast.Ident); ok {
			info.ReceiverType = resolveReceiverType(ctx, xIdent.Name)
		}
	}

	return info
}

// resolveReceiverType 尝试在当前包中查找变量的类型
// 这是一个非常简化的实现，只支持局部变量赋值和全局变量声明
func resolveReceiverType(ctx *driver.Context, varName string) string {
	if ctx.File == nil {
		return ""
	}

	var typeName string
	// 扫描文件中的赋值语句 := 和 var 声明
	// 注意：这里简单的全文件扫描，不考虑作用域遮蔽，仅作为 MVP 实现
	ast.Inspect(ctx.File, func(n ast.Node) bool {
		if typeName != "" {
			return false // 已找到
		}

		switch stmt := n.(type) {
		case *ast.AssignStmt:
			// api := &UserAPI{}
			for i, lhs := range stmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && ident.Name == varName {
					// 找到赋值，查看右边
					if i < len(stmt.Rhs) {
						rhs := stmt.Rhs[i]
						typeName = extractTypeFromExpr(rhs)
					}
				}
			}
		case *ast.GenDecl:
			// var api UserAPI
			if stmt.Tok == token.VAR {
				for _, spec := range stmt.Specs {
					vSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range vSpec.Names {
						if name.Name == varName {
							if vSpec.Type != nil {
								typeName = extractTypeFromExpr(vSpec.Type)
							} else if len(vSpec.Values) > 0 {
								// var api = &UserAPI{}
								typeName = extractTypeFromExpr(vSpec.Values[0])
							}
						}
					}
				}
			}
		}
		return true
	})

	return typeName
}

func extractTypeFromExpr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.CompositeLit:
		// &UserAPI{} or UserAPI{}
		return extractTypeFromExpr(t.Type)
	case *ast.UnaryExpr:
		// &UserAPI{}
		if t.Op == token.AND {
			return extractTypeFromExpr(t.X)
		}
	case *ast.Ident:
		// UserAPI
		return t.Name
	case *ast.StarExpr:
		// *UserAPI
		return extractTypeFromExpr(t.X)
	}
	return ""
}
