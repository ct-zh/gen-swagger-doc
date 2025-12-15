# Module: Gin Driver Implementation

**Status**: Stable (MVP)
**Author**: System Architect
**Module Path**: `pkg/driver/gin`

## 1. Overview
本模块实现 `driver.Driver` 接口，专门用于解析 **Gin** 框架的路由注册代码。

## 2. Implementation Strategy

### 2.1 Struct Definition
```go
type GinDriver struct{}

type GinHandler struct {
    ctx  *driver.Context
    node *ast.CallExpr // 存储路由注册的函数调用节点
}
```

### 2.2 Interface Implementation
`GinDriver` 实现了 `driver.Driver`。
`GinHandler` 实现了：
*   `driver.Handler` (Marker)
*   `driver.RouterParser` (Path/Method)
*   `driver.HandlerNameProvider` (Function Name)

## 3. Tasks (研发任务)

### Phase 1: MVP (已完成)
- [x] **Task 1**: 创建 `pkg/driver/gin/driver.go`。
- [x] **Task 2**: 实现 `GinDriver.CheckNode`。
- [x] **Task 3**: 实现 `GinHandler` 及其 `ParseRouter` 和 `GetHandlerName` 方法。
- [x] **Task 4**: 编写单元测试 `pkg/driver/gin/driver_test.go`。

### Phase 2: Advanced Features (TODO)
- [ ] **Task 5**: 实现 `ParamParser`。
    - 需要遍历 `Handler` 函数体（不仅仅是路由注册处），查找 `c.Query`, `c.Param`, `c.ShouldBind` 等调用。
    - **难点**: Driver 目前只持有路由注册处的 AST (`CallExpr`)，而不知道 Handler 函数体的 AST。
    - **架构变更建议**: `CheckNode` 返回的 `Handler` 可能需要包含（或延迟获取）Handler 函数定义的 AST 节点。这需要与 `Processor` 协作。
- [ ] **Task 6**: 实现 `ResponseParser`。
    - 类似地，需要分析 Handler 函数体，查找 `c.JSON` 等调用。
