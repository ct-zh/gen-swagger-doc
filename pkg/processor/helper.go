package processor

import (
	"fmt"
	"go/ast"

	"gen-swagger-doc/pkg/driver"
)

// makeHandlerKey 生成唯一的 Handler Key
func makeHandlerKey(info driver.HandlerInfo) string {
	// 如果 ReceiverType 为空，使用 "PkgName..FuncName"
	return fmt.Sprintf("%s.%s.%s", info.PkgName, info.ReceiverType, info.FunctionName)
}

// getReceiverTypeName 获取 FuncDecl 的 Receiver 类型名称
func getReceiverTypeName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}

	// Receiver type expr
	expr := fn.Recv.List[0].Type

	// Unwind *StarExpr
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	// Get Ident name
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}
