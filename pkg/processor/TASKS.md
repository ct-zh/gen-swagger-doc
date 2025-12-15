# Module: Processor Design

**Status**: Stable (V2)
**Author**: System Architect
**Module Path**: `pkg/processor`

## 1. Overview
本模块负责协调文件扫描、AST 解析、Driver 调用和代码修改。
**本次更新**: 配合 Driver 的升级，支持基于 Receiver 类型的精确 Handler 匹配。

## 2. Architecture

### 2.1 Route Map Key Design
原先使用 `string` (HandlerName) 作为 Key，导致冲突。
现在使用复合 Key：

```go
// 格式: "PkgName.ReceiverType.FunctionName"
// e.g. "main.UserAPI.List"
// e.g. "main..Pong" (无 Receiver)
func makeHandlerKey(info driver.HandlerInfo) string {
    return fmt.Sprintf("%s.%s.%s", info.PkgName, info.ReceiverType, info.FunctionName)
}
```

### 2.2 Execution Flow Update

**Phase 1 (Collection)**:
1.  调用 `HandlerInfoProvider.GetHandlerInfo` 获取详细信息。
2.  构建 Key，存入 `routeMap`。

**Phase 2 (Injection)**:
1.  遍历 AST 时，对于每个 `FuncDecl`：
    *   提取 FunctionName。
    *   提取 ReceiverType (如果 `fn.Recv` 不为空)。
        *   注意：AST 中 Receiver 可能是 `*StarExpr` (指针接收者) 或 `Ident` (值接收者)。需要提取基础类型名。
    *   构建 Key。
    *   在 `routeMap` 中查找。
2.  不再需要全局的 `funcCount` 统计来跳过冲突，因为 Key 已经足够唯一（在同一个包内）。
    *   *Edge Case*: 依然可能存在重载（Go 不支持）或同一个结构体定义了同名方法（编译错误），所以理论上 Key 是唯一的。

## 3. Tasks (研发任务)

### Phase 2: Enhanced Matching (Current Focus)

- [x] **Task 1**: 升级 `routeMap` 的 Key 生成逻辑。
- [x] **Task 2**: 升级 AST 遍历逻辑，解析 `FuncDecl` 的 Receiver 类型。
    - 编写辅助函数 `getReceiverTypeName(fn *ast.FuncDecl) string`。
- [x] **Task 3**: 移除旧的同名函数冲突检测代码。
- [x] **Task 4**: 更新集成测试 `test/integration_test.go`，启用之前的同名函数测试用例，并验证是否通过。
