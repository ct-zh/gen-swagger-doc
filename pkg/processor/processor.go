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
	"sort"
	"strings"

	"gen-swagger-doc/pkg/driver"
	"gen-swagger-doc/pkg/generator"
)

// Options 运行参数
type Options struct {
	WorkDir    string        // 工作目录
	Driver     driver.Driver // 驱动实例
	PathFilter string        // 可选：仅处理匹配此路径的路由 (例如: "/api/v1/user/strategy/async_switch")
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

		// 优先使用 FileParser
		if fileParser, ok := opts.Driver.(driver.FileParser); ok {
			ctx := &driver.Context{
				FileSet: fset,
				File:    f,
				PkgName: pkgName,
			}
			parsedRoutes := fileParser.ParseFile(ctx)
			for _, pr := range parsedRoutes {
				key := makeHandlerKey(pr.HandlerInfo)
				routeMap[key] = pr.RouteInfo
			}
			return nil
		}

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

	// 如果指定了路径过滤器，只保留匹配的路由
	if opts.PathFilter != "" {
		filteredMap := make(map[string]driver.RouteInfo)
		for key, route := range routeMap {
			if route.Path == opts.PathFilter {
				filteredMap[key] = route
			}
		}
		if len(filteredMap) == 0 {
			return fmt.Errorf("no routes found matching path filter: %s", opts.PathFilter)
		}
		routeMap = filteredMap
		fmt.Printf("After filtering by path '%s': %d routes\n", opts.PathFilter, len(routeMap))
	}

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
			var params []driver.ParamInfo
			var responses []driver.ResponseInfo

			if analyzer, ok := opts.Driver.(driver.FuncBodyAnalyzer); ok {
				params, responses = analyzer.AnalyzeFunction(fn)
			}

			if injectComments(f, fn, routeInfo, params, responses) {
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
func injectComments(file *ast.File, fn *ast.FuncDecl, route driver.RouteInfo, params []driver.ParamInfo, responses []driver.ResponseInfo) bool {
	// 调用 Generator 生成标准注释
	rawLines := generator.GenerateSwaggerDocs(route, params, responses)

	// 捕获函数声明的位置，用于设置注释的位置
	// 使用 fn.Pos() (即 "func" 关键字的位置) 之前的位置
	pos := fn.Pos()

	if fn.Doc == nil {
		fn.Doc = &ast.CommentGroup{}
		// 关键修复: 如果创建了新的 Doc，必须将其添加到 file.Comments 中，
		// 否则 go/format 打印 *ast.File 时会忽略这些“孤立”的注释。
		// 注意: 这可能会打乱注释顺序，但由于我们稍后重置了 Pos，go/format 应该会根据 AST 结构打印。
		file.Comments = append(file.Comments, fn.Doc)
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
			Slash: pos - 1, // 设置正确的位置，避免位置错乱导致 printer 打印到错误位置
			Text:  "// " + line,
		})
	}

	// 重新排序 file.Comments，确保所有注释按位置顺序排列
	// 这对 go/printer 正确打印注释至关重要，特别是当我们在列表末尾添加了位置靠前的注释时
	sort.Slice(file.Comments, func(i, j int) bool {
		return file.Comments[i].Pos() < file.Comments[j].Pos()
	})

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
