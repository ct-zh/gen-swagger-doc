package driver

import (
	"go/ast"
	"go/token"
)

// Context 用于在解析过程中传递 AST 上下文信息
type Context struct {
	FileSet *token.FileSet // 用于获取文件位置信息
	Node    ast.Node       // 当前正在处理的 AST 节点 (通常是 *ast.CallExpr)
	File    *ast.File      // 当前正在处理的文件 (AST Root)
	PkgName string         // 当前包名
}

// RouteInfo 描述 HTTP 路由的基本信息 (对应 @Router)
type RouteInfo struct {
	Method string // e.g., "GET", "POST"
	Path   string // e.g., "/api/v1/users"
}

// ParamInfo 描述请求参数 (对应 Swagger @Param)
type ParamInfo struct {
	Name     string // 参数名
	In       string // 位置: "path", "query", "body", "header", "formData"
	Type     string // 类型: "string", "integer", "boolean", "object"
	Required bool   // 是否必填
	Desc     string // 参数描述
	Schema   string // 如果是 body/object，这里存储 Go 结构体名称 (e.g., "UserRequest")
}

// ResponseInfo 描述响应数据 (对应 Swagger @Success/@Failure)
type ResponseInfo struct {
	Code    int    // HTTP 状态码
	Schema  string // 响应结构体名称 (e.g., "UserResponse")
	IsArray bool   // 是否返回数组
	Desc    string // 描述
}

// Driver 是所有驱动必须实现的入口接口
type Driver interface {
	// CheckNode 检查给定的 AST 节点是否是路由注册点。
	// 如果是，返回一个 Handler 实例和 true；否则返回 nil, false。
	// 这是一个工厂方法，用于创建特定节点的处理器。
	CheckNode(ctx *Context) (Handler, bool)
}

// FileParser 允许驱动接管整个文件的解析逻辑
// 这是一个可选接口。如果 Driver 实现了此接口，Processor 将优先调用 ParseFile，
// 而不再对文件中的每个节点调用 CheckNode。
// 这对于需要分析变量作用域、路由组 (Group) 或全局配置的框架非常有用。
type FileParser interface {
	ParseFile(ctx *Context) []ParsedRoute
}

// ParsedRoute 包含解析出的完整路由信息
type ParsedRoute struct {
	HandlerInfo HandlerInfo
	RouteInfo   RouteInfo
}

// Handler 代表一个被识别出的路由处理逻辑。
// 具体的 Driver 实现需要返回实现此接口的对象。
// 这是一个标记接口，具体的解析能力通过实现 Optional Parsers (Mix-ins) 来提供。
type Handler interface {
	// 这是一个标记接口，目前为空
}

// HandlerInfo 包含 Handler 的详细定位信息
type HandlerInfo struct {
	PkgName      string // 包名 (e.g., "main", "user")
	FunctionName string // 函数名 (e.g., "List", "Create")
	ReceiverType string // 接收者类型名 (e.g., "UserAPI", "OrderAPI")，如果是普通函数则为空
}

// --- Optional Parsers (Mix-ins) ---
// Handler 实现类可以选择性地实现以下接口，以提供特定的解析能力。

// HandlerInfoProvider 提供获取路由处理函数详细信息的能力
type HandlerInfoProvider interface {
	GetHandlerInfo(ctx *Context) HandlerInfo
}

// HandlerNameProvider 提供获取路由处理函数名称的能力
// Deprecated: Use HandlerInfoProvider instead
type HandlerNameProvider interface {
	GetHandlerName(ctx *Context) string
}

// RouterParser 解析路由路径和方法
type RouterParser interface {
	ParseRouter(ctx *Context) RouteInfo
}

// SummaryParser 解析接口摘要 (@Summary)
type SummaryParser interface {
	ParseSummary(ctx *Context) string
}

// DescriptionParser 解析详细描述 (@Description)
type DescriptionParser interface {
	ParseDescription(ctx *Context) string
}

// TagsParser 解析标签 (@Tags)
type TagsParser interface {
	ParseTags(ctx *Context) []string
}

// ParamParser 解析请求参数 (@Param)
type ParamParser interface {
	ParseParams(ctx *Context) []ParamInfo
}

// FuncBodyAnalyzer 允许驱动分析函数体以提取参数和响应信息
// 这是一个可选接口，如果 Driver 实现了它，Processor 会在注入阶段调用
type FuncBodyAnalyzer interface {
	AnalyzeFunction(fn *ast.FuncDecl) ([]ParamInfo, []ResponseInfo)
}

// ResponseParser 解析响应 (@Success/@Failure)
type ResponseParser interface {
	ParseResponses(ctx *Context) []ResponseInfo
}
