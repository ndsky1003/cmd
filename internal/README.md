# internal 共享模块

这个模块包含所有工具共享的代码和工具函数。

## 目录结构

```
internal/
├── go.mod              # internal 模块定义
├── version/            # 版本相关功能
│   └── version.go      # 版本号获取函数
```

## 功能说明

### version.GetVersion()

统一的版本号获取函数，支持以下场景：

1. **GitHub Release 下载**：通过 ldflags 注入的版本号
2. **go install 安装**：从 Go modules build info 读取并提取版本号
3. **本地开发构建**：返回 "dev"

#### 使用方法

```go
import "github.com/ndsky1003/cmd/internal/version"

var Version = "dev"  // 可以通过 ldflags 注入

func main() {
    v := version.GetVersion(Version)
    fmt.Printf("version: %s\n", v)
}
```

#### 版本号提取逻辑

- 如果 `Version != "dev"`，直接返回（ldflags 注入的场景）
- 否则从 `debug.ReadBuildInfo()` 读取
- 从 `tool/v1.0.0` 格式提取纯版本号 `v1.0.0`
- 本地开发返回 "dev"

## 如何添加新的共享功能

1. 在 `internal/` 下创建新的包（如 `internal/utils`）
2. 实现共享函数
3. 在各工具的 `go.mod` 中添加引用：
   ```go
   require github.com/ndsky1003/cmd/internal v0.0.0
   replace github.com/ndsky1003/cmd/internal => ../internal
   ```
4. 在代码中 import 使用

## 依赖关系

```
launch     ──┐
filemgr    ──┼──> internal/version
structset  ──┘
```

每个工具模块通过 `replace` 指令引用本地的 internal 模块。
