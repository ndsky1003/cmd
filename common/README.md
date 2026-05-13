# common

`github.com/ndsky1003/cmd/common` — 共享工具库，为 cmd 下的各工具模块提供通用基础功能。

## 安装

```bash
go get github.com/ndsky1003/cmd/common@latest
```

## 包

### `version` — 版本号获取

`version.GetVersion` 根据优先级返回当前工具的版本号：

1. **ldflags 注入** — `go build -ldflags="-X main.Version=v1.0.0"` 时直接返回
2. **go install** — 从 Go modules 构建信息中提取 `tool/v1.0.0` 格式的版本
3. **本地开发** — 返回 `"dev"`

```go
import "github.com/ndsky1003/cmd/common/version"

var VERSION = "dev"

func main() {
    v := version.GetVersion(VERSION)
    fmt.Println(v)
}
```

## 新增包

在 `common/` 下创建新目录和 `.go` 文件即可。各工具通过 `go mod tidy` 拉取更新：

```bash
cd filemgr
go get github.com/ndsky1003/cmd/common@latest
go mod tidy
```

## 依赖关系

```
launch     ──┐
filemgr    ──┼──> common
structset  ──┘
```
