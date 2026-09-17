package report

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// newTestManager 构造测试用的报告管理器。
func newTestManager(formats []string, files map[string]string) *Manager {
	return newTestManagerWithTable(formats, files, "")
}

// newTestManagerWithTable 构造带 SQL 表名模板的报告管理器。
func newTestManagerWithTable(formats []string, files map[string]string, table string) *Manager {
	meta := StartInfo{StartTime: "2024-06-15 12:00:00", Command: "hidir -u http://a.com/"}
	return NewManager(formats, meta, files, table)
}

func testResult() Result {
	return Result{
		Datetime: "2024-06-15 12:00:01",
		URL:      "http://a.com/admin",
		Status:   200,
		Length:   1234,
		Type:     "text/html",
		Redirect: "",
		Elapsed:  0.123,
	}
}

func tmpFile(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name)
}

func readAll(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// ---- simple ----

func TestSimpleReport(t *testing.T) {
	file := tmpFile(t, "out.txt")
	m := newTestManager([]string{"simple"}, map[string]string{"simple": file})
	if err := m.Prepare("http://a.com/"); err != nil {
		t.Fatal(err)
	}
	m.Save(testResult())
	m.Save(testResult())

	content := readAll(t, file)
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) != 2 || lines[0] != "http://a.com/admin" {
		t.Errorf("simple report = %q", content)
	}
}

// ---- plain ----

func TestPlainReport(t *testing.T) {
	file := tmpFile(t, "out.txt")
	m := newTestManager([]string{"plain"}, map[string]string{"plain": file})
	if err := m.Prepare("http://a.com/"); err != nil {
		t.Fatal(err)
	}
	m.Save(testResult())

	content := readAll(t, file)
	if !strings.HasPrefix(content, "# HiDir started at 2024-06-15 12:00:00 as: hidir -u http://a.com/\n") {
		t.Errorf("plain header = %q", content)
	}
	if !strings.Contains(content, "200    1KB http://a.com/admin  (123ms)") {
		t.Errorf("plain row = %q", content)
	}
}

func TestPlainReportWithRedirect(t *testing.T) {
	file := tmpFile(t, "out.txt")
	m := newTestManager([]string{"plain"}, map[string]string{"plain": file})
	_ = m.Prepare("http://a.com/")
	result := testResult()
	result.Redirect = "/login/"
	result.Elapsed = 0
	m.Save(result)

	content := readAll(t, file)
	if !strings.Contains(content, "200    1KB http://a.com/admin  ->  /login/") {
		t.Errorf("plain row with redirect = %q", content)
	}
}

// ---- json ----

