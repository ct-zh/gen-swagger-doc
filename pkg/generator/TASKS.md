# Module: Generator Design

**Status**: Stable (MVP)
**Author**: System Architect
**Module Path**: `pkg/generator`

## 1. Overview
本模块负责将结构化的路由信息（`RouteInfo`, `ParamInfo`, `ResponseInfo` 等）格式化为标准的 Swagger 2.0 注释块。它是纯函数式的模块，不涉及 IO 操作。

## 2. Architecture

### 2.1 Interface
虽然本模块目前只是一个简单的工具包，但为了将来支持 OpenAPI 3.0 或其他格式，我们定义一个简单的生成器接口（或者直接提供函数）。

```go
// GeneratorConfig 配置项
type GeneratorConfig struct {
    Indent string // 缩进字符，默认 "    " (4 spaces) or "\t"
}

// GenerateSwaggerDocs 根据提供的信息生成注释块
// lines 是生成的注释行列表 (不包含 "// " 前缀，由调用方决定如何添加)
func GenerateSwaggerDocs(route driver.RouteInfo, params []driver.ParamInfo, responses []driver.ResponseInfo) []string
```

### 2.2 Formatting Rules
生成的注释应该遵循 [swag](https://github.com/swaggo/swag) 的标准格式：

```go
// @Summary ...
// @Description ...
// @Tags ...
// @Accept json
// @Produce json
// @Param ...
// @Success ...
// @Router /path [method]
```

## 3. Tasks (研发任务)

请 **研发工程师** 按以下步骤完成代码编写：

- [x] **Task 1**: 创建 `pkg/generator/generator.go`。
- [x] **Task 2**: 实现 `GenerateSwaggerDocs` 函数。
    - [x] 生成 `@Router` 行。
    - [x] 生成 `@Summary` (如果为空，可以使用 "HandlerName" 占位)。
    - [x] 生成 `@Success 200 {object} JSONResult` (默认占位，后续由 ResponseParser 提供)。
- [x] **Task 3**: 编写单元测试 `generator_test.go`，验证生成的字符串格式是否正确。
