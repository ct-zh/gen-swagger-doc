# Module: Gin Driver Implementation

**Status**: Ready for Dev
**Author**: System Architect
**Module Path**: `pkg/driver/gin`

## 1. Overview
本模块实现 `driver.Driver` 接口，专门用于解析 **Gin** 框架的路由注册代码。
MVP 阶段目标：能够识别标准 Gin 路由（如 `r.GET(...)`）并提取路径和方法。

## 2. Implementation Strategy

### 2.1 Struct Definition
我们需要定义一个 `GinDriver` 结构体和一个 `GinHandler` 结构体。

```go
type GinDriver struct{}

type GinHandler struct {
    ctx  *driver.Context
    node *ast.CallExpr // 存储路由注册的函数调用节点
}
```

### 2.2 Interface Implementation
`GinDriver` 需要实现 `driver.Driver`。
`GinHandler` 需要实现 `driver.Handler` 和 `driver.RouterParser`。

## 3. Logic Details

### 3.1 CheckNode (In GinDriver)
逻辑如下：
1.  检查 `ctx.Node` 是否为 `*ast.CallExpr` (函数调用)。
2.  检查调用的函数是否为 SelectorExpr (例如 `r.GET`)。
3.  检查 Method Name 是否为 HTTP 方法 (GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD)。
4.  如果是，返回 `&GinHandler{node: callExpr}`。

### 3.2 ParseRouter (In GinHandler)
逻辑如下：
1.  分析 `h.node.Args`。
2.  第一个参数通常是 Path (BasicLit string)。提取它。
3.  Method 就是函数调用的名字 (GET, POST...)。

## 4. Tasks (研发任务)

请 **研发工程师** 按以下步骤完成代码编写：

- [ ] **Task 1**: 创建 `pkg/driver/gin/driver.go`。
- [ ] **Task 2**: 实现 `GinDriver.CheckNode`。需要构建一个 `map[string]bool` 来存储支持的 HTTP 方法。
- [ ] **Task 3**: 实现 `GinHandler` 及其 `ParseRouter` 方法。
- [ ] **Task 4**: 编写单元测试 `pkg/driver/gin/driver_test.go`，构造一段伪造的 AST 代码，验证能否正确识别。

**Tip**: 在处理 Path 时，注意去除字符串字面量的引号 (e.g., `"/ping"` -> `/ping`)。可以使用 `strconv.Unquote`。
