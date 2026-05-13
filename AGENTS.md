# AGENTS.md

## 多模块结构

无根 `go.mod`，5 个独立 Go module：

| 目录 | module | Go 版本 |
|------|--------|---------|
| `launch/` | `github.com/ndsky1003/cmd/launch` | 1.25.7 |
| `filemgr/` | `github.com/ndsky1003/cmd/filemgr` | 1.25.7 |
| `filemgr_front/` | `github.com/ndsky1003/cmd/filemgr_front` | 1.24 |
| `structset/` | `github.com/ndsky1003/cmd/structset` | 1.25.7 |
| `common/` | `github.com/ndsky1003/cmd/common` | 1.22 |

无 `go.work`，构建必须在各工具目录内执行。

## 构建

```bash
cd launch && go build -o launch .
cd filemgr && go build -o filemgr .
cd structset && go build -o structset .
cd filemgr_front && wails build
```

发布构建通过 ldflags 注入版本号：
```bash
go build -ldflags="-s -w -X main.Version=v1.0.0" -o launch .
```

## 内部依赖

各工具通过 `replace` 引用 `internal/`：

```
require github.com/ndsky1003/cmd/common v0.0.0
replace github.com/ndsky1003/cmd/common => ../common
```

修改 `internal/` 后需在工具目录执行 `go mod tidy`。

## 版本管理

- Tag 格式：`tool/vX.Y.Z`（如 `launch/v1.0.5`）
- 各工具声明 `var Version = "dev"`（`main.go`）
- `internal/version.GetVersion(Version)` 按优先级解析：ldflags 注入 > go.mod build info > `"dev"`
- CI 通过 `gh release create` 自动创建 Release

## CI

`.github/workflows/build-on-tag.yml` — tag push 触发，前缀匹配 `launch/**`、`filemgr/**`、`structset/**`。构建 linux/darwin amd64+arm64 并发布 Release。

## 格式化

`goimports -w .`（README 中注明，无独立配置）。

## 测试

全仓库无 `*_test.go`，无需运行测试。

## 其他

- `gencrpc/` 目录已废弃（commit `5fe7553` 删除，目录残留），不应修改。
- `filemgr` 依赖 `github.com/ndsky1003/crpc/v3`，运行需要 server/client 模式（`-isserver` / `-urls`）。
- `filemgr_front` 是基于 Wails 的桌面 GUI，通过 crpc/v3 连接 `filemgr` 服务。构建使用 `wails build`。
- `structset` 使用 `//go:embed struct_template.go.tmpl` 内嵌模板，生成后调用 `goimports -w` 格式化输出。
