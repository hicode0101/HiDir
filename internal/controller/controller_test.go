package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hidir/internal/options"
	"hidir/internal/scanner"
	"hidir/internal/utils"
	"hidir/internal/view"
)

// scanServer 定义测试服务器的路由响应。
type scanServer struct {
	handler http.HandlerFunc
}

// newScanServer 构建一个模拟"软 404"站点的测试服务器。
// 默认行为：未知路径返回 404 + 固定页面（软 404），
// /admin/、/api/、/login.php 返回 200。
func newScanServer() *httptest.Server {
	soft404 := "<html><body>Sorry, page not found</body></html>"
	existing := map[string]string{
		"/admin/":     "<html><body>Admin Panel</body></html>",
		"/api/":       "<html><body>API v1</body></html>",
		"/login.php":  "<html><body>Login</body></html>",
		"/index.html": "<html><body>Home</body></html>",
	}
	dirs := map[string]bool{"/admin": true, "/api": true, "/secret": true}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if dirs[path] {
			// 目录重定向
			http.Redirect(w, r, path+"/", 301)
			return
		}
		if path == "/secret/" {
			// 403 受保护目录
			w.WriteHeader(403)
			fmt.Fprint(w, "forbidden area")
			return
		}
		if body, ok := existing[path]; ok {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, body)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(404)
		fmt.Fprint(w, soft404)
	}))
}

// testOptions 构造集成测试的选项（基于真实参数解析管线）。
func integrationOptions(t *testing.T, args []string) *options.Options {
	t.Helper()
	opts, err := options.ParseOptions(args, options.OSFilesystem(), utils.NewResourceFS())
	if err != nil {
		t.Fatalf("ParseOptions: %v", err)
	}
	return opts
}

// runScan 执行一次完整扫描并返回 UI 输出。
func runScan(t *testing.T, opts *options.Options) string {
	t.Helper()
	rfs := utils.NewResourceFS()
	ui := view.NewCLI(false, false, false)
	ctrl := NewController(opts, rfs, ui)
	if err := ctrl.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return ui.Buffer()
}

// ---- 基础扫描 ----

func TestScanBasicDiscovery(t *testing.T) {
	server := newScanServer()
	defer server.Close()

	// 词表：admin 命中（301→200 目录）、login.php 未加扩展、random 软 404 被过滤
	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("admin\nlogin.php\nrandomword1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := integrationOptions(t, []string{
		"-u", server.URL + "/",
		"-w", wordlist,
		"-t", "3",
		"--no-color",
		"--request-backend", "python",
	})
	output := runScan(t, opts)

	if !strings.Contains(output, "/admin") {
		t.Errorf("admin should be discovered, output:\n%s", output)
	}
	if strings.Contains(output, "randomword") {
		t.Errorf("soft 404 should be filtered, output:\n%s", output)
	}
}

// ---- %EXT% 扩展 ----

func TestScanExtensions(t *testing.T) {
	server := newScanServer()
	defer server.Close()

	wordlist := t.TempDir() + "/words.txt"
	// %EXT% 展开为 php → /login.php 命中
	if err := os.WriteFile(wordlist, []byte("login.%EXT%\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-e", "php", "-t", "2", "--no-color",
	})
	output := runScan(t, opts)
	if !strings.Contains(output, "/login.php") {
		t.Errorf("login.php should be found:\n%s", output)
	}
}

// ---- 递归扫描 ----

func TestScanRecursive(t *testing.T) {
	server := newScanServer()
	defer server.Close()

	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("admin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-t", "2", "--no-color", "-r",
	})
	output := runScan(t, opts)
	if !strings.Contains(output, "Added to the queue") || !strings.Contains(output, "/admin/") {
		t.Errorf("recursion should queue /admin/:\n%s", output)
	}
}

// ---- include/exclude 状态码 ----

func TestScanIncludeExcludeStatus(t *testing.T) {
	server := newScanServer()
	defer server.Close()

	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("admin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 仅包含 403：/secret/ 返回 403
	secretWordlist := t.TempDir() + "/secret.txt"
	if err := os.WriteFile(secretWordlist, []byte("secret/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", secretWordlist, "-t", "2", "--no-color",
		"-i", "200-299",
	})
	output := runScan(t, opts)
	if strings.Contains(output, "/secret") {
		t.Errorf("403 should be excluded by -i 200-299:\n%s", output)
	}

	opts = integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", secretWordlist, "-t", "2", "--no-color",
		"-i", "403",
	})
	output = runScan(t, opts)
	if !strings.Contains(output, "/secret") {
		t.Errorf("403 should be included with -i 403:\n%s", output)
	}
}

