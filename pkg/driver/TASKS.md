# Module: Driver Interface Design

**Status**: Stable (MVP)
**Author**: System Architect
**Module Path**: `pkg/driver`

## 1. Overview
本模块定义了 `Driver` 接口和相关数据结构，用于适配不同的 Web 框架（如 Gin, Echo, Fiber）。

## 2. Architecture

### 2.1 Core Interface
`Driver` 接口是入口，`Handler` 是核心抽象。

```go
type Driver interface {
    CheckNode(ctx *Context) (Handler, bool)
}

type Handler interface {
    // Empty marker interface
}
```

### 2.2 Mix-in Interfaces (Extension Points)
为了支持不同的解析能力，我们使用 Mix-in 模式：

*   `HandlerNameProvider`: 获取函数名 (MVP 必需)
*   `RouterParser`: 解析 HTTP Method 和 Path (MVP 必需)
*   `ParamParser`: 解析请求参数 (TODO)
*   `ResponseParser`: 解析响应 (TODO)

## 3. Tasks (研发任务)

### Phase 1: MVP (已完成)
- [x] **Task 1**: 定义 `Context`, `RouteInfo` 等基础结构体。
- [x] **Task 2**: 定义 `Driver` 和 `Handler` 接口。
- [x] **Task 3**: 定义 Mix-in 接口 (`HandlerNameProvider`, `RouterParser` 等)。

### Phase 2: Enhanced Parsing (TODO)
- [ ] **Task 4**: 完善 `GinDriver`，实现 `ParamParser` 接口。
    - 尝试从 AST 中解析 `c.Query("id")`, `c.Param("name")` 等调用，自动填充 `ParamInfo`。
- [ ] **Task 5**: 完善 `GinDriver`，实现 `ResponseParser` 接口。
    - 尝试解析 `c.JSON(200, obj)`，提取状态码和响应对象类型。
- [ ] **Task 6**: 解决同名函数冲突问题。
    - 扩展 `HandlerNameProvider` 或新增接口，使其能返回 Receiver 的类型信息。
    - 配合 `Processor` 使用 `go/types` 或增强的 AST 查找逻辑来精确匹配 Handler。
