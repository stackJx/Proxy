
# Go 反向代理服务器

一个简单的 Go 语言反向代理服务器实现，支持多后端目标、基于路径的路由、日志记录和灵活的配置。
为了解决内网项目的一些反代问题。

## 功能特点

- 🔄 支持多后端目标服务器，基于路径的智能路由
- ⚡ 支持 HTTP/HTTPS，可选择跳过 TLS 证书验证
- 📝 完整的日志系统（控制台 + 文件日志）
- ⚙️ 基于 JSON 的配置文件
- ⏱️ 可配置的超时设置（读取、写入、空闲连接）
- 🔌 连接池管理，可配置空闲连接数
- 🎯 最长前缀匹配的路由算法
- 📊 请求/响应监控，包含耗时统计

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

# Linux ARM64系统构建命令（适用于树莓派等ARM设备）
GOOS=linux GOARCH=arm64 go build -o go-proxy-linux-arm64

# Linux ARM 32位系统构建命令
GOOS=linux GOARCH=arm go build -o go-proxy-linux-arm

# Linux MIPS系统构建命令（适用于路由器等设备）
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

Windows 批处理脚本 (build-all.bat):
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

Linux/macOS 构建脚本 (build-all.sh):
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

在项目根目录创建 `config.json` 配置文件。配置示例：

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

### 配置参数说明

- `listen_addr`: 代理服务器监听的地址和端口（例如：":8080"）
- `enable_logs`: 是否启用详细日志记录（true/false）
- `max_idle_conns`: 最大空闲连接数，建议根据实际情况调整
- `timeout`: 超时设置（所有时间单位均为秒）
    - `read_timeout`: 读取整个请求的最大时间
    - `write_timeout`: 写入响应的最大时间
    - `idle_timeout`: 空闲连接的最大保持时间
- `targets`: 后端服务配置数组
    - `name`: 后端服务标识符（便于识别和管理）
    - `url`: 后端服务 URL（支持 HTTP/HTTPS）
    - `path_prefix`: 用于路由请求的路径前缀

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

服务器启动后会自动创建 `logs` 目录，并按天存储日志文件，格式为 `proxy_YYYY-MM-DD.log`。

## 日志系统

代理服务器提供详细的日志记录功能，包括但不限于：
- 请求开始和结束的时间戳
- 请求的方法（GET、POST等）和访问路径
- 客户端的远程地址和用户代理信息
- 响应状态码（200、404、502等）
- 请求处理的总耗时
- URL 重写的详细信息
- 错误详情（如果发生错误）

所有日志会同时输出到：
1. 控制台（标准输出）
2. 按天归档的日志文件中（logs目录下）

## 错误处理

代理服务器包含完善的错误处理机制：
- 当后端 URL 无效或无法连接时返回 502 Bad Gateway
- 当请求的路径没有匹配的后端服务时返回 404 Not Found
- 所有连接问题都会记录详细的错误信息到日志
- 对 TLS 证书验证错误进行优雅处理
- 超时处理机制可防止连接挂起

## 开发指南

### 环境要求

- Go 1.x 或更高版本（推荐使用最新的稳定版本）
- 基本的 HTTP/HTTPS 协议知识
- 反向代理相关概念理解
- 用于编辑代码的文本编辑器或 IDE（推荐使用 VSCode 或 GoLand）

### 项目结构

```
.
├── main.go           # 主程序代码
├── config.json       # 配置文件
├── build-all.bat     # Windows 批量构建脚本
├── build-all.sh      # Linux/macOS 批量构建脚本
├── logs/            # 日志目录
│   └── proxy_*.log  # 按天存储的日志文件
└── README.md        # 说明文档
```

### 代码风格

本项目遵循 Go 语言标准的代码风格：
- 使用 `gofmt` 进行代码格式化
- 遵循官方的命名规范
- 包含适当的注释和文档字符串

## 贡献指南

1. Fork 本仓库到您的 GitHub 账户
2. 克隆您 fork 的仓库到本地
3. 创建新的特性分支 (`git checkout -b feature/amazing-feature`)
4. 提交您的更改 (`git commit -m '添加某个很棒的特性'`)
5. 将更改推送到分支 (`git push origin feature/amazing-feature`)
6. 通过 GitHub 创建新的 Pull Request

## 版本历史

- 1.0.0 (2025-02-08)
    - 初始版本发布
    - 支持基本的反向代理功能
    - 实现日志系统
    - 添加多平台构建支持

## 开源许可

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详细信息

## 作者

[@stackJx](https://github.com/stackJx)

## 致谢

- Go 标准库提供的出色 HTTP 处理能力
- 所有提供反馈和建议的用户
- 为项目做出贡献的开发者

## 最后更新

最后更新时间：2025-02-08 08:17:34 UTC
