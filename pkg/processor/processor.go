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
	// Map: HandlerName -> RouteInfo
	routeMap := make(map[string]driver.RouteInfo)
	// Map: FuncName -> Count (用于检测同名函数冲突)
	funcCount := make(map[string]int)

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
			// 统计函数定义
			if fn, ok := n.(*ast.FuncDecl); ok {
				funcCount[fn.Name.Name]++
			}

			ctx := &driver.Context{
				FileSet: fset,
				Node:    n,
				PkgName: pkgName,
			}

			handler, ok := opts.Driver.CheckNode(ctx)
			if !ok || handler == nil {
				return true
			}

			// 提取 Handler 名称
			var handlerName string
			if nameProvider, ok := handler.(driver.HandlerNameProvider); ok {
				handlerName = nameProvider.GetHandlerName(ctx)
			}

			if handlerName == "" {
				return true
			}

			// 提取路由信息
			if routerParser, ok := handler.(driver.RouterParser); ok {
				routeInfo := routerParser.ParseRouter(ctx)
				routeMap[handlerName] = routeInfo
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

		var modified bool

		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}

			handlerName := fn.Name.Name
			routeInfo, exists := routeMap[handlerName]
			if !exists {
				return true
			}

			// 如果存在同名函数，跳过注入以避免错误
			if funcCount[handlerName] > 1 {
				fmt.Printf("Warning: Ambiguous handler name %q (found %d times), skipping injection\n", handlerName, funcCount[handlerName])
				return true
			}

			// 找到目标函数，开始注入注释
			if injectComments(fn, routeInfo) {
				modified = true
				fmt.Printf("Injecting docs for %s\n", handlerName)
				// 清除函数声明的位置信息，强制 printer 根据 AST 结构重新排版
				// 这样可以避免因插入注释导致的偏移量冲突
				// fn.Pos = token.NoPos // 这是一个只读方法，不能赋值
				// 我们需要修改结构体字段，但在 AST 接口中 Pos() 是方法
				// 对于 FuncDecl: Type, Name, Doc 等都有 Pos。
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
	// 简单的注释生成逻辑 (MVP)
	// TODO: 后面会由 generator 模块接管
	newComments := []string{
		fmt.Sprintf("// @Router %s [%s]", route.Path, strings.ToLower(route.Method)),
	}

	if fn.Doc == nil {
		fn.Doc = &ast.CommentGroup{}
	}

	// 检查是否已经存在 @Router 注释，避免重复添加
	for _, c := range fn.Doc.List {
		if strings.Contains(c.Text, "@Router") {
			return false // 已经有了，跳过
		}
	}

	// 追加注释
	for _, text := range newComments {
		fn.Doc.List = append(fn.Doc.List, &ast.Comment{
			// Slash: fn.Pos() - 1, // 移除 Hack，看看原始行为
			Text: text,
		})
	}

	return true
}
