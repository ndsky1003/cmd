# cmd

> 一组实用的 Go 命令行工具集合

这个仓库包含多个独立的 Go 命令行工具，每个工具都有自己的版本号和独立的功能。这些工具旨在简化日常开发任务和提升工作效率。

## 📦 包含的工具

### 🚀 [launch](./launch/)

带有日志功能的进程启动器，可以管理子进程并记录其输出。

**功能特性：**
- 启动并管理子进程
- 自动记录 stdout/stderr 到日志文件
- 支持日志轮转（大小、时间、备份数量）
- 可配置的日志目录和文件名
- 自动过滤 `-r` 标志传递给子进程

**安装：**
```bash
go install github.com/ndsky1003/cmd/launch@latest
```

**使用：**
```bash
# 启动一个子进程并记录日志
launch -r /path/to/subprocess -dir=logdir -filename=app.log --arg1 --arg2

# 查看版本
launch -v
```

---

### 📁 [filemgr](./filemgr/)

基于 crpc 的远程文件管理服务。

**功能特性：**
- 客户端/服务器模式
- 目录列表和创建
- 文件上传（支持分块）
- 基于 secret 的身份验证
- 通过 crpc 框架通信

**安装：**
```bash
go install github.com/ndsky1003/cmd/filemgr@latest
```

**使用：**
```bash
# 服务器模式
filemgr -isserver -uris=:18083 -name=filemgr

# 客户端模式
filemgr -urls=127.0.0.1:18083 -root=/path/to/root -secret=your_secret

# 查看版本
filemgr -v
```

---

### 🛠️ [structset](./structset/)

Go 结构体代码生成工具，自动生成 Set、Inc、Add、Copy 等方法。

**功能特性：**
- 生成 `Set()` 方法设置字段值
- 生成 `Inc()` 方法递增数值字段
- 生成 `Add()` 方法添加到切片/Map
- 生成 `Copy()` 方法复制结构体
- 支持自定义标签控制生成行为
- 字段级别属性：`inc`, `noinc`, `add`, `noadd`, `copy`, `nocopy`

**安装：**
```bash
go install github.com/ndsky1003/cmd/structset@latest
```

**使用：**
```bash
# 生成当前文件的代码
structset -f=input.go -o=output_gen.go

# 查看版本
structset -v

# 在 Go 文件中使用
//go:generate structset
```

**代码示例：**
```go
//go:generate structset
type User struct {
    Name     string `structset:"copy"`
    Age      int    `structset:"inc,add"`
    Email    string `structset:"nocopy"`
    Tags     []string `structset:"add"`
}
```

## 🚀 快速开始

### 安装

从源码安装最新版本：

```bash
# 安装 launch
go install github.com/ndsky1003/cmd/launch@latest

# 安装 filemgr
go install github.com/ndsky1003/cmd/filemgr@latest

# 安装 structset
go install github.com/ndsky1003/cmd/structset@latest
```

### 安装特定版本

```bash
# 安装指定版本（使用纯版本号）
go install github.com/ndsky1003/cmd/launch@v1.0.5
go install github.com/ndsky1003/cmd/filemgr@v1.0.5
go install github.com/ndsky1003/cmd/structset@v1.0.5
```

### 从 GitHub Release 下载

访问 [Releases 页面](https://github.com/ndsky1003/cmd/releases) 下载对应平台的预编译二进制文件。

```bash
# 下载并安装（Linux/macOS）
wget https://github.com/ndsky1003/cmd/releases/download/launch/v1.0.5/launch-linux-amd64
chmod +x launch-linux-amd64
sudo mv launch-linux-amd64 /usr/local/bin/launch
```

## 📋 系统要求

- Go 1.22 或更高版本（用于从源码编译）
- Linux、macOS 或 Windows

## 🔧 开发

### 本地开发

```bash
# 克隆仓库
git clone https://github.com/ndsky1003/cmd.git
cd cmd

# 进入工具目录
cd launch

# 编译
go build -o launch .

# 运行
./launch -v
```

### 运行测试

```bash
# 测试所有工具
go test ./...

# 测试特定工具
go test ./launch
go test ./filemgr
go test ./structset
```

### 代码格式化

```bash
# 使用 goimports 格式化代码
goimports -w .
```

## 📝 版本管理

本项目采用**多模块架构**，每个工具都是独立的 Go module，拥有独立的版本号。

### Tag 命名规范

```
launch/v1.0.5     → launch 工具的 v1.0.5 版本
filemgr/v2.0.1    → filemgr 工具的 v2.0.1 版本
structset/v3.1.0  → structset 工具的 v3.1.0 版本
```

### 安装方式

虽然 Git tag 使用 `launch/v1.0.5` 格式，但安装时使用简洁的版本号：

```bash
# 使用纯版本号安装（Go 会自动匹配对应的 tag）
go install github.com/ndsky1003/cmd/launch@v1.0.5
go install github.com/ndsky1003/cmd/filemgr@v1.0.5
go install github.com/ndsky1003/cmd/structset@v1.0.5
```

### 版本查看

所有工具都支持查看版本信息：

```bash
launch -v       # 输出: launch version v1.0.5
filemgr -v      # 输出: filemgr version v1.0.5
structset -v    # 输出: structset version v1.0.5
```

## 📚 文档

- [多模块架构指南](./MULTI_MODULE_GUIDE.md) - 详细的架构说明和使用指南
- [CLAUDE.md](./CLAUDE.md) - 项目概览和开发指南（用于 Claude Code）
- [VERSION_MANAGEMENT.md](./VERSION_MANAGEMENT.md) - 版本管理说明（旧版本文档，仅供参考）

## 🤝 贡献

欢迎贡献！请随时提交 Issue 或 Pull Request。

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 🔗 相关项目

- [crpc](https://github.com/ndsky1003/crpc) - 自定义 RPC 框架（filemgr 使用）

## 📮 联系方式

- GitHub: [@ndsky1003](https://github.com/ndsky1003)

---

**注意：** 这些工具持续开发中，API 和功能可能会变化。建议在生产环境使用前进行充分测试。