// ---- skip-on-status ----

func TestScanSkipOnStatus(t *testing.T) {
	server := newScanServer()
	defer server.Close()

	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("secret/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-t", "2", "--no-color",
		"--skip-on-status", "403",
	})
	output := runScan(t, opts)
	if !strings.Contains(output, "Skipped the target due to 403 status code") {
		t.Errorf("skip-on-status message missing:\n%s", output)
	}
}

// ---- 文本排除 ----

func TestScanExcludeText(t *testing.T) {
	server := newScanServer()
	defer server.Close()

	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("login.php\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-t", "2", "--no-color",
		"--exclude-text", "Login",
	})
	output := runScan(t, opts)
	if strings.Contains(output, "/admin") {
		t.Errorf("response with excluded text should be filtered:\n%s", output)
	}
}

// ---- 高级匹配器 ----

func TestScanAdvancedMatchers(t *testing.T) {
	server := newScanServer()
	defer server.Close()

	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("login.php\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// match-status 200：命中；换 404 则全部过滤
	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-t", "2", "--no-color",
		"--match-status", "200",
	})
	output := runScan(t, opts)
	if !strings.Contains(output, "/login.php") {
		t.Errorf("match-status 200 should keep /login.php:\n%s", output)
	}

	opts = integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-t", "2", "--no-color",
		"--match-status", "404",
	})
	output = runScan(t, opts)
	if strings.Contains(output, "/login.php") {
		t.Errorf("match-status 404 should drop /login.php:\n%s", output)
	}
}

// ---- 输出报告 ----

func TestScanReports(t *testing.T) {
	server := newScanServer()
	defer server.Close()

	outDir := t.TempDir()
	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("admin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	formats := "simple,plain,json,xml,md,csv,html"
	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-t", "2", "--no-color",
		"-O", formats, "-o", outDir + "/out.{format}",
	})
	runScan(t, opts)

	// simple
	simpleData, err := os.ReadFile(outDir + "/out.simple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(simpleData), server.URL+"/admin\n") {
		t.Errorf("simple report = %q", simpleData)
	}

	// json
	jsonData, err := os.ReadFile(outDir + "/out.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonData), `"status": 301`) && !strings.Contains(string(jsonData), `"status": 200`) {
		t.Errorf("json report = %q", jsonData)
	}

	// 其余格式文件应存在且非空
	// {format} 变量对 md 展开为 markdown（与 dirsearch 的 __format__ 一致）
	for _, format := range []string{"plain", "xml", "markdown", "csv", "html"} {
		info, err := os.Stat(outDir + "/out." + format)
		if err != nil {
			t.Errorf("%s report missing: %v", format, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s report is empty", format)
		}
	}
}

// ---- 多目标 ----

func TestScanMultipleTargets(t *testing.T) {
	server1 := newScanServer()
	defer server1.Close()
	server2 := newScanServer()
	defer server2.Close()

	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("admin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 相同 URL 会被去重（与 dirsearch 一致），这里使用两个不同目标
	opts := integrationOptions(t, []string{
		"-u", server1.URL + "/", "-u", server2.URL + "/", "-w", wordlist, "-t", "2", "--no-color",
	})
	output := runScan(t, opts)
	if !strings.Contains(output, "Target: "+server1.URL) ||
		!strings.Contains(output, "Target: "+server2.URL) {
		t.Errorf("both targets should be scanned:\n%s", output)
	}
}

// ---- 子目录扫描 ----

func TestScanSubdirs(t *testing.T) {
	server := newScanServer()
	defer server.Close()

	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("admin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-t", "2", "--no-color",
		"--subdirs", "api/",
	})
	output := runScan(t, opts)
	if !strings.Contains(output, "Scanning: api/") {
		t.Errorf("subdir job missing:\n%s", output)
	}
}

// ---- 爬虫模式 ----

