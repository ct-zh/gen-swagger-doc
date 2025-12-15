# Gen-Swagger-Doc

**Gen-Swagger-Doc** 是一个基于 Go AST (Abstract Syntax Tree) 的自动化工具，旨在为 Go 语言项目（支持 Gin, Echo 等主流框架及私有框架）自动生成符合 Swagger 2.0 标准的注释文档。

通过解析源代码中的路由注册、请求参数绑定和响应结构，本工具可以自动插入 `@Summary`, `@Router`, `@Param`, `@Success` 等注释，从而配合 `swag` 工具生成最终的 API 文档，极大地减少手动编写文档的工作量。

## 核心特性

*   **框架无关 (Framework Agnostic)**: 核心逻辑与框架解耦，通过 `FrameworkDriver` 接口适配不同框架。
*   **AST 深度解析**: 不仅仅是正则匹配，而是真正理解 Go 代码结构，能够准确识别函数调用、结构体定义和参数类型。
*   **CI/CD 友好**: 支持作为 CLI 工具直接运行，或作为 SDK 集成到私有项目的构建流程中。
*   **可扩展**: 针对私有框架，只需实现简单的 Driver 接口即可无缝接入。

## 架构设计

本项目采用分层架构，确保核心逻辑的复用性和扩展性。

### 1. 核心层 (Core / Processor)
位于 `pkg/processor`。
*   负责遍历文件系统，读取 `.go` 源码。
*   调用 `go/parser` 将源码解析为 AST。
*   协调 Driver 进行代码分析。
*   负责将生成的注释块注入回 AST 并保存文件。

### 2. 驱动层 (Driver Interface)
位于 `pkg/driver`。
采用 **Mix-in 接口模式**，允许用户按需实现解析逻辑。

*   **核心接口**:
    *   `Driver`: 入口接口，只需实现 `CheckNode` 方法，用于识别 AST 节点是否为路由注册点。
*   **可选能力接口 (按需实现)**:
    *   `RouterParser`: 解析路由路径和 HTTP 方法 (`@Router`)。
    *   `SummaryParser`: 解析接口摘要 (`@Summary`)。
    *   `ParamParser`: 解析请求参数 (`@Param`)。
    *   `ResponseParser`: 解析响应结构 (`@Success`)。

这种设计使得用户可以只实现自己关心的部分，工具会自动检测并聚合已实现的能力。

### 3. 实现层 (Implementation)
位于 `pkg/driver/{framework}`。
*   **Gin Driver**: 内置支持，识别 `gin.Context` 相关操作。
*   **Echo Driver**: (计划中) 支持 Echo 框架。
*   **Custom Driver**: 用户可编写自己的驱动适配私有框架。

## 目录结构

```text
gen-swagger-doc/
├── cmd/
│   └── swagger-gen/       # 标准 CLI 工具入口
│       └── main.go
├── pkg/
│   ├── driver/            # 驱动接口定义
│   │   ├── interface.go
│   │   └── gin/           # Gin 框架适配器实现
│   ├── processor/         # 核心处理逻辑 (扫描、解析、注入)
│   └── generator/         # Swagger 注释生成逻辑
├── go.mod
└── README.md
```

## 使用指南

### 1. 安装
```bash
go install github.com/your-repo/gen-swagger-doc/cmd/swagger-gen@latest
```

### 2. 基本使用 (针对 Gin 项目)
```bash
swagger-gen -dir ./src -framework gin
```
该命令会扫描 `./src` 目录下的所有 Go 文件，识别 Gin 路由，并在 Handler 函数上方自动插入 Swagger 注释。

### 3. 接入 CI/CD
在 `.gitlab-ci.yml` 或 GitHub Actions 中：
```yaml
generate_docs:
  script:
    - swagger-gen -dir ./ -framework gin
    - swag init  # 使用 swaggo 生成最终 json/html
    - git diff --exit-code # 检查是否有未提交的文档变更
```

### 4. 适配私有框架 (SDK 模式)
如果在私有项目中使用自研框架，可以创建一个 `tools/doc.go` 文件：

```go
package main

import (
    "github.com/your-repo/gen-swagger-doc/pkg/processor"
    "github.com/your-repo/gen-swagger-doc/pkg/driver"
)

// 定义私有驱动
type MyDriver struct{}
// 实现 driver.FrameworkDriver 接口...

func main() {
    processor.Run(processor.Options{
        Driver: &MyDriver{},
        Path:   "./src",
    })
}
```
运行 `go run tools/doc.go` 即可。

## 开发路线图

- [ ] **Phase 1**: 定义 `FrameworkDriver` 接口与基础架构。
- [ ] **Phase 2**: 实现 AST 文件扫描与 Import 别名解析。
- [ ] **Phase 3**: 实现 Gin 驱动 (路由识别、参数提取)。
- [ ] **Phase 4**: 实现 Swagger 注释模板渲染与 AST 注入。
- [ ] **Phase 5**: CLI 封装与代码回写功能。
