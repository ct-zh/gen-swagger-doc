package processor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"gen-swagger-doc/pkg/driver"
	"gen-swagger-doc/pkg/generator"
)

// Options 运行参数
type Options struct {
	WorkDir string        // 工作目录
	Driver  driver.Driver // 驱动实例
}

// Run 执行处理流程
func Run(opts Options) error {
	fset := token.NewFileSet()

	// 1. 收集阶段：找到所有路由注册信息
	// Map: HandlerKey -> RouteInfo
	// Key format: PkgName.ReceiverType.FuncName
	routeMap := make(map[string]driver.RouteInfo)

	err := filepath.Walk(opts.WorkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// 解析 AST (只读)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		pkgName := f.Name.Name

		ast.Inspect(f, func(n ast.Node) bool {
			ctx := &driver.Context{
				FileSet: fset,
				Node:    n,
				File:    f,
				PkgName: pkgName,
			}

			handler, ok := opts.Driver.CheckNode(ctx)
			if !ok || handler == nil {
				return true
			}

			// 提取 Handler 信息
			var handlerInfo driver.HandlerInfo
			if infoProvider, ok := handler.(driver.HandlerInfoProvider); ok {
				handlerInfo = infoProvider.GetHandlerInfo(ctx)
			} else if nameProvider, ok := handler.(driver.HandlerNameProvider); ok {
				// Fallback for legacy drivers
				handlerInfo = driver.HandlerInfo{
					PkgName:      pkgName,
					FunctionName: nameProvider.GetHandlerName(ctx),
				}
			}

			if handlerInfo.FunctionName == "" {
				return true
			}

			// 提取路由信息
			if routerParser, ok := handler.(driver.RouterParser); ok {
				routeInfo := routerParser.ParseRouter(ctx)
				key := makeHandlerKey(handlerInfo)
				routeMap[key] = routeInfo
			}

			return true
		})

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Found %d routes\n", len(routeMap))

	// 2. 注入阶段：修改 Handler 函数的注释
	// 使用新的 FileSet，避免与第一阶段的偏移量冲突
	injectFset := token.NewFileSet()

	err = filepath.Walk(opts.WorkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// 重新解析 AST (这次是为了修改)
		f, err := parser.ParseFile(injectFset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		pkgName := f.Name.Name
		var modified bool

		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}

			receiverType := getReceiverTypeName(fn)
			key := makeHandlerKey(driver.HandlerInfo{
				PkgName:      pkgName,
				ReceiverType: receiverType,
				FunctionName: fn.Name.Name,
			})

			routeInfo, exists := routeMap[key]
			if !exists {
				return true
			}

			// 找到目标函数，开始注入注释
			if injectComments(fn, routeInfo) {
				modified = true
				fmt.Printf("Injecting docs for %s (Key: %s)\n", fn.Name.Name, key)
			}

			return true
		})

		// 如果文件被修改，写回磁盘
		if modified {
			var buf bytes.Buffer
			if err := format.Node(&buf, injectFset, f); err != nil {
				return err
			}
			return os.WriteFile(path, buf.Bytes(), info.Mode())
		}

		return nil
	})

	return err
}

// injectComments 注入 Swagger 注释
// 返回 true 表示是否有修改
func injectComments(fn *ast.FuncDecl, route driver.RouteInfo) bool {
	// 调用 Generator 生成标准注释
	// 目前 params 和 responses 为空，后续会从 Driver 中获取
	// TODO: 从 Driver 获取 Params 和 Responses
	rawLines := generator.GenerateSwaggerDocs(route, nil, nil)

	if fn.Doc == nil {
		fn.Doc = &ast.CommentGroup{}
	}

	// 检查是否已经存在 @Router 注释，避免重复添加
	// 如果存在，我们假设整个 Swagger 块都已经存在，暂时不做增量更新
	for _, c := range fn.Doc.List {
		if strings.Contains(c.Text, "@Router") {
			return false // 已经有了，跳过
		}
	}

	// 追加注释
	for _, line := range rawLines {
		fn.Doc.List = append(fn.Doc.List, &ast.Comment{
			// Slash: fn.Pos() - 1, // 移除 Hack，看看原始行为
			Text: "// " + line,
		})
	}

	// 强制清除 Func 关键字和函数名的位置信息，使 printer 重新排版
	// 这样可以确保新添加的 Doc 注释被正确打印
	if fn.Type != nil {
		fn.Type.Func = token.NoPos
	}
	if fn.Name != nil {
		fn.Name.NamePos = token.NoPos
	}
	if fn.Recv != nil {
		fn.Recv.Opening = token.NoPos
		fn.Recv.Closing = token.NoPos
	}

	return true
}
