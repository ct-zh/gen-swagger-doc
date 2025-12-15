# Module: Driver Interface Design

**Status**: Draft
**Author**: System Architect
**Module Path**: `pkg/driver`

## 1. Overview
本模块定义了 `gen-swagger-doc` 与具体 Web 框架交互的契约。我们采用 Mix-in (混入) 模式，允许 Driver 按需实现解析逻辑。

## 2. Data Structures (数据结构)

研发工程师需在 `pkg/driver/interface.go` 中定义以下核心结构体：

### 2.1 Context
用于在解析过程中传递 AST 上下文信息。

```go
type Context struct {
    FileSet *token.FileSet // 用于获取文件位置信息
    Node    ast.Node       // 当前正在处理的 AST 节点 (通常是 *ast.CallExpr)
    PkgName string         // 当前包名
}
```

### 2.2 RouteInfo
描述 HTTP 路由的基本信息。

```go
type RouteInfo struct {
    Method string // e.g., "GET", "POST"
    Path   string // e.g., "/api/v1/users"
}
```

### 2.3 ParamInfo
描述请求参数 (对应 Swagger `@Param`)。

```go
type ParamInfo struct {
    Name     string // 参数名
    In       string // 位置: "path", "query", "body", "header"
    Type     string // 类型: "string", "integer", "object"
    Required bool   // 是否必填
    Desc     string // 描述
    Schema   string // 如果是 body/object，这里存储 Go 结构体名称 (e.g., "UserRequest")
}
```

### 2.4 ResponseInfo
描述响应数据 (对应 Swagger `@Success`/`@Failure`)。

```go
type ResponseInfo struct {
    Code    int    // HTTP 状态码
    Schema  string // 响应结构体名称 (e.g., "UserResponse")
    IsArray bool   // 是否返回数组
    Desc    string // 描述
}
```

## 3. Interfaces (接口定义)

### 3.1 Driver (Core)
这是所有驱动必须实现的入口接口。

```go
type Driver interface {
    // CheckNode 检查给定的 AST 节点是否是路由注册点。
    // 如果是，返回一个 Handler 实例和 true；否则返回 nil, false。
    CheckNode(ctx *Context) (Handler, bool)
}
```

### 3.2 Handler (Marker Interface)
代表一个被识别出的路由处理逻辑。具体的 Driver 实现需要返回实现此接口的对象。

```go
type Handler interface {
    // 这是一个标记接口，允许为空
}
```

### 3.3 Optional Parsers (Mix-ins)
Handler 实现类可以选择性地实现以下接口。

```go
// RouterParser 解析路由路径和方法
type RouterParser interface {
    ParseRouter(ctx *Context) RouteInfo
}

// SummaryParser 解析接口摘要
type SummaryParser interface {
    ParseSummary(ctx *Context) string
}

// DescriptionParser 解析详细描述
type DescriptionParser interface {
    ParseDescription(ctx *Context) string
}

// TagsParser 解析标签
type TagsParser interface {
    ParseTags(ctx *Context) []string
}

// ParamParser 解析请求参数
type ParamParser interface {
    ParseParams(ctx *Context) []ParamInfo
}

// ResponseParser 解析响应
type ResponseParser interface {
    ParseResponses(ctx *Context) []ResponseInfo
}
```

## 4. Tasks (研发任务)

请 **研发工程师** 按以下步骤完成代码编写：

- [ ] **Task 1**: 在 `pkg/driver/interface.go` 中导入必要的包 (`go/ast`, `go/token`)。
- [ ] **Task 2**: 定义上述所有 **数据结构** (`Context`, `RouteInfo`, `ParamInfo`, `ResponseInfo`)。
- [ ] **Task 3**: 定义 **核心接口** (`Driver`, `Handler`)。
- [ ] **Task 4**: 定义 **可选 Parser 接口** (`RouterParser`, `ParamParser` 等)。

**注意**: 代码必须包含清晰的注释，解释每个字段和接口的作用。
