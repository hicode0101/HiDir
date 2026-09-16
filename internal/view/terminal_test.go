package view

import (
	"strings"
	"testing"

	"hidir/internal/httpx"
)

// ---- 颜色 ----

func TestSetColorAndClean(t *testing.T) {
	DisableColor()
	if got := SetColor("msg", "red", "", "bright"); got != "msg" {
		t.Errorf("disabled color output = %q", got)
	}
	colorEnabled = true
	colored := SetColor("msg", "red", "", "bright")
	if !strings.Contains(colored, "msg") || !strings.Contains(colored, "\x1b[") {
		t.Errorf("colored output = %q", colored)
	}
	if cleaned := CleanColor(colored); cleaned != "msg" {
		t.Errorf("CleanColor = %q", cleaned)
	}
	colorEnabled = true
}

// ---- 文本展示 ----

func TestSafeDisplayText(t *testing.T) {
	if got := SafeDisplayText("hello\x00world"); got != "helloworld" {
		t.Errorf("control chars should be removed: %q", got)
	}
	long := strings.Repeat("a", 300)
	got := SafeDisplayText(long)
	if len(got) != maxDisplayTextLength || !strings.HasSuffix(got, "...") {
		t.Errorf("long text should be truncated: len=%d", len(got))
	}
}

// ---- CLI 输出 ----

func TestCLIStatusReport(t *testing.T) {
	DisableColor()
	cli := NewCLI(false, false, false)
	resp := httpx.NewResponse("http://a.com/admin", 200,
		map[string]string{"content-type": "text/html"}, []byte("x"))
	resp.Datetime = "2024-06-15 12:00:00"
	cli.StatusReport(resp, false, false)

	buffer := cli.Buffer()
	if !strings.Contains(buffer, "[12:00:00] 200 -") {
		t.Errorf("status report = %q", buffer)
	}
	if !strings.Contains(buffer, "/admin") {
		t.Errorf("path missing: %q", buffer)
	}

	// full URL 模式
	cli2 := NewCLI(false, false, false)
	cli2.StatusReport(resp, true, false)
	if !strings.Contains(cli2.Buffer(), "http://a.com/admin") {
		t.Errorf("full url report = %q", cli2.Buffer())
	}

	// verbose 模式
	cli3 := NewCLI(false, false, false)
	cli3.StatusReport(resp, false, true)
	if !strings.Contains(cli3.Buffer(), "ms, text/html") {
		t.Errorf("verbose report = %q", cli3.Buffer())
	}
}

func TestCLIRedirectHistory(t *testing.T) {
	DisableColor()
	cli := NewCLI(false, false, false)
	resp := httpx.NewResponse("http://a.com/new", 301,
		map[string]string{"location": "/final"}, []byte(""))
	resp.History = []string{"http://a.com/old"}
	cli.StatusReport(resp, false, false)

	buffer := cli.Buffer()
	if !strings.Contains(buffer, "->  /final") {
		t.Errorf("redirect missing: %q", buffer)
	}
	if !strings.Contains(buffer, "-->  http://a.com/old") {
		t.Errorf("redirect history missing: %q", buffer)
	}
}

func TestCLIQuietAndDisable(t *testing.T) {
	quiet := NewCLI(false, true, false)
	quiet.Warning("warn", true)
	quiet.Header("header")
	quiet.Config(nil, nil, nil, "GET", 1, 1)
	quiet.Target("http://a.com/")
	if quiet.Buffer() != "" {
		t.Errorf("quiet mode should suppress these outputs: %q", quiet.Buffer())
	}

	disabled := NewCLI(false, false, true)
	disabled.NewLine("hidden")
	disabled.Error("hidden")
	if disabled.Buffer() != "" {
		t.Errorf("disable-cli should suppress all outputs: %q", disabled.Buffer())
	}
}

func TestCLIErrorWarningProgress(t *testing.T) {
	DisableColor()
	cli := NewCLI(false, false, false)
	cli.Error("something failed")
	if !strings.Contains(cli.Buffer(), "something failed") {
		t.Errorf("error output = %q", cli.Buffer())
	}

	cli.Warning("careful", true)
	if !strings.Contains(cli.Buffer(), "careful") {
		t.Errorf("warning output = %q", cli.Buffer())
	}

	// 进度条是瞬态输出（写入 stdout 但不进入会话缓冲，与 dirsearch 一致）
	before := cli.Buffer()
	cli.LastPath(50, 100, 1, 2, 10, 3)
	if cli.Buffer() != before {
		t.Errorf("progress should be transient, buffer = %q", cli.Buffer())
	}

	cli.NewDirectories([]string{"admin/", "api/"})
	if !strings.Contains(cli.Buffer(), "Added to the queue") {
		t.Errorf("new directories output = %q", cli.Buffer())
	}
}
