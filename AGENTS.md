# AGENTS.md

## 多模块结构

无根 `go.mod`、无 `go.work`，构建必须在各工具目录内执行。共 4 个独立 Go module：

| 目录 | module | Go 版本 |
|------|--------|---------|
| `launch/` | `github.com/ndsky1003/cmd/launch` | 1.25.7 |
| `filemgr/` | `github.com/ndsky1003/cmd/filemgr` | 1.25.7 |
| `structset/` | `github.com/ndsky1003/cmd/structset` | 1.25.7 |
| `common/` | `github.com/ndsky1003/cmd/common` | 1.22 |

## 构建

```bash
cd launch && go build -o launch .
cd filemgr && go build -o filemgr .
cd structset && go build -o structset .
```

发布构建通过 ldflags 注入版本号（见 `.github/workflows/build-on-tag.yml`）：

```bash
go build -ldflags="-s -w -X main.Version=v1.0.0" -o launch .
```

## 内部依赖

- `common/` 是已发布模块 `github.com/ndsky1003/cmd/common v1.0.0`，三个工具**直接 require、无 replace**。改动 `common/` 后本地构建不会自动生效，需发布新版本或临时加 `replace`。
- 公共版本逻辑位于 `common/version` 包（`GetVersion(Version)`）。
- `filemgr/go.mod` 含 `replace github.com/ndsky1003/crpc/v3 => ../../crpc`，依赖仓库外的兄弟目录 `crpc`（本地构建与 CI 需先 checkout 该目录）。

## 版本管理

- Tag 格式：`tool/vX.Y.Z`（如 `launch/v1.0.5`）
- 各工具 `main.go` 声明 `var Version = "dev"`
- `common/version.GetVersion(Version)` 解析优先级：ldflags 注入 > go.mod build info > `"dev"`
- CI 通过 `gh release create` 自动创建 Release

## CI

`.github/workflows/build-on-tag.yml` — tag push 触发，前缀匹配 `launch/**`、`filemgr/**`、`structset/**`；构建 linux/darwin amd64+arm64 并发布 Release。

## 格式化

`goimports -w .`（无独立 lint/typecheck 配置）。

## 测试

- `launch/logger/` 有 `*_test.go`（覆盖轮转、清理、压缩、并发写），运行：`cd launch && go test ./logger/`（含 `-race`）。
- 其余目录无测试。

## launch 日志子包

- `launch/logger/`（`github.com/ndsky1003/cmd/launch/logger`）包含两个策略 logger：
  - `SizeLogger`（`size_logger.go`）：按大小轮转，可选 `Interval` 字段与时间边界混合（先到先触发）。
  - `IntervalLogger`（`interval_logger.go`）：按时间间隔轮转（`time.Hour`/`24*time.Hour`/`7*24*time.Hour` 或任意 duration）。
- `launch` 通过 `-split` flag 选择：`size|hour|day|week|<duration>`，`+` 组合表示混合（如 `size+day`）。
- 两个 logger 均可通过 `OnError func(error)` 字段接收清理/压缩失败回调（未设置时用标准库 `log` 输出）。

## 其他

- `gencrpc/` 已废弃（未纳入 git，目录残留），不应修改。
- `filemgr` 依赖 `github.com/ndsky1003/crpc/v3`，需 server/client 模式：server `-server -suris=:18083`，client `-client -curis=... -root=... -secret=...`；`-web -wuris` 启用 Web UI。`web_index.html` 通过 `//go:embed` 内嵌。
- `structset` 通过 `//go:embed struct_template.go.tmpl` 内嵌模板，生成后调用 `goimports -w` 格式化输出。flag：`-f` 输入文件、`-o` 输出文件、`-sub` 后缀、`-tagname` 标签名，`-k/-jm/-bm/-im/-bd/-id/-s/-i/-a/-c` 控制各生成方法。
