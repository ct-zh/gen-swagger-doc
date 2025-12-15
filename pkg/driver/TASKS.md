# Module: Driver Interface Design

**Status**: Planning (V2)
**Author**: System Architect
**Module Path**: `pkg/driver`

## 1. Overview
本模块定义了 `Driver` 接口和相关数据结构，用于适配不同的 Web 框架（如 Gin, Echo, Fiber）。
**本次更新**: 针对同名函数冲突问题，升级接口定义，引入 Receiver 类型识别。

## 2. Architecture

### 2.1 Core Interface
`Driver` 接口保持不变。

### 2.2 Data Structures
新增 `HandlerInfo` 结构体，替代原先只返回 `string` 的做法。

```go
// HandlerInfo 包含 Handler 的详细定位信息
type HandlerInfo struct {
    PkgName      string // 包名 (e.g., "main", "user")
    FunctionName string // 函数名 (e.g., "List", "Create")
    ReceiverType string // 接收者类型名 (e.g., "UserAPI", "OrderAPI")，如果是普通函数则为空
}
```

### 2.3 Mix-in Interfaces (Extension Points)

修改 `HandlerNameProvider` 接口：

```go
// HandlerInfoProvider 提供获取路由处理函数详细信息的能力
// (原 HandlerNameProvider 的升级版)
type HandlerInfoProvider interface {
    GetHandlerInfo(ctx *Context) HandlerInfo
}
```

## 3. Implementation Strategy (Gin Driver)

在 `GinHandler.GetHandlerInfo` 中：
1.  **普通函数**: `r.GET("/path", MyFunc)` -> `ReceiverType: ""`
2.  **方法调用**: `r.GET("/path", api.List)`
    *   `api` 是一个变量。我们需要找到 `api` 的定义。
    *   在 AST 中向上查找（在当前 Block 或全局变量中），找到 `api := &UserAPI{}` 或 `var api UserAPI`。
    *   提取类型名 `UserAPI`，赋值给 `ReceiverType`。
    *   **难点**: 这涉及到简单的 **Symbol Resolution (符号解析)**。如果不想引入完整的 `go/types`，我们可以实现一个轻量级的 AST 查找器。

## 4. Tasks (研发任务)

### Phase 2: Enhanced Parsing (Current Focus)

- [x] **Task 1**: 更新 `pkg/driver/interface.go`。
    - 定义 `HandlerInfo` 结构体。
    - 新增 `HandlerInfoProvider` 接口 (为了兼容性，可以保留 `HandlerNameProvider`，但标记为 Deprecated)。
- [x] **Task 2**: 更新 `GinDriver` 实现。
    - 实现 `GetHandlerInfo`。
    - 实现轻量级的 Symbol Resolution：
        - 检查 `CallExpr` 的 Receiver (e.g., `api` in `api.List`)。
        - 遍历当前文件的 AST (`ctx.FileSet` 对应的 File)，查找变量声明 (`AssignStmt` 或 `GenDecl`)。
        - 提取类型名称。
- [x] **Task 3**: 更新 `Processor` 逻辑。
    - 使用 `HandlerInfo` 而不是简单的 `string` 作为 Map Key。
    - Key 格式建议: `PkgName.ReceiverType.FunctionName` (e.g., "main.UserAPI.List") 或 "main..List"。
    - 在遍历 AST 查找 `FuncDecl` 时，同时检查 `FuncDecl.Recv`。