func TestScanCrawl(t *testing.T) {
	// 在 admin 页面中加入同源链接
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><a href="/admin/panel">panel</a></body></html>`)
			return
		}
		if r.URL.Path == "/admin/panel" {
			fmt.Fprint(w, "panel page")
			return
		}
		w.WriteHeader(404)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("admin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-t", "2", "--no-color", "--crawl",
	})
	output := runScan(t, opts)
	if !strings.Contains(output, "panel") {
		t.Errorf("crawled path should be scanned:\n%s", output)
	}
}

// ---- 备份探测 ----

func TestScanFindBackup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login.php", "/login.php.bak":
			fmt.Fprint(w, "content")
			return
		}
		w.WriteHeader(404)
		fmt.Fprint(w, "not found")
	}))
	defer server.Close()

	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("login.php\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-t", "2", "--no-color", "--find-backup",
	})
	output := runScan(t, opts)
	if !strings.Contains(output, "login.php.bak") {
		t.Errorf("backup path should be scanned:\n%s", output)
	}
}

// ---- 会话保存与恢复 ----

func TestSessionSaveAndList(t *testing.T) {
	sessionDir := t.TempDir()
	payload := `{
  "version": 1,
  "args": ["-u", "http://example.com/", "-e", "php"],
  "start_time": "2024-06-15 12:00:00",
  "url": "http://example.com/",
  "controller": {
    "jobs_processed": 3,
    "errors": 1,
    "consecutive_errors": 0,
    "directories_left": [""],
    "urls_left": ["http://example.com/"]
  },
  "output_history": [{"start_time": "2024-06-15 12:00:00", "output": "previous run"}]
}`
	sessionFile := filepath.Join(sessionDir, "2024-06-15", "session_2024-06-15_12-00-00.json")
	if err := os.MkdirAll(filepath.Dir(sessionFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionFile, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}

	// 列表
	var output []string
	count := PrintSessionList(sessionDir, func(message string) { output = append(output, message) })
	if count != 1 {
		t.Fatalf("sessions found = %d, output = %v", count, output)
	}
	if !strings.Contains(output[1], "targets left: 1") {
		t.Errorf("session list = %v", output)
	}

	// 按编号解析
	resolved, err := ResolveSessionID(sessionDir, "1")
	if err != nil || resolved != sessionFile {
		t.Errorf("resolve session id = %q, %v", resolved, err)
	}
	if _, err := ResolveSessionID(sessionDir, "99"); err == nil {
		t.Error("out of range id should error")
	}
	if _, err := ResolveSessionID(sessionDir, "abc"); err == nil {
		t.Error("non-numeric id should error")
	}
}

// ---- view 输出 ----

func TestViewStatusReportFormatting(t *testing.T) {
	server := newScanServer()
	defer server.Close()

	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("admin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// verbose 模式应显示耗时与 Content-Type
	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-t", "2", "--no-color", "-v",
	})
	output := runScan(t, opts)
	if !strings.Contains(output, "ms, text/html") {
		t.Errorf("verbose output should contain elapsed and type:\n%s", output)
	}
}

// ---- 无法连接的目标 ----

func TestScanUnreachableTarget(t *testing.T) {
	opts := integrationOptions(t, []string{
		"-u", "http://127.0.0.1:1/", "-t", "2", "--no-color",
	})
	output := runScan(t, opts)
	if !strings.Contains(output, "Cannot connect to") && !strings.Contains(output, "timeout") {
		t.Errorf("connect error should be reported:\n%s", output)
	}
}

// ---- Ctrl+C 暂停菜单 ----

func TestHandlePauseMenu(t *testing.T) {
	server := newScanServer()
	defer server.Close()

	wordlist := t.TempDir() + "/words.txt"
	if err := os.WriteFile(wordlist, []byte("admin\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := integrationOptions(t, []string{
		"-u", server.URL + "/", "-w", wordlist, "-t", "2", "--no-color",
	})

	rfs := utils.NewResourceFS()
	ui := view.NewCLI(false, false, false)
	ctrl := NewController(opts, rfs, ui)
	if err := ctrl.Setup(); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := ctrl.SetTarget(opts.URLs[0]); err != nil {
		t.Fatalf("SetTarget: %v", err)
	}
	ctrl.fuzzer = scanner.NewFuzzer(ctrl.requester, ctrl.dictionary, opts, ctrl.blacklists,
		scanner.Callbacks{}, 0)

	// [c]ontinue：菜单等待输入期间必须冻结进度刷新，返回后解除冻结
	inputs := []string{"c"}
	pos := 0
	ctrl.QuietInput = func() string {
		if !ctrl.progressHold.Load() {
			t.Error("progressHold must be set while the pause menu waits for input")
		}
		v := inputs[pos]
		pos++
		return v
	}
	ctrl.HandlePause()
	if ctrl.progressHold.Load() {
		t.Error("progressHold must be cleared after resuming")
	}

	// 无效输入后 [q]uit：菜单循环应重新提示并记录退出中断
	inputs = []string{"x", "q", "q"}
	pos = 0
	ctrl.HandlePause()
	if ctrl.currentInterrupt == nil || ctrl.currentInterrupt.Type != InterruptQuit {
		t.Errorf("expected quit interrupt, got %+v", ctrl.currentInterrupt)
	}
	if ctrl.progressHold.Load() {
		t.Error("progressHold must be cleared after quitting")
	}
}
