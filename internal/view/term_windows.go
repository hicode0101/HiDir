//go:build windows

package view

import (
	"os"
	"syscall"
	"unsafe"
)

// tiocgwinsz 是 Windows 上的占位（实际使用控制台缓冲区 API）。
const tiocgwinsz = 0

type consoleScreenBufferInfo struct {
	DwSize              coord
	DwCursorPosition    coord
	WAttributes         uint16
	SrWindow            smallRect
	DwMaximumWindowSize coord
}

type coord struct {
	X int16
	Y int16
}

type smallRect struct {
	Left   int16
	Top    int16
	Right  int16
	Bottom int16
}

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

// ENABLE_VIRTUAL_TERMINAL_PROCESSING 启用控制台对 ANSI 转义序列的支持。
const enableVirtualTerminalProcessing = 0x0004

// ansiEnabled 指示控制台是否已启用 ANSI 转义序列（Windows 10+）。
// 启用失败时（旧版终端），进度条回退为"回车 + 空格覆盖"的原地刷新方式。
var ansiEnabled = enableConsoleVT()

// enableConsoleVT 为标准输出启用虚拟终端处理。
func enableConsoleVT() bool {
	getStdHandle := kernel32.NewProc("GetStdHandle")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")

	const stdOutputHandle = ^uintptr(10) + 1 // STD_OUTPUT_HANDLE = -11
	handle, _, _ := getStdHandle.Call(stdOutputHandle)
	if handle == 0 {
		return false
	}
	var mode uint32
	if ret, _, _ := getConsoleMode.Call(handle, uintptr(unsafe.Pointer(&mode))); ret == 0 {
		return false
	}
	ret, _, _ := setConsoleMode.Call(handle, uintptr(mode|enableVirtualTerminalProcessing))
	return ret != 0
}

// queryTerminalWidth 通过 Windows 控制台 API 查询宽度。
func queryTerminalWidth() (int, bool) {
	getCSBI := kernel32.NewProc("GetConsoleScreenBufferInfo")
	handle := syscall.Handle(os.Stdout.Fd())
	var csbi consoleScreenBufferInfo
	ret, _, _ := getCSBI.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))
	if ret == 0 {
		return 0, false
	}
	width := int(csbi.SrWindow.Right - csbi.SrWindow.Left + 1)
	if width <= 0 {
		return 0, false
	}
	return width, true
}