func TestJSONReport(t *testing.T) {
	file := tmpFile(t, "out.json")
	m := newTestManager([]string{"json"}, map[string]string{"json": file})
	if err := m.Prepare("http://a.com/"); err != nil {
		t.Fatal(err)
	}
	m.Save(testResult())

	var data struct {
		Info struct {
			Args string `json:"args"`
			Time string `json:"time"`
		} `json:"info"`
		Results []struct {
			URL           string  `json:"url"`
			Status        int     `json:"status"`
			ContentLength int64   `json:"contentLength"`
			ContentType   string  `json:"contentType"`
			Redirect      string  `json:"redirect"`
			Elapsed       float64 `json:"elapsed"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(readAll(t, file)), &data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if data.Info.Time != "2024-06-15 12:00:00" {
		t.Errorf("info.time = %q", data.Info.Time)
	}
	if len(data.Results) != 1 || data.Results[0].URL != "http://a.com/admin" ||
		data.Results[0].Status != 200 || data.Results[0].ContentLength != 1234 ||
		data.Results[0].ContentType != "text/html" {
		t.Errorf("results = %+v", data.Results)
	}
}

// ---- xml ----

func TestXMLReport(t *testing.T) {
	file := tmpFile(t, "out.xml")
	m := newTestManager([]string{"xml"}, map[string]string{"xml": file})
	if err := m.Prepare("http://a.com/"); err != nil {
		t.Fatal(err)
	}
	m.Save(testResult())

	content := readAll(t, file)
	if !strings.Contains(content, `<hidirscan args="hidir -u http://a.com/" time="2024-06-15 12:00:00">`) {
		t.Errorf("xml root = %q", content)
	}
	if !strings.Contains(content, "<status>200</status>") ||
		!strings.Contains(content, "<contentLength>1234</contentLength>") {
		t.Errorf("xml result = %q", content)
	}
}

// ---- markdown ----

func TestMarkdownReport(t *testing.T) {
	file := tmpFile(t, "out.md")
	m := newTestManager([]string{"md"}, map[string]string{"md": file})
	if err := m.Prepare("http://a.com/"); err != nil {
		t.Fatal(err)
	}
	m.Save(testResult())

	content := readAll(t, file)
	if !strings.Contains(content, "URL | Status | Size | Content Type | Redirection | Elapsed (ms)") {
		t.Errorf("md header = %q", content)
	}
	if !strings.Contains(content, "http://a.com/admin | 200 | 1234 | text/html |  | 123") {
		t.Errorf("md row = %q", content)
	}
}

// ---- csv ----

func TestCSVReport(t *testing.T) {
	file := tmpFile(t, "out.csv")
	m := newTestManager([]string{"csv"}, map[string]string{"csv": file})
	if err := m.Prepare("http://a.com/"); err != nil {
		t.Fatal(err)
	}
	m.Save(testResult())

	content := readAll(t, file)
	if !strings.HasPrefix(content, "URL,Status,Size,Content Type,Redirection,Elapsed (s)\n") {
		t.Errorf("csv header = %q", content)
	}
	if !strings.Contains(content, `http://a.com/admin,200,1234,text/html,,0.123`) {
		t.Errorf("csv row = %q", content)
	}
}

// ---- html ----

func TestHTMLReport(t *testing.T) {
	file := tmpFile(t, "out.html")
	m := newTestManager([]string{"html"}, map[string]string{"html": file})
	if err := m.Prepare("http://a.com/"); err != nil {
		t.Fatal(err)
	}
	m.Save(testResult())

	content := readAll(t, file)
	if !strings.Contains(content, "<title>HiDir report</title>") {
		t.Errorf("html title missing")
	}
	if !strings.Contains(content, `resources: [{"url":"http://a.com/admin","status":200`) {
		t.Errorf("html resources line missing: %q", content[:min(len(content), 400)])
	}
	for _, marker := range []string{
		"cdn.jsdelivr.net/npm/vue@2.6.12/dist/vue.js",
		"cdn.jsdelivr.net/npm/bootstrap@5.0.0/dist/css/bootstrap.min.css",
		">HiDir</a>",
		"Exclude status codes, separated by space",
		"Exclude content lengths, separated by space",
		"Open all URLs",
		"sortColumn",
		"getReadableSize",
		"getStatusColorClass",
	} {
		if !strings.Contains(content, marker) {
			t.Errorf("html template marker missing: %s", marker)
		}
	}
}

// ---- 输出文件冲突 ----

func TestOutputFileConflict(t *testing.T) {
	file := tmpFile(t, "out.txt")
	if err := os.WriteFile(file, []byte("not a valid report header"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newTestManager([]string{"json"}, map[string]string{"json": file})
	err := m.Prepare("http://a.com/")
	if err == nil {
		t.Fatal("conflicting file should error")
	}
	if _, ok := err.(*FileExistsError); !ok {
		t.Fatalf("error type = %T", err)
	}
	if err.Error() != "Output file "+file+" already exists" {
		t.Errorf("message = %q", err.Error())
	}
}

// ---- 路径变量 ----

func TestFormatPath(t *testing.T) {
	m := newTestManager(nil, nil)
	got := m.FormatPath("report-{format}-{extension}-{date}.txt", "http://a.com:8080/x", "json")
	want := "report-json-json-2024-06-15.txt"
	if got != want {
		t.Errorf("FormatPath = %q, want %q", got, want)
	}

	got = m.FormatPath("{scheme}-{host}-{port}", "https://b.com/x", "plain")
	if got != "https-b.com-443" {
		t.Errorf("FormatPath host vars = %q", got)
	}
}

// ---- 数据库格式警告 ----

func TestUnsupportedFormatsWarning(t *testing.T) {
	var warnings []string
	m := newTestManager([]string{"sqlite", "mysql", "postgresql"}, nil)
	m.Warning = func(message string) { warnings = append(warnings, message) }
	if err := m.Prepare("http://a.com/"); err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(m.reports) != 0 {
		t.Error("unsupported formats should not create reports")
	}
}

// ---- FormatName / FileExtension ----

func TestFormatNames(t *testing.T) {
	if FormatName("md") != "markdown" {
		t.Error("md format name")
	}
	if FormatName("sqlite") != "sql" {
		t.Error("sqlite format name")
	}
	if FileExtension("simple") != "txt" || FileExtension("json") != "json" {
		t.Error("file extensions")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---- sqlite ----

func TestSQLiteReport(t *testing.T) {
	file := tmpFile(t, "out.sqlite")
	m := newTestManagerWithTable([]string{"sqlite"},
		map[string]string{"sqlite": file}, "https_a.com:443")
	if err := m.Prepare("http://a.com/"); err != nil {
		t.Fatal(err)
	}
	m.Save(testResult())
	second := testResult()
	second.URL = "http://a.com/api"
	second.Status = 301
	second.Redirect = "/admin/"
	m.Save(second)

	conn, err := sql.Open("sqlite", file)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	rows, err := conn.Query(`SELECT CAST(time AS TEXT), url, status_code, content_length, content_type, redirect
		FROM "https_a.com:443" ORDER BY url`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	var got [][6]string
	for rows.Next() {
		var time, url, redirect string
		var status, length int
		var contentType string
		if err := rows.Scan(&time, &url, &status, &length, &contentType, &redirect); err != nil {
			t.Fatal(err)
		}
		got = append(got, [6]string{time, url, fmt.Sprint(status), fmt.Sprint(length), contentType, redirect})
	}
	if len(got) != 2 {
		t.Fatalf("rows = %d, want 2", len(got))
	}
	if got[0][1] != "http://a.com/admin" || got[0][2] != "200" || got[0][3] != "1234" ||
		got[0][4] != "text/html" || got[0][5] != "" {
		t.Errorf("row0 = %v", got[0])
	}
	if got[1][1] != "http://a.com/api" || got[1][2] != "301" || got[1][5] != "/admin/" {
		t.Errorf("row1 = %v", got[1])
	}
	if got[0][0] != "2024-06-15 12:00:01" {
		t.Errorf("time = %q", got[0][0])
	}
}

func TestSQLiteReportInvalidFile(t *testing.T) {
	file := tmpFile(t, "out.sqlite")
	if err := os.WriteFile(file, []byte("definitely not a sqlite database"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newTestManagerWithTable([]string{"sqlite"},
		map[string]string{"sqlite": file}, "result")
	err := m.Prepare("http://a.com/")
	if err == nil {
		t.Fatal("invalid sqlite file should error")
	}
	if !strings.Contains(err.Error(), "is not empty or is not a valid SQLite database") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestSQLiteReportSkippedWithoutTable(t *testing.T) {
	file := tmpFile(t, "out.sqlite")
	m := newTestManager([]string{"sqlite"}, map[string]string{"sqlite": file})
	if err := m.Prepare("http://a.com/"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("sqlite report without table should be skipped")
	}
}
