// Package assets 通过 go:embed 将词表数据库嵌入二进制文件，
// 使 HiDir 成为可以独立运行的单一可执行文件。
package assets

import (
	"embed"
)

// FS 是嵌入的词表数据库根目录（对应磁盘上的 db/ 目录）。
//
//go:embed db
var FS embed.FS

// DBDir 是嵌入文件系统中词表数据库的目录名。
const DBDir = "db"
