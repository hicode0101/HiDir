package view

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout 捕获标准输出内容。
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = original
	data, _ := io.ReadAll(r)
	return string(data)
}

// TestInLineErasesPreviousContent 验证进度条原地刷新：
// 每次 InLine 先擦除上一条内容（ANSI 擦除序列），而不是换行追加。
func TestInLineErasesPreviousContent(t *testing.T) {
	ansiEnabled = true
	defer func() { ansiEnabled = true }()
	DisableColor()

	cli := NewCLI(false, false, false)
	output := captureStdout(t, func() {
		cli.InLine("[##  ] 10% 1/10")
		cli.InLine("[####  ] 20% 2/10")
		cli.InLine("[######  ] 30% 3/10")
	})

	// 每次刷新都应包含"擦除整行 + 光标回到行首"的转义序列
	if got := strings.Count(output, "\x1b[1K\x1b[0G"); got != 3 {
		t.Errorf("expected 3 erase sequences, got %d in %q", got, output)
	}
	// 进度刷新不应产生换行（原地更新）
	if strings.Contains(strings.ReplaceAll(output, "\x1b[1K\x1b[0G", ""), "\n") {
		t.Errorf("progress refresh should not emit newlines: %q", output)
	}
}

// TestNewLineErasesProgressLine 验证命中结果打印时先擦除进度条行。
func TestNewLineErasesProgressLine(t *testing.T) {
	ansiEnabled = true
	defer func() { ansiEnabled = true }()
	DisableColor()

	cli := NewCLI(false, false, false)
	output := captureStdout(t, func() {
		cli.InLine("[##  ] 10% 1/10")
		cli.NewLine("[12:00:00] 200 -   10B - /admin")
	})

	// 换行输出前应擦除残留的进度条（擦除序列紧跟新内容）
	if !strings.Contains(output, "\x1b[1K\x1b[0G[12:00:00] 200") {
		t.Errorf("newline should erase progress first: %q", output)
	}
}

// TestInLineFallbackWithoutANSI 验证不支持 ANSI 时的回退擦除（回车 + 空格覆盖）。
func TestInLineFallbackWithoutANSI(t *testing.T) {
	ansiEnabled = false
	defer func() { ansiEnabled = true }()
	DisableColor()

	cli := NewCLI(false, false, false)
	output := captureStdout(t, func() {
		cli.InLine("progress-1")
		cli.InLine("progress-22")
	})

	// 第二次刷新应先回车、再用空格覆盖上一次的 10 个字符、再回到行首
	if !strings.Contains(output, "\r          \r") {
		t.Errorf("fallback erase missing: %q", output)
	}
}
