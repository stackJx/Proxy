# Go 反向代理服务器

一个简单的 Go 语言反向代理服务器实现，支持多后端目标、基于路径的路由、日志记录和灵活的配置。  
为了解决内网项目的一些反代问题，项目包含了详细的日志系统、管理界面以及多平台构建支持。

## 功能特点

- 🔄 **多后端目标**：支持多个后端服务器的配置，基于路径前缀进行智能路由；使用最长前缀匹配路由算法。
- ⚡ ** HTTP/HTTPS 支持**：可选跳过 TLS 证书验证，适用于各种环境下的部署。
- 📝 **完整日志记录**：日志同时输出到控制台与文件。日志文件按日期存储于 `logs/` 目录中，命名格式为 `proxy_YYYY-MM-DD.log`。
- ⚙️ **JSON 配置文件**：所有服务设置均从 `config.json` 中加载，管理页面每次启动时会从 JSON 中读取并初始化页面数据。
- ⏱️ **灵活的超时设置**：提供可配置的读取、写入与空闲超时，以适应不同网络场景。
- 🔌 **连接池管理**：配置最大空闲连接数，有效利用系统资源。
- 📊 **请求/响应监控**：记录请求开始、结束时间、处理耗时、状态码等详细日志信息。
- 📂 **日志追踪与分组**：支持检测 `logs/` 目录下的日志文件，并按日期分组追踪，方便日志管理和问题排查。

## 安装说明

1. 克隆仓库：

   ```bash
   git clone [仓库地址]
   cd [仓库名称]
   ```

2. 构建项目：

### 单平台构建

```bash
# 当前平台构建
go build

# 指定输出文件名构建
go build -o go-proxy
```

### 多平台构建

#### Windows 平台构建命令

```bash
# Windows 64位系统构建命令
GOOS=windows GOARCH=amd64 go build -o go-proxy-windows-amd64.exe

# Windows 32位系统构建命令
GOOS=windows GOARCH=386 go build -o go-proxy-windows-386.exe

# Windows ARM64系统构建命令
GOOS=windows GOARCH=arm64 go build -o go-proxy-windows-arm64.exe
```

#### Linux 平台构建命令

```bash
# Linux 64位系统构建命令
GOOS=linux GOARCH=amd64 go build -o go-proxy-linux-amd64

# Linux 32位系统构建命令
GOOS=linux GOARCH=386 go build -o go-proxy-linux-386

# Linux ARM64系统构建命令（适用于树莓派等 ARM 设备）
GOOS=linux GOARCH=arm64 go build -o go-proxy-linux-arm64

# Linux ARM 32位系统构建命令
GOOS=linux GOARCH=arm go build -o go-proxy-linux-arm

# Linux MIPS 系统构建命令（适用于路由器等设备）
GOOS=linux GOARCH=mips go build -o go-proxy-linux-mips
GOOS=linux GOARCH=mipsle go build -o go-proxy-linux-mipsle
```

#### macOS 平台构建命令

```bash
# macOS Intel芯片构建命令
GOOS=darwin GOARCH=amd64 go build -o go-proxy-darwin-amd64

# macOS M系列芯片构建命令（M1/M2等）
GOOS=darwin GOARCH=arm64 go build -o go-proxy-darwin-arm64
```

#### 批量构建脚本

**Windows 批处理脚本 (build-all.bat):**
```batch
@echo off
setlocal

:: Windows 构建
set GOOS=windows
set GOARCH=amd64
go build -o go-proxy-windows-amd64.exe
set GOARCH=386
go build -o go-proxy-windows-386.exe
set GOARCH=arm64
go build -o go-proxy-windows-arm64.exe

:: Linux 构建
set GOOS=linux
set GOARCH=amd64
go build -o go-proxy-linux-amd64
set GOARCH=386
go build -o go-proxy-linux-386
set GOARCH=arm64
go build -o go-proxy-linux-arm64
set GOARCH=arm
go build -o go-proxy-linux-arm

:: macOS 构建
set GOOS=darwin
set GOARCH=amd64
go build -o go-proxy-darwin-amd64
set GOARCH=arm64
go build -o go-proxy-darwin-arm64

echo 构建完成！
```

**Linux/macOS 构建脚本 (build-all.sh):**
```bash
#!/bin/bash

# Windows 构建
GOOS=windows GOARCH=amd64 go build -o go-proxy-windows-amd64.exe
GOOS=windows GOARCH=386 go build -o go-proxy-windows-386.exe
GOOS=windows GOARCH=arm64 go build -o go-proxy-windows-arm64.exe

# Linux 构建
GOOS=linux GOARCH=amd64 go build -o go-proxy-linux-amd64
GOOS=linux GOARCH=386 go build -o go-proxy-linux-386
GOOS=linux GOARCH=arm64 go build -o go-proxy-linux-arm64
GOOS=linux GOARCH=arm go build -o go-proxy-linux-arm
GOOS=linux GOARCH=mips go build -o go-proxy-linux-mips
GOOS=linux GOARCH=mipsle go build -o go-proxy-linux-mipsle

# macOS 构建
GOOS=darwin GOARCH=amd64 go build -o go-proxy-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o go-proxy-darwin-arm64

echo "构建完成！"
```

