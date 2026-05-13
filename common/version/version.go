package version

import (
	"runtime/debug"
	"strings"
)

// GetVersion 返回当前工具的版本号
// 支持以下场景：
// 1. GitHub Release 下载：通过 ldflags 注入的版本
// 2. go install 安装：从 Go modules build info 读取
// 3. 本地开发：返回 "dev"
func GetVersion(toolVersion string) string {
	// 1. 如果通过 ldflags 注入了版本，直接使用
	if toolVersion != "dev" {
		return toolVersion
	}

	// 2. 否则尝试从 Go modules 构建信息读取
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	// 3. 如果是通过 go install 安装的，info.Main.Version 会包含版本信息
	//    例如: launch/v1.2.3, filemgr/v1.0.0, 或者 devel (如果是本地开发)
	if info.Main.Version != "(devel)" {
		// 从 tool/v1.0.0 格式中提取 v1.0.0
		version := info.Main.Version
		// 检查是否包含斜杠（工具名/版本号）

		if _, after, b := strings.Cut(version, "/"); b {
			return after
		}
		return version
	}

	// 4. 本地开发，返回 dev
	return "dev"
}
