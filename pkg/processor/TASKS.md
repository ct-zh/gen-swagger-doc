# Module: Processor Design

**Status**: In Progress
**Author**: System Architect
**Module Path**: `pkg/processor`

## 1. Overview
本模块负责协调文件扫描、AST 解析、Driver 调用和代码修改。它是 CLI 工具直接调用的核心库。

## 2. Architecture

### 2.1 Options
```go
type Options struct {
    WorkDir string         // 工作目录
    Driver  driver.Driver  // 使用的驱动实例 (e.g., GinDriver)
}
```

### 2.2 Execution Flow
1.  `Run(opts)` 启动。
2.  **Phase 1 (Collection)**:
    *   遍历所有 `.go` 文件。
    *   统计 `FuncDecl` 出现次数 (用于检测同名函数冲突)。
    *   使用 `Driver` 解析路由注册代码，建立 `HandlerName -> RouteInfo` 映射。
3.  **Phase 2 (Injection)**:
    *   再次遍历所有 `.go` 文件 (使用新的 `FileSet` 以避免偏移量问题)。
    *   找到对应的 Handler 函数定义 (`*ast.FuncDecl`)。
    *   检查是否存在同名函数冲突 (Ambiguous Handler)。如果是，跳过并警告。
    *   生成 Swagger 注释 (调用 `pkg/generator`)。
    *   将注释附加到 `FuncDecl.Doc`。
    *   如果文件被修改，使用 `go/format` 写回磁盘。

## 3. Key Challenges & Solutions

### 3.1 Finding the Handler Function
Driver 返回的是路由注册点（e.g., `r.GET`），但我们需要给 Handler 函数（e.g., `func Pong(...)`）加注释。
**解决方案**:
1.  Driver 需要返回 Handler 函数的名称。
2.  在当前包的 AST 中查找名为 `Pong` 的 `FuncDecl`。
3.  *MVP 限制*: 暂时只支持 Handler 函数在同一个包内。
4.  *MVP 限制*: 如果存在同名函数（例如不同结构体的同名方法），暂不支持区分，直接跳过以保证安全。

## 4. Tasks (研发任务)

请 **研发工程师** 按以下步骤完成代码编写：

- [x] **Task 1**: 创建 `pkg/processor/processor.go`。定义 `Options` 和 `Run` 函数。
- [x] **Task 2**: 实现 AST 遍历逻辑和 Handler 查找。
- [x] **Task 3**: 实现简单的注释生成逻辑 (目前是硬编码)。
- [x] **Task 4**: 实现 AST 修改和回写逻辑 (已解决 FileSet 偏移量 Bug)。
- [x] **Task 5**: 实现同名函数冲突检测 (Ambiguous Handler Check)。
- [x] **Task 6**: 集成 `pkg/generator` 模块。
    - 将硬编码的 `// @Router` 生成逻辑替换为调用 `generator.GenerateSwaggerDocs`。
- [ ] **Task 7**: 优化：支持跨文件/跨包查找 Handler (V2 规划)。

**注意**:
目前 `Processor` 已经能够通过集成测试 `test/integration_test.go`。
接下来的重点是接入 `pkg/generator` 以生成完整的注释块。