## 配置说明

在项目根目录创建 `config.json` 配置文件。示例内容如下：

```json
{
  "listen_addr": ":8080",
  "enable_logs": true,
  "max_idle_conns": 100,
  "timeout": {
    "read_timeout": 30,
    "write_timeout": 30,
    "idle_timeout": 60
  },
  "targets": [
    {
      "name": "api-service",
      "url": "http://api.example.com",
      "path_prefix": "/api"
    },
    {
      "name": "web-service",
      "url": "https://web.example.com",
      "path_prefix": "/web"
    }
  ]
}
```

### 参数说明

- **listen_addr**: 代理服务器监听的地址和端口（例如：":8080"）。
- **enable_logs**: 是否启用详细日志记录，影响控制台和文件日志输出。
- **max_idle_conns**: 最大空闲连接数，可根据实际需求进行调整。
- **timeout**: 超时设置，单位均为秒：
  - **read_timeout**: 读取请求的最大时间。
  - **write_timeout**: 写入响应时间的最大值。
  - **idle_timeout**: 空闲连接最长保持时间。
- **targets**: 后端服务配置列表：
  - **name**: 后端服务标识，便于识别。
  - **url**: 后端服务地址（支持 HTTP 和 HTTPS）。
  - **path_prefix**: 用于匹配请求的路径前缀。

## 使用方法

1. 根据您的环境需求配置 `config.json` 文件。
2. 运行代理服务器：

在 Windows 系统：
```bash
go-proxy-windows-amd64.exe
```

在 Linux 系统：
```bash
chmod +x go-proxy-linux-amd64
./go-proxy-linux-amd64
```

在 macOS 系统：
```bash
chmod +x go-proxy-darwin-amd64
./go-proxy-darwin-amd64
```

服务器启动后会自动创建 `logs` 目录，并按天存储日志文件（格式为 `proxy_YYYY-MM-DD.log`）。

## 日志系统

代理服务器记录详细日志，包括：
- 请求的开始与结束时间
- 请求方法、路径、客户端信息及用户代理
- URL 重写日志（详情记录路径和目标服务器信息）
- 响应状态码、处理耗时
- 错误信息（如后端连接失败、TLS 验证错误、超时等）

日志输出方式：
1. **控制台**：标准输出实时显示日志信息
2. **文件日志**：按日期存储于 `logs/` 目录，可使用附带的日志检测工具对日志进行分组和追踪。

## 日志追踪工具

项目提供一个简单的日志检测工具，通过扫描 `logs/` 目录下符合 `proxy_YYYY-MM-DD.log` 格式的日志文件，按日期分组并显示文件大小及修改日期，方便你监控日志生成情况。例如：
```bash
go run logs/monitor_logs.go
```

## 管理界面

管理页面基于 HTML/JavaScript 实现，不依赖任何外部资源。  
每次页面启动时会自动通过 `/api/config` 接口从 JSON 文件加载初始配置，并允许用户修改后提交配置。  
提交的配置更新将实时反映在代理服务中。

## 项目结构

```
.
├── main.go           # 主程序代码，实现反向代理功能
├── config.json       # JSON 格式的配置文件
├── README.md         # 项目说明文档
├── build-all.bat     # Windows 批量构建脚本
├── build-all.sh      # Linux/macOS 批量构建脚本
├── logs/             # 日志目录，日志文件按天归档
│   └── proxy_*.log   # 示例：proxy_2025-03-07.log
├── static/           # 管理页面的静态资源目录
│   └── index.html    # 基于内嵌 CSS/JS 的管理界面，不依赖外部资源
└── (其他源代码文件)
```

## 开发指南

### 环境要求

- Go 1.x 或更高版本（建议使用最新稳定版本）
- 对 HTTP/HTTPS 及反向代理略有了解
- 推荐使用 VSCode、GoLand 或其他支持 Go 的 IDE

### 代码风格

- 代码格式化使用 `gofmt`
- 遵循 Go 官方的命名规范
- 保持适当的注释和文档

## 贡献指南

1. Fork 本仓库到您的 GitHub 账户
2. 克隆您 fork 的仓库到本地
3. 创建新的分支（例如：`git checkout -b feature/new-feature`）
4. 提交您的更改（`git commit -m "添加新特性"`）
5. 将分支推送到远程（`git push origin feature/new-feature`）
6. 通过 GitHub 创建新的 Pull Request

## 版本历史

- **1.0.0 (2025-02-08)**  
  初始版本发布，包含基本反向代理、日志系统、管理界面及多平台构建支持。

## 开源许可

本项目采用 MIT 许可证，详情请查看 [LICENSE](LICENSE) 文件。

## 作者

[@stackJx](https://github.com/stackJx)

## 致谢

- 感谢 Go 标准库提供的强大 HTTP 处理能力
- 感谢所有反馈建议和贡献代码的开发者

## 最后更新

最后更新时间：2025-02-08 08:17:34 UTC