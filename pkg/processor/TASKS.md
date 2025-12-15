# Module: Processor Design

**Status**: Ready for Dev
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
2.  使用 `filepath.Walk` 遍历 `WorkDir`。
3.  对每个 `.go` 文件调用 `processFile`。
4.  `processFile` 解析 AST。
5.  遍历 AST，对每个节点调用 `opts.Driver.CheckNode`。
6.  如果发现路由 (Handler != nil)：
    *   使用 `RouterParser` 提取 URL/Method。
    *   找到对应的 Handler 函数定义 (`*ast.FuncDecl`)。
    *   生成 Swagger 注释 (调用 `pkg/generator`)。
    *   将注释附加到 `FuncDecl.Doc`。
7.  如果文件被修改，使用 `go/format` 写回磁盘。

## 3. Key Challenges & Solutions

### 3.1 Finding the Handler Function
Driver 返回的是路由注册点（e.g., `r.GET`），但我们需要给 Handler 函数（e.g., `func Pong(...)`）加注释。
**解决方案**:
1.  Driver 需要返回 Handler 函数的名称 (这需要扩展 GinDriver 的能力，或者在 Processor 层做简单的 Ident 提取)。
2.  在当前包的 AST 中查找名为 `Pong` 的 `FuncDecl`。
3.  *MVP 限制*: 暂时只支持 Handler 函数在同一个包内。

## 4. Tasks (研发任务)

请 **研发工程师** 按以下步骤完成代码编写：

- [ ] **Task 1**: 创建 `pkg/processor/processor.go`。定义 `Options` 和 `Run` 函数。
- [ ] **Task 2**: 实现 AST 遍历逻辑。
- [ ] **Task 3**: 实现简单的注释生成逻辑 (暂时硬编码，或者简单的字符串拼接)。
- [ ] **Task 4**: 实现 AST 修改和回写逻辑。

**注意**:
目前 `GinDriver` 的 `ParseRouter` 只返回了 Path/Method，没有返回 Handler 函数名。
你需要修改 `pkg/driver/interface.go` 中的 `RouteInfo` 或者 `Handler` 接口，以便 Processor 知道去哪里加注释。
**架构师修正**: 建议在 `driver.Handler` 接口中增加一个可选接口 `HandlerNameParser`，或者直接在 `GinHandler` 中暴露获取函数名的方法。
为了保持通用性，我们定义一个新的 Mix-in:

```go
// pkg/driver/interface.go (Update required)
type HandlerNameProvider interface {
    GetHandlerName(ctx *Context) string
}
```
(请研发工程师先在 interface.go 中添加这个接口，然后更新 GinDriver 实现它)
