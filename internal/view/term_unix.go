//go:build !windows

package view

import (
	"os"
	"syscall"
	"unsafe"
)

// ansiEnabled 表示终端是否支持 ANSI 转义序列（类 Unix 终端原生支持）。
var ansiEnabled = func() bool {
	// dumb 终端不支持转义序列
	term := os.Getenv("TERM")
	return term != "dumb" && term != ""
}

// queryTerminalWidth 查询标准输出终端宽度。
func queryTerminalWidth() (int, bool) {
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}
	ws := &winsize{}
	// TIOCGWINSZ 在 Windows 与 Unix 上的值不同，这里按平台选择
	ret, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		os.Stdout.Fd(),
		tiocgwinsz,
		uintptr(unsafe.Pointer(ws)),
	)
	if ret != 0 || ws.Col == 0 {
		return 0, false
	}
	return int(ws.Col), true
}
