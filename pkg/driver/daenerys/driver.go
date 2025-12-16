package daenerys

import (
	"gen-swagger-doc/pkg/driver"
	"go/ast"
	"go/token"
	"path"
	"strings"
)

// DaenerysDriver 适配 code.nexita.net 的 Daenerys 框架
type DaenerysDriver struct{}

// New 创建一个新的 DaenerysDriver
func New() driver.Driver {
	return &DaenerysDriver{}
}

// CheckNode 对于 DaenerysDriver，我们主要依赖 ParseFile，
// 但为了兼容接口，这里返回 false。
func (d *DaenerysDriver) CheckNode(ctx *driver.Context) (driver.Handler, bool) {
	return nil, false
}

// ParseFile 实现了 driver.FileParser 接口，用于全文件解析以处理 Group 路由
func (d *DaenerysDriver) ParseFile(ctx *driver.Context) []driver.ParsedRoute {
	var routes []driver.ParsedRoute

	// 变量名 -> 路径前缀
	// e.g. "api" -> "/api/novel/v1"
	// "s" -> "" (假设 s 是根 server)
	varPathMap := make(map[string]string)

	ast.Inspect(ctx.File, func(n ast.Node) bool {
		// 我们只关心函数定义中的语句
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// 在每个函数内部，重新初始化变量跟踪（或者是基于控制流的，这里简化为线性扫描）
		// 注意：如果路由定义跨越多个函数，这种简单扫描会失效。
		// 但根据 router.go 的样例，所有路由都在 initRoute 中，或者嵌套 block 中。
		// 由于 ast.Inspect 是深度优先，我们可以在遍历 BlockStmt 时维护作用域。
		// 但为了简单，假设变量名在函数内是唯一的且按顺序定义。

		// 找到 Server 参数名
		// func initRoute(s httpserver.Server)
		if len(fn.Type.Params.List) > 0 {
			for _, field := range fn.Type.Params.List {
				// 简单的类型名称检查
				if isServerType(field.Type) {
					for _, name := range field.Names {
						varPathMap[name.Name] = ""
					}
				}
			}
		}

		// 遍历函数体
		if fn.Body != nil {
			ast.Inspect(fn.Body, func(bn ast.Node) bool {
				switch stmt := bn.(type) {
				case *ast.AssignStmt:
					// api := s.GROUP("/...")
					handleAssignment(stmt, varPathMap)
				case *ast.ExprStmt:
					// book.GET("/...", handler)
					if call, ok := stmt.X.(*ast.CallExpr); ok {
						if route, ok := handleRouteCall(call, varPathMap, ctx.PkgName); ok {
							routes = append(routes, route)
						}
					}
				}
				return true
			})
		}

		return true
	})

	return routes
}

func isServerType(expr ast.Expr) bool {
	// 检查是否是 httpserver.Server
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok {
			return x.Name == "httpserver" && sel.Sel.Name == "Server"
		}
	}
	return false
}

func handleAssignment(stmt *ast.AssignStmt, varMap map[string]string) {
	if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
		return
	}

	// 左边必须是变量名
	ident, ok := stmt.Lhs[0].(*ast.Ident)
	if !ok {
		return
	}
	varName := ident.Name

	// 右边必须是 CallExpr (e.g. s.GROUP(...))
	call, ok := stmt.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}

	// 检查方法名是否为 GROUP
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "GROUP" {
		return
	}

	// 获取调用者变量名 (e.g. "s" in "s.GROUP")
	callerIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}
	callerName := callerIdent.Name

	// 获取路径参数
	if len(call.Args) > 0 {
		if pathLit, ok := call.Args[0].(*ast.BasicLit); ok && pathLit.Kind == token.STRING {
			pathVal := strings.Trim(pathLit.Value, "\"")

			// 查找父路径
			if parentPath, exists := varMap[callerName]; exists {
				// 拼接路径
				fullPath := joinPath(parentPath, pathVal)
				varMap[varName] = fullPath
				// fmt.Printf("Mapped variable %s -> %s (parent: %s)\n", varName, fullPath, callerName)
			}
		}
	}
}

func handleRouteCall(call *ast.CallExpr, varMap map[string]string, pkgName string) (driver.ParsedRoute, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return driver.ParsedRoute{}, false
	}

	method := sel.Sel.Name
	if !isHTTPMethod(method) {
		return driver.ParsedRoute{}, false
	}

	// 获取调用者 (e.g. "book" in "book.GET")
	callerIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return driver.ParsedRoute{}, false
	}
	callerName := callerIdent.Name

	// 获取父路径
	parentPath, exists := varMap[callerName]
	if !exists {
		// 如果找不到变量映射，可能不是我们关注的路由变量，或者解析失败
		// 暂且跳过
		return driver.ParsedRoute{}, false
	}

	// 提取参数
	if len(call.Args) < 2 {
		return driver.ParsedRoute{}, false
	}

	// Arg 0: Path
	pathLit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || pathLit.Kind != token.STRING {
		return driver.ParsedRoute{}, false
	}
	subPath := strings.Trim(pathLit.Value, "\"")
	fullPath := joinPath(parentPath, subPath)

	// Arg 1: Handler Function
	handlerIdent, ok := call.Args[1].(*ast.Ident)
	if !ok {
		// 可能是 selector (e.g. api.Handler)，暂不支持，只支持包级函数
		return driver.ParsedRoute{}, false
	}
	handlerFunc := handlerIdent.Name

	return driver.ParsedRoute{
		HandlerInfo: driver.HandlerInfo{
			PkgName:      pkgName,
			FunctionName: handlerFunc,
			ReceiverType: "", // 普通函数
		},
		RouteInfo: driver.RouteInfo{
			Method: method,
			Path:   fullPath,
		},
	}, true
}

