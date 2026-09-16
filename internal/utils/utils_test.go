package utils

import (
	"math"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// ---- SafeQuote ----

func TestSafeQuote(t *testing.T) {
	cases := []struct{ in, want string }{
		{"admin", "admin"},
		{"hello world", "hello%20world"},
		{"file.php", "file.php"},
		{"a/b/c", "a/b/c"},
		{"100%", "100%"}, // % 属于 ASCII 标点，不编码
		{"café", "caf%C3%A9"},
		{"q?v=1", "q?v=1"},
		{"a&b", "a&b"},
	}
	for _, c := range cases {
		if got := SafeQuote(c.in); got != c.want {
			t.Errorf("SafeQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ---- StripAndUniquify / SplitCSV ----

func TestStripAndUniquify(t *testing.T) {
	got := StripAndUniquify([]string{" a ", "a", "b", "", "  ", "b", "c"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("StripAndUniquify = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("StripAndUniquify = %v, want %v", got, want)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	if got := SplitCSV("a, b,,c"); len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("SplitCSV = %v", got)
	}
	if got := SplitCSV(""); got != nil {
		t.Errorf("SplitCSV(\"\") = %v, want nil", got)
	}
}

// ---- LstripOnce / RstripOnce ----

func TestLstripRstripOnce(t *testing.T) {
	if got := LstripOnce("//path", "/"); got != "/path" {
		t.Errorf("LstripOnce = %q", got)
	}
	if got := LstripOnce("path", "/"); got != "path" {
		t.Errorf("LstripOnce no-op = %q", got)
	}
	if got := RstripOnce("path//", "/"); got != "path/" {
		t.Errorf("RstripOnce = %q", got)
	}
}

// ---- GetReadableSize ----

func TestGetReadableSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0B"},
		{512, "512B"},
		{1023, "1023B"},
		{1024, "1KB"},
		{4096, "4KB"},
		{1536, "2KB"}, // round(1.5)=2（银行家舍入）
		{1048576, "1MB"},
		{1073741824, "1GB"},
	}
	for _, c := range cases {
		if got := GetReadableSize(c.in); got != c.want {
			t.Errorf("GetReadableSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGetReadableSizeBankersRounding(t *testing.T) {
	// Python round(1.5)=2、round(2.5)=2（银行家舍入）
	if got := GetReadableSize(1024 * 3 / 2); got != "2KB" {
		t.Errorf("1.5KB -> %q, want 2KB", got)
	}
	if got := GetReadableSize(1024 * 5 / 2); got != "2KB" {
		t.Errorf("2.5KB -> %q, want 2KB (banker's rounding)", got)
	}
}

// ---- GetResponseLength ----

func TestGetResponseLength(t *testing.T) {
	if got := GetResponseLength("123", 10); got != 123 {
		t.Errorf("GetResponseLength header = %d", got)
	}
	if got := GetResponseLength("", 42); got != 42 {
		t.Errorf("GetResponseLength fallback = %d", got)
	}
	if got := GetResponseLength("abc", 42); got != 42 {
		t.Errorf("GetResponseLength invalid = %d", got)
	}
	if got := GetResponseLength("-5", 42); got != 42 {
		t.Errorf("GetResponseLength negative = %d", got)
	}
}

// ---- IsBinary ----

func TestIsBinary(t *testing.T) {
	if IsBinary([]byte("hello world\n")) {
		t.Error("text should not be binary")
	}
	if !IsBinary([]byte{0x00, 0x01, 0x02}) {
		t.Error("NUL bytes should be binary")
	}
	if IsBinary([]byte("")) {
		t.Error("empty should not be binary")
	}
}

// ---- IPRange ----

func TestIPRange(t *testing.T) {
	ips, err := IPRange("192.168.1.0/30")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"192.168.1.0", "192.168.1.1", "192.168.1.2", "192.168.1.3"}
	if len(ips) != 4 {
		t.Fatalf("IPRange len = %d", len(ips))
	}
	for i := range want {
		if ips[i] != want[i] {
			t.Errorf("IPRange[%d] = %s, want %s", i, ips[i], want[i])
		}
	}

	if _, err := IPRange("not-a-cidr"); err == nil {
		t.Error("invalid CIDR should error")
	}
}

// ---- URL helpers ----

func TestCleanPath(t *testing.T) {
	if got := CleanPath("/path?a=1#frag", false, false); got != "/path" {
		t.Errorf("CleanPath = %q", got)
	}
	if got := CleanPath("/path?a=1#frag", true, false); got != "/path?a=1" {
		t.Errorf("CleanPath keepQueries = %q", got)
	}
	if got := CleanPath("/path?a=1#frag", true, true); got != "/path?a=1#frag" {
		t.Errorf("CleanPath keepAll = %q", got)
	}
}

func TestParsePath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"http://example.com/admin/page", "admin/page"},
		{"http://example.com/", ""},
		{"/root/path", "root/path"},
		{"relative", "relative"},
	}
	for _, c := range cases {
		if got := ParsePath(c.in); got != c.want {
			t.Errorf("ParsePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSameOrigin(t *testing.T) {
	if !SameOrigin("http://a.com/x", "http://a.com/y") {
		t.Error("same host should be same origin")
	}
	if SameOrigin("http://a.com", "https://a.com") {
		t.Error("different scheme should differ")
	}
	if SameOrigin("http://a.com:8080", "http://a.com") {
		t.Error("different port should differ")
	}
	if !SameOrigin("http://a.com", "http://a.com:80") {
		t.Error("default port should match implicit")
	}
}

func TestSameOriginPath(t *testing.T) {
	if got := SameOriginPath("http://a.com/base/", "/other"); got != "other" {
		t.Errorf("SameOriginPath = %q", got)
	}
	if got := SameOriginPath("http://a.com/", "http://b.com/x"); got != "" {
		t.Errorf("cross-origin should be empty, got %q", got)
	}
}

func TestEnsureTrailingPathSlash(t *testing.T) {
	if got := EnsureTrailingPathSlash("http://a.com/x"); got != "http://a.com/x/" {
		t.Errorf("EnsureTrailingPathSlash = %q", got)
	}
	if got := EnsureTrailingPathSlash("http://a.com/x/"); got != "http://a.com/x/" {
		t.Errorf("EnsureTrailingPathSlash unchanged = %q", got)
	}
}

func TestAppendQueryString(t *testing.T) {
	if got := AppendQueryString("path", "a=1"); got != "path?a=1" {
		t.Errorf("AppendQueryString = %q", got)
	}
	if got := AppendQueryString("path?b=2", "a=1"); got != "path?b=2" {
		t.Errorf("AppendQueryString existing = %q", got)
	}
	if got := AppendQueryString("path#frag", "a=1"); got != "path?a=1#frag" {
		t.Errorf("AppendQueryString fragment = %q", got)
	}
	if got := AppendQueryString("path", ""); got != "path" {
		t.Errorf("AppendQueryString empty = %q", got)
	}
}

func TestJoinRequestTarget(t *testing.T) {
	if got := JoinRequestTarget("http://a.com/", "admin"); got != "/admin" {
		t.Errorf("JoinRequestTarget = %q", got)
	}
	if got := JoinRequestTarget("http://a.com/base/", "admin"); got != "/base/admin" {
		t.Errorf("JoinRequestTarget subpath = %q", got)
	}
}

func TestMergePath(t *testing.T) {
	if got := MergePath("http://a.com/folder/foo", "bar"); got != "http://a.com/folder/bar" {
		t.Errorf("MergePath = %q", got)
	}
}

// ---- diff helpers ----

func TestNormalizeDynamicContent(t *testing.T) {
	a := `Request 123456 processed at 2024-01-01 12:00:00 token="abcdef-123456" tracked`
	b := `Request 654321 processed at 2023-05-05 01:02:03 token="zzzzzz-987654" tracked`
	na := NormalizeDynamicContent(a)
	nb := NormalizeDynamicContent(b)
	if na != nb {
		t.Errorf("normalized contents differ:\n%s\n%s", na, nb)
	}
	if !contains(na, "__DYNAMIC__") || !contains(na, "__DYNAMIC_ATTR__") {
		t.Errorf("expected dynamic markers in %q", na)
	}
}

func TestNormalizedContentSimilarity(t *testing.T) {
	if got := NormalizedContentSimilarity("a b c", "a b c"); math.Abs(got-1) > 1e-9 {
		t.Errorf("identical similarity = %f", got)
	}
	if got := NormalizedContentSimilarity("a b c", "x y z"); got != 0 {
		t.Errorf("disjoint similarity = %f", got)
	}
	if got := NormalizedContentSimilarity("", ""); got != 1 {
		t.Errorf("empty similarity = %f", got)
	}
}

func TestGenerateMatchingRegex(t *testing.T) {
	re := GenerateMatchingRegex("/foo/__M__", "/foo/__M__")
	if re != "^/foo/__M__$" {
		t.Errorf("identical strings regex = %q", re)
	}
	re = GenerateMatchingRegex("/aaa123", "/bbb123")
	if re != "^/.*123$" && re != "^.*123$" {
		t.Errorf("partial regex = %q", re)
	}
}

// ---- crawl helpers ----

func TestCrawlFilter(t *testing.T) {
	got := CrawlFilter([]string{"/a", "/a", "/b.jpg", "", "/c?q=1"})
	if len(got) != 2 || got[0] != "/a" || got[1] != "/c?q=1" {
		t.Errorf("CrawlFilter = %v", got)
	}
}

func TestCrawlText(t *testing.T) {
	content := `Check http://127.0.0.1:8971/secret-path and "http://127.0.0.1:8971/quoted-path" end`
	got := CrawlText("http://127.0.0.1:8971/", content)
	found := map[string]bool{}
	for _, p := range got {
		found[p] = true
	}
	// 与 dirsearch 一致：提取的路径不含开头斜杠
	if !found["secret-path"] || !found["quoted-path"] {
		t.Errorf("CrawlText = %v", got)
	}
}

func TestCrawlRobots(t *testing.T) {
	got := CrawlRobots("Disallow: /admin\nAllow: /public\nUser-agent: *")
	if len(got) != 2 {
		t.Fatalf("CrawlRobots = %v", got)
	}
	if got[0] != "admin" || got[1] != "public" {
		t.Errorf("CrawlRobots paths = %v", got)
	}
}

func TestCrawlHTML(t *testing.T) {
	html := `<html><base href="http://a.com/dir/"><a href="page1">x</a>
	<img src="/images/logo.png"><a href="http://a.com/page2">y</a>
	<a href="https://b.com/evil">z</a><script src="app.js"></script></html>`
	got := CrawlHTML("http://a.com/dir/index", "http://a.com/", html)
	found := map[string]bool{}
	for _, p := range got {
		found[p] = true
	}
	// base href 指向 /dir/，相对路径基于其解析（不含开头斜杠）
	if !found["dir/page1"] || !found["page2"] {
		t.Errorf("CrawlHTML = %v", got)
	}
	if found["images/logo.png"] || found["/images/logo.png"] {
		t.Error("media extension should be filtered")
	}
	if found["evil"] || found["/evil"] {
		t.Error("cross-origin link should be filtered")
	}
}

func TestSrcsetURLs(t *testing.T) {
	got := SrcsetURLs("/a.png 1x, /b.png 2x, /c.png 100w")
	if len(got) != 3 || got[0] != "/a.png" || got[2] != "/c.png" {
		t.Errorf("SrcsetURLs = %v", got)
	}
}

func TestBrowserURLValue(t *testing.T) {
	if got := BrowserURLValue(`/a\b?q=1`); got != "/a/b?q=1" {
		t.Errorf("BrowserURLValue = %q", got)
	}
}

// ---- headers ----

func TestParseHeaders(t *testing.T) {
	headers, err := ParseHeaders("Host: example.com\nX-Custom: a\n\tb\nAccept: json")
	if err != nil {
		t.Fatal(err)
	}
	if headers["host"] != "example.com" {
		t.Errorf("host = %q", headers["host"])
	}
	if headers["x-custom"] != "a b" {
		t.Errorf("folded header = %q", headers["x-custom"])
	}
	if headers["accept"] != "json" {
		t.Errorf("accept = %q", headers["accept"])
	}

	// 非法行（缺少冒号）应报错
	if _, err := ParseHeaders("Bad-Line\nAccept: json"); err == nil {
		t.Error("invalid line should error")
	}
}

// ---- mimetype ----

func TestGuessMimetype(t *testing.T) {
	if got := GuessMimetype([]byte(`{"a": 1}`)); got != "application/json" {
		t.Errorf("json = %q", got)
	}
	if got := GuessMimetype([]byte("<root><a/></root>")); got != "application/xml" {
		t.Errorf("xml = %q", got)
	}
	if got := GuessMimetype([]byte("a=1&b=2")); got != "application/x-www-form-urlencoded" {
		t.Errorf("query = %q", got)
	}
	if got := GuessMimetype([]byte("plain text")); got != "text/plain" {
		t.Errorf("plain = %q", got)
	}
}

// ---- raw request ----

func TestParseRawContentOriginForm(t *testing.T) {
	raw := "GET /admin/dashboard HTTP/1.1\r\nHost: example.com\r\nUser-Agent: test\r\n\r\nbody-here"
	req, err := ParseRawContent([]byte(raw), "")
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "GET" {
		t.Errorf("method = %q", req.Method)
	}
	if req.URL != "example.com/admin/dashboard" {
		t.Errorf("url = %q", req.URL)
	}
	if req.Headers["host"] != "example.com" {
		t.Errorf("headers = %v", req.Headers)
	}
	if string(req.Body) != "body-here" {
		t.Errorf("body = %q", req.Body)
	}
}

func TestParseRawContentAbsoluteForm(t *testing.T) {
	raw := "GET http://example.com/path?a=1 HTTP/1.1\r\nHost: ignored\r\n\r\n"
	req, err := ParseRawContent([]byte(raw), "")
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "http://example.com/path?a=1" {
		t.Errorf("url = %q", req.URL)
	}
}

func TestParseRawContentWithScheme(t *testing.T) {
	raw := "GET /x HTTP/1.1\r\nHost: example.com\r\n\r\n"
	req, err := ParseRawContent([]byte(raw), "https")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(req.URL, "https://") {
		t.Errorf("scheme url = %q", req.URL)
	}
}

func TestParseRawContentMissingHost(t *testing.T) {
	raw := "GET /x HTTP/1.1\r\n\r\n"
	if _, err := ParseRawContent([]byte(raw), ""); err == nil {
		t.Error("missing Host should error")
	}
}

// ---- nmap ----

func TestParseNmapReport(t *testing.T) {
	xmlData := `<?xml version="1.0"?>
<nmaprun>
 <host>
  <hostnames><hostname name="web1.example.com"/></hostnames>
  <address addr="10.0.0.1"/>
  <ports>
   <port protocol="tcp" portid="80"><state state="open"/><service name="http"/></port>
   <port protocol="tcp" portid="22"><state state="open"/><service name="ssh"/></port>
   <port protocol="tcp" portid="443"><state state="closed"/><service name="http"/></port>
   <port protocol="udp" portid="8080"><state state="open"/><service name="http"/></port>
   <port protocol="tcp" portid="8181"><state state="open"/><service name="unknown"/></port>
  </ports>
 </host>
</nmaprun>`
	path := t.TempDir() + "/nmap.xml"
	if err := writeFile(path, xmlData); err != nil {
		t.Fatal(err)
	}
	targets, err := ParseNmapReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("targets = %v", targets)
	}
	if targets[0] != "web1.example.com:80" || targets[1] != "web1.example.com:8181" {
		t.Errorf("targets = %v", targets)
	}
}

// ---- command redaction ----

func TestRedactCommand(t *testing.T) {
	got := RedactCommand([]string{"hidir", "-u", "http://user:pass@a.com/", "-e", "php"})
	if !contains(got, "<redacted>") {
		t.Errorf("URL with credentials should be redacted: %s", got)
	}
	if !contains(got, "-e php") {
		t.Errorf("plain options should remain: %s", got)
	}

	got = RedactCommand([]string{"--auth", "user:pass", "-t", "10"})
	if !contains(got, "--auth <redacted>") {
		t.Errorf("auth should be redacted: %s", got)
	}
	if !contains(got, "-t 10") {
		t.Errorf("numeric option should remain: %s", got)
	}

	got = RedactCommand([]string{"--header=X-Api: secret"})
	if !contains(got, "--header=<redacted>") {
		t.Errorf("inline header should be redacted: %s", got)
	}
}

// ---- resource FS ----

func TestResourceFSEmbedded(t *testing.T) {
	rfs := &ResourceFS{}
	if !rfs.Exists("dicc.txt") {
		t.Fatal("embedded dicc.txt should exist")
	}
	lines, err := rfs.GetLines("dicc.txt")
	if err != nil || len(lines) == 0 {
		t.Fatalf("embedded dicc.txt lines = %d, err = %v", len(lines), err)
	}
	if !rfs.Exists("400_blacklist.txt") {
		t.Error("embedded blacklist should exist")
	}
	if !rfs.IsDir("categories") {
		t.Error("categories should be a directory")
	}
	files, err := rfs.WalkFiles("categories")
	if err != nil || len(files) == 0 {
		t.Errorf("WalkFiles categories = %d files, err = %v", len(files), err)
	}
}

func TestResourceFSOnDisk(t *testing.T) {
	dir := t.TempDir()
	if err := writeFile(dir+"/custom.txt", "hello\nworld\n"); err != nil {
		t.Fatal(err)
	}
	rfs := &ResourceFS{Dir: dir}
	if !rfs.Exists("custom.txt") {
		t.Fatal("on-disk file should exist")
	}
	lines, _ := rfs.GetLines("custom.txt")
	if len(lines) != 2 || lines[0] != "hello" {
		t.Errorf("GetLines = %v", lines)
	}
	abs := dir + string(os.PathSeparator) + "custom.txt"
	if !rfs.Exists(abs) && !rfs.Exists(mustAbs(dir)+"/custom.txt") {
		t.Error("absolute path should resolve to disk")
	}
}

// ---- helpers ----

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func TestURLParsingDefaults(t *testing.T) {
	u, _ := url.Parse("http://a.com:8080/x")
	if u.Port() != "8080" {
		t.Errorf("port = %q", u.Port())
	}
}
