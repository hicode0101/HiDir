//go:build !windows

package view

import (
	"os"

	"golang.org/x/sys/unix"
)

// ansiEnabled 表示终端是否支持 ANSI 转义序列（类 Unix 终端原生支持）。
var ansiEnabled = func() bool {
	// dumb 终端不支持转义序列
	term := os.Getenv("TERM")
	return term != "dumb" && term != ""
}()

// queryTerminalWidth 查询标准输出终端宽度。
func queryTerminalWidth() (int, bool) {
	// TIOCGWINSZ 的请求值因平台而异，由 x/sys/unix 按平台提供
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 {
		return 0, false
	}
	return int(ws.Col), true
}