func joinPath(p1, p2 string) string {
	return path.Join(p1, p2)
}

func isHTTPMethod(m string) bool {
	switch m {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "ANY":
		return true
	}
	return false
}

// AnalyzeFunction 实现 FuncBodyAnalyzer
func (d *DaenerysDriver) AnalyzeFunction(fn *ast.FuncDecl) ([]driver.ParamInfo, []driver.ResponseInfo) {
	var params []driver.ParamInfo
	var responses []driver.ResponseInfo

	// 1. 找到 context 参数名
	ctxParamName := ""
	for _, field := range fn.Type.Params.List {
		if isContextType(field.Type) {
			if len(field.Names) > 0 {
				ctxParamName = field.Names[0].Name
				break
			}
		}
	}

	if ctxParamName == "" {
		return nil, nil
	}

	// 简单的变量类型表: varName -> typeName
	varTypeMap := make(map[string]string)

	// 2. 遍历函数体
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		// 追踪变量定义: data := SomeStruct{}
		if assign, ok := n.(*ast.AssignStmt); ok {
			if len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
				if ident, ok := assign.Lhs[0].(*ast.Ident); ok {
					// 尝试从 Rhs 获取类型
					if comp, ok := assign.Rhs[0].(*ast.CompositeLit); ok {
						if typeIdent, ok := comp.Type.(*ast.Ident); ok {
							varTypeMap[ident.Name] = typeIdent.Name
						}
					}
				}
			}
		}

		// 追踪变量声明: var data SomeStruct
		if declStmt, ok := n.(*ast.DeclStmt); ok {
			if genDecl, ok := declStmt.Decl.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						// var x T
						if valueSpec.Type != nil {
							typeName := ""
							if ident, ok := valueSpec.Type.(*ast.Ident); ok {
								typeName = ident.Name
							} else if sel, ok := valueSpec.Type.(*ast.SelectorExpr); ok {
								// e.g. somepkg.Struct -> Struct
								typeName = sel.Sel.Name
							}

							if typeName != "" {
								for _, name := range valueSpec.Names {
									varTypeMap[name.Name] = typeName
								}
							}
						}
					}
				}
			}
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// 检查接收者是否是 context 变量
		if x, ok := sel.X.(*ast.Ident); !ok || x.Name != ctxParamName {
			return true
		}

		// Params
		switch sel.Sel.Name {
		case "QueryInt64", "QueryInt", "QueryString":
			if len(call.Args) > 0 {
				if name := extractStringLit(call.Args[0]); name != "" {
					typ := "string"
					if strings.Contains(sel.Sel.Name, "Int") {
						typ = "integer"
					}
					params = append(params, driver.ParamInfo{
						Name:     name,
						In:       "query",
						Type:     typ,
						Required: true, // 假设这些特定类型的获取方法暗示必填
					})
				}
			}
		}

		// Responses
		// c.JSON(data, nil) -> success
		// c.JSONAbort(nil, err) -> failure
		switch sel.Sel.Name {
		case "JSON":
			if len(call.Args) >= 1 {
				// 通常第一个参数是 data
				// 如果是 nil，可能忽略
				// 这里的 Daenerys API 似乎是 c.JSON(data, error) ?
				// book.go: c.JSON(data, nil)
				// book.go: c.JSON(svc.GetChapterCount(c.Ctx, bookId)) -> 单参数？

				// 假设: c.JSON(data, err) or c.JSON(data)
				// 如果第二个参数存在且为 nil，或者是单参数调用，我们认为是 Success 200

				isSuccess := true
				if len(call.Args) == 2 {
					// Check if 2nd arg is nil
					if ident, ok := call.Args[1].(*ast.Ident); ok && ident.Name == "nil" {
						isSuccess = true
					} else {
						// 如果第二个参数不是 nil，可能是 error，暂不处理或视为 error
						// 但根据代码 c.JSON(data, nil)，这是成功路径
					}
				}

				if isSuccess {
					// 提取第一个参数的类型
					// 如果是变量，需要解析类型。这里暂时只做简单的变量名匹配或函数调用返回类型推断
					// 这是一个难点，因为我们无法轻易解析 `svc.GetBookInfo` 的返回类型
					// 除非我们进行跨包分析。
					// 我们可以尝试从 call.Args[0] 提取变量名，并查找它的定义

					respType := "object" // default
					// 尝试提取变量名
					if ident, ok := call.Args[0].(*ast.Ident); ok {
						// 查找该变量的定义
						if typeName, exists := varTypeMap[ident.Name]; exists {
							respType = typeName
						} else {
							// 如果找不到定义，不要使用变量名作为类型，而是使用 object
							// 因为 {object} data 是无效的 Swagger
							// respType = ident.Name
							respType = "object"
						}
					} else if _, ok := call.Args[0].(*ast.CallExpr); ok {
						// c.JSON(svc.Get...)
						respType = "object"
					}

					responses = append(responses, driver.ResponseInfo{
						Code:   200,
						Schema: respType,
					})
				}
			}
		case "JSONAbort":
			// c.JSONAbort(nil, code.InvalidParam)
			// c.JSONAbort(nil, err)
			responses = append(responses, driver.ResponseInfo{
				Code:   400, // 默认 400，或者尝试解析 code
				Schema: "code.Error",
			})
		}

		return true
	})

	return params, responses
}

func isContextType(expr ast.Expr) bool {
	// Handle pointer: *httpserver.Context
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok {
			return x.Name == "httpserver" && sel.Sel.Name == "Context"
		}
	}
	return false
}

func extractStringLit(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		return strings.Trim(lit.Value, "\"")
	}
	return ""
}
