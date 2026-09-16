// Package report 实现扫描结果报告的生成，
// 支持 simple/plain/json/xml/md/csv/html 七种文件格式，
// 对应 dirsearch 的 lib/report 目录。
package report

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"hidir/internal/settings"
	"hidir/internal/utils"
)

// Result 表示一条扫描结果。
type Result struct {
	// Datetime 是发现时间（YYYY-MM-DD HH:MM:SS）。
	Datetime string
	// URL 是完整 URL。
	URL string
	// Status 是状态码。
	Status int
	// Length 是响应长度。
	Length int64
	// Type 是 Content-Type。
	Type string
	// Redirect 是跳转地址。
	Redirect string
	// Elapsed 是耗时秒数。
	Elapsed float64
}

// StartInfo 保存报告元数据（启动时间与命令行）。
type StartInfo struct {
	// StartTime 是格式化的启动时间。
	StartTime string
	// Command 是（已脱敏的）命令行。
	Command string
}

// FormatName 返回格式的展示名（与 dirsearch 的 __format__ 一致）。
func FormatName(format string) string {
	if format == "md" {
		return "markdown"
	}
	if format == "sqlite" {
		return "sql"
	}
	return format
}

// FileExtension 返回格式的默认文件扩展名（与 __extension__ 一致）。
func FileExtension(format string) string {
	extensions := map[string]string{
		"simple": "txt", "plain": "txt", "json": "json", "xml": "xml",
		"md": "md", "csv": "csv", "html": "html", "sqlite": "sqlite",
	}
	return extensions[format]
}

// FileExistsError 表示输出文件已存在且无法解析。
type FileExistsError struct {
	// File 是冲突的文件路径。
	File string
}

// Error 实现 error 接口（消息与 dirsearch 一致）。
func (e *FileExistsError) Error() string {
	return fmt.Sprintf("Output file %s already exists", e.File)
}

// fileReport 是基于文件的报告的公共实现。
type fileReport struct {
	format   string
	meta     StartInfo
	mutex    sync.Mutex
	newFunc  func(r *fileReport) string
	saveFunc func(r *fileReport, file string, result Result)
	parseFn  func(file string) (any, error)
}

// initiate 初始化报告文件：目录创建、已有文件校验或写入文件头。
func (r *fileReport) initiate(file string) error {
	if dir := filepath.Dir(file); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	if info, err := os.Stat(file); err == nil && !info.IsDir() && info.Size() > 0 {
		if err := r.validate(file); err != nil {
			return err
		}
		return nil
	}
	return utils.AtomicWriteText(file, r.newFunc(r))
}

// validate 校验已有文件可被解析（否则视为文件已存在冲突）。
func (r *fileReport) validate(file string) error {
	if _, err := r.parse(file); err != nil {
		return &FileExistsError{File: file}
	}
	return nil
}

// parse 优先使用格式特定解析器，默认按文本读取。
func (r *fileReport) parse(file string) (any, error) {
	if r.parseFn != nil {
		return r.parseFn(file)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

// save 追加一条结果（线程安全）。
func (r *fileReport) save(file string, result Result) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.saveFunc(r, file, result)
}

// appendText 以追加模式写入文本。
func appendText(file, data string) error {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(data)
	return err
}

// Manager 汇总多个报告格式并统一分发结果。
type Manager struct {
	formats  []string
	meta     StartInfo
	reports  map[string]*fileReport
	files    map[string]string
	target   string
	optionOf map[string]string // format -> 输出路径（可能含变量）
	Warning  func(string)
}

// NewManager 创建报告管理器。
// filePaths 将格式映射到输出文件路径（可含 {format} 等变量）。
func NewManager(formats []string, meta StartInfo, filePaths map[string]string) *Manager {
	return &Manager{
		formats:  formats,
		meta:     meta,
		reports:  map[string]*fileReport{},
		files:    map[string]string{},
		optionOf: filePaths,
	}
}

// buildReport 构建指定格式的报告实现。
func buildReport(format string, meta StartInfo) *fileReport {
	r := &fileReport{format: format, meta: meta}
	switch format {
	case "simple":
		r.newFunc = newSimple
		r.saveFunc = saveSimple
	case "plain":
		r.newFunc = newPlain
		r.saveFunc = savePlain
	case "json":
		r.newFunc = newJSON
		r.saveFunc = saveJSON
		r.parseFn = parseJSON
	case "xml":
		r.newFunc = newXML
		r.saveFunc = saveXML
		r.parseFn = parseXML
	case "md":
		r.newFunc = newMarkdown
		r.saveFunc = saveMarkdown
	case "csv":
		r.newFunc = newCSV
		r.saveFunc = saveCSV
		r.parseFn = parseCSV
	case "html":
		r.newFunc = newHTML
		r.saveFunc = saveHTML
		r.parseFn = parseHTML
	}
	return r
}

// Prepare 初始化全部报告（解析输出路径变量）。
func (m *Manager) Prepare(target string) error {
	m.target = target
	for _, format := range m.formats {
		// 数据库类格式在当前构建中不支持（需要外部驱动）
		if format == "sqlite" || format == "mysql" || format == "postgresql" {
			if m.Warning != nil {
				m.Warning(fmt.Sprintf(
					"Warning: %s output format is not supported in this build, skipping",
					format))
			}
			continue
		}
		if format == "sqlite" {
			continue
		}

		pathTemplate := m.optionOf[format]
		if pathTemplate == "" {
			continue
		}
		file := m.FormatPath(pathTemplate, target, format)

		report := buildReport(format, m.meta)
		if report.newFunc == nil {
			continue
		}
		if err := report.initiate(file); err != nil {
			return err
		}
		m.reports[format] = report
		m.files[format] = file
	}
	return nil
}

// Save 向全部报告写入一条结果。
func (m *Manager) Save(result Result) {
	for format, report := range m.reports {
		pathTemplate := m.optionOf[format]
		if pathTemplate == "" {
			continue
		}
		file := m.FormatPath(pathTemplate, result.URL, format)
		report.save(file, result)
	}
}

// Finish 完成全部报告。
func (m *Manager) Finish() {}

// FormatPath 渲染输出路径中的变量（与 dirsearch 的 ReportManager.format 一致）。
func (m *Manager) FormatPath(pathTemplate, target, format string) string {
	parsed, err := url.Parse(target)
	host, scheme, port := "", "", ""
	if err == nil {
		host = parsed.Hostname()
		scheme = parsed.Scheme
		port = parsed.Port()
		if port == "" {
			port = strconv.Itoa(settings.StandardPorts[scheme])
		}
	}
	replacer := strings.NewReplacer(
		"{datetime}", utils.FormatDatetimeForPath(m.meta.StartTime),
		"{date}", dateOf(m.meta.StartTime),
		"{host}", host,
		"{scheme}", scheme,
		"{port}", port,
		"{format}", FormatName(format),
		"{extension}", FileExtension(format),
	)
	return replacer.Replace(pathTemplate)
}

// dateOf 提取 "YYYY-MM-DD HH:MM:SS" 的日期部分。
func dateOf(startTime string) string {
	if idx := strings.Index(startTime, " "); idx >= 0 {
		return startTime[:idx]
	}
	return startTime
}

// ElapsedMS 返回毫秒整数（未计时时为 0）。
func ElapsedMS(elapsed float64) int {
	if elapsed <= 0 {
		return 0
	}
	return int(elapsed * 1000)
}

// EnsureStartInfo 构建默认的元数据。
func EnsureStartInfo() StartInfo {
	return StartInfo{
		StartTime: time.Now().Format("2006-01-02 15:04:05"),
		Command:   "",
	}
}
