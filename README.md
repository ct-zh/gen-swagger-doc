# Gen-Swagger-Doc

**Gen-Swagger-Doc** 是一个智能的 Swagger 文档自动生成工具，专为 Go 语言后端项目设计。

不同于传统的 `swag` 工具需要开发者手动编写大量繁琐的注释，**Gen-Swagger-Doc** 通过深度解析 Go 源代码（AST），自动理解路由结构、请求参数和响应类型，并为您生成标准规范的 Swagger 2.0 注释。

## 🚀 核心特性

*   **自动化注释注入**: 自动识别 Handler 函数，并在上方插入 `@Summary`, `@Router`, `@Param`, `@Success` 等注释。
*   **深度代码理解**:
    *   **路由解析**: 支持多级 `Group` 嵌套路由解析。
    *   **参数提取**: 自动识别 `Query`, `Param`, `BindJSON` 等参数绑定操作。
    *   **响应推断**: 自动分析 `JSON`, `XML` 响应中的结构体类型，甚至支持变量类型追溯。
*   **多框架支持**:
    *   **Gin**: 完美支持 (内置)。
    *   **Daenerys**: 支持内部自研框架 (code.nexita.net)。
    *   **可扩展架构**: 提供 Driver 接口，轻松适配 Echo, Fiber 或任何私有框架。
*   **非侵入式**: 仅在源代码中添加注释，不修改原有业务逻辑。

## 📦 安装

```bash
go install github.com/your-repo/gen-swagger-doc/cmd/swagger-gen@latest
```

## 🛠 使用指南

### 1. 命令行参数

```bash
swagger-gen -dir <项目路径> -framework <框架名称>
```

*   `-dir`: 指定要扫描的项目根目录（默认为当前目录 `.`）。
*   `-framework`: 指定使用的 Web 框架驱动。目前支持 `gin` 和 `daenerys`。

### 2. 示例：Gin 项目

```bash
# 进入项目目录
cd my-gin-project

# 运行生成器
swagger-gen -framework gin
```

**运行前:**

```go
func GetUser(c *gin.Context) {
    id := c.Query("id")
    c.JSON(200, UserResponse{ID: id})
}
```

**运行后:**

```go
// @Summary GET /users
// @Router /users [get]
// @Param id query string false ""
// @Success 200 {object} UserResponse
func GetUser(c *gin.Context) {
    id := c.Query("id")
    c.JSON(200, UserResponse{ID: id})
}
```

### 3. 示例：Daenerys 项目 (内部框架)

```bash
swagger-gen -dir ./services/novel-api -framework daenerys
```

支持特性：
*   **嵌套路由**: 完美解析 `api := s.GROUP(...)` -> `book := api.GROUP(...)` 的多级结构。
*   **Context 方法**: 支持 `QueryInt64`, `QueryString`, `JSON` 等方法分析。
*   **作用域追踪**: 即使在不同函数间传递 `Group` 对象，也能正确追踪路由前缀。

## 🏗 架构设计

本项目采用 **Core-Driver** 分层架构：

1.  **Core (Processor)**: 负责文件扫描、AST 解析、注释注入与持久化。
2.  **Driver (Adapter)**: 负责特定框架的语义理解。
    *   `FileParser`: 全文件扫描（处理全局变量、路由组）。
    *   `FuncBodyAnalyzer`: 函数体分析（提取参数、响应）。

### 添加新框架支持

只需实现 `pkg/driver.Driver` 接口即可：

```go
type MyDriver struct{}

func (d *MyDriver) CheckNode(ctx *driver.Context) (driver.Handler, bool) {
    // 识别路由注册点
}

// 可选：实现 FuncBodyAnalyzer 以支持参数提取
func (d *MyDriver) AnalyzeFunction(fn *ast.FuncDecl) ([]driver.ParamInfo, []driver.ResponseInfo) {
    // 分析函数体
}
```

## 📝 开发计划 (Roadmap)

- [x] **Gin Driver**: 基础路由与参数解析。
- [x] **Daenerys Driver**: 复杂嵌套路由支持。
- [x] **AST 增强**: 变量类型回溯解析。
- [ ] **CLI 优化**: 支持 Dry-Run 模式和覆盖策略配置。
- [ ] **更多框架**: 计划支持 Echo, Fiber。

---
**Happy Coding!** 减少文档编写时间，专注于业务逻辑。
