package report

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"hidir/internal/utils"
)

// newSimple 生成 simple 格式的空文件内容。
func newSimple(r *fileReport) string { return "" }

// saveSimple 追加 URL 行。
func saveSimple(r *fileReport, file string, result Result) {
	_ = appendText(file, result.URL+"\n")
}

// newPlain 生成 plain 格式头部。
func newPlain(r *fileReport) string {
	return fmt.Sprintf("# Dirsearch started at %s as: %s\n",
		r.meta.StartTime, r.meta.Command)
}

// savePlain 追加状态行。
func savePlain(r *fileReport, file string, result Result) {
	data := fmt.Sprintf("%d %6s %s", result.Status, utils.GetReadableSize(result.Length), result.URL)
	if ms := ElapsedMS(result.Elapsed); ms > 0 {
		data += fmt.Sprintf("  (%dms)", ms)
	}
	if result.Redirect != "" {
		data += "  ->  " + result.Redirect
	}
	_ = appendText(file, data+"\n")
}

// jsonReport 表示 JSON 报告结构。
type jsonReport struct {
	Info    jsonInfo    `json:"info"`
	Results []jsonEntry `json:"results"`
}

type jsonInfo struct {
	Args string `json:"args"`
	Time string `json:"time"`
}

type jsonEntry struct {
	URL           string  `json:"url"`
	Status        int     `json:"status"`
	ContentLength int64   `json:"contentLength"`
	ContentType   string  `json:"contentType"`
	Redirect      string  `json:"redirect"`
	Elapsed       float64 `json:"elapsed,omitempty"`
}

// newJSON 生成 JSON 报告头。
func newJSON(r *fileReport) string {
	data := jsonReport{Info: jsonInfo{Args: r.meta.Command, Time: r.meta.StartTime}}
	out, _ := json.MarshalIndent(data, "", "    ")
	return string(out)
}

// parseJSON 解析既有 JSON 报告。
func parseJSON(file string) (any, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var report jsonReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return report, nil
}

// saveJSON 追加结果并整体重写。
func saveJSON(r *fileReport, file string, result Result) {
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}
	var report jsonReport
	if json.Unmarshal(data, &report) != nil {
		return
	}
	entry := jsonEntry{
		URL:           result.URL,
		Status:        result.Status,
		ContentLength: result.Length,
		ContentType:   result.Type,
		Redirect:      result.Redirect,
	}
	if ms := ElapsedMS(result.Elapsed); ms > 0 {
		entry.Elapsed = result.Elapsed
	}
	report.Results = append(report.Results, entry)
	out, _ := json.MarshalIndent(report, "", "    ")
	_ = utils.AtomicWriteText(file, string(out))
}

// xmlResult 表示 XML 报告结构。
type xmlScan struct {
	XMLName xml.Name    `xml:"dirsearchscan"`
	Args    string      `xml:"args,attr"`
	Time    string      `xml:"time,attr"`
	Results []xmlResult `xml:"result"`
}

type xmlResult struct {
	URL           string `xml:"url,attr"`
	Status        int    `xml:"status"`
	ContentLength int64  `xml:"contentLength"`
	ContentType   string `xml:"contentType"`
	Redirect      string `xml:"redirect"`
	Elapsed       string `xml:"elapsed,omitempty"`
}

// newXML 生成 XML 报告头。
func newXML(r *fileReport) string {
	scan := xmlScan{Args: r.meta.Command, Time: r.meta.StartTime}
	return marshalXML(scan)
}

// marshalXML 序列化 XML（带缩进与声明）。
func marshalXML(scan xmlScan) string {
	out, _ := xml.MarshalIndent(scan, "", "    ")
	return xml.Header[:len(xml.Header)-1] + "\n" + string(out)
}

// parseXML 解析既有 XML 报告。
func parseXML(file string) (any, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var scan xmlScan
	if err := xml.Unmarshal(data, &scan); err != nil {
		return nil, err
	}
	return scan, nil
}

// saveXML 追加结果并整体重写。
func saveXML(r *fileReport, file string, result Result) {
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}
	var scan xmlScan
	if xml.Unmarshal(data, &scan) != nil {
		return
	}
	entry := xmlResult{
		URL:           result.URL,
		Status:        result.Status,
		ContentLength: result.Length,
		ContentType:   result.Type,
		Redirect:      result.Redirect,
	}
	if ms := ElapsedMS(result.Elapsed); ms > 0 {
		entry.Elapsed = fmt.Sprintf("%.3f", result.Elapsed)
	}
	scan.Results = append(scan.Results, entry)
	_ = utils.AtomicWriteText(file, marshalXML(scan))
}

// newMarkdown 生成 Markdown 表头。
func newMarkdown(r *fileReport) string {
	header := "### Information\n"
	header += fmt.Sprintf("Command: %s\n", r.meta.Command)
	header += fmt.Sprintf("Time: %s\n", r.meta.StartTime)
	header += "\n"
	header += "URL | Status | Size | Content Type | Redirection | Elapsed (ms)\n"
	header += "----|--------|------|--------------|-------------|-------------\n"
	return header
}

// saveMarkdown 追加表格行。
func saveMarkdown(r *fileReport, file string, result Result) {
	elapsed := "-"
	if ms := ElapsedMS(result.Elapsed); ms > 0 {
		elapsed = fmt.Sprintf("%d", ms)
	}
	row := fmt.Sprintf("%s | %d | %d | %s | %s | %s\n",
		result.URL, result.Status, result.Length, result.Type, result.Redirect, elapsed)
	_ = appendText(file, row)
}

// csvHeader 是 CSV 报告的表头。
var csvHeader = []string{"URL", "Status", "Size", "Content Type", "Redirection", "Elapsed (s)"}

// newCSV 生成 CSV 表头。
func newCSV(r *fileReport) string {
	return csvRow(csvHeader)
}

// csvRow 序列化一行 CSV。
func csvRow(fields []string) string {
	escaped := make([]string, len(fields))
	for i, field := range fields {
		if strings.ContainsAny(field, ",\"\n\r") {
			escaped[i] = `"` + strings.ReplaceAll(field, `"`, `""`) + `"`
		} else {
			escaped[i] = field
		}
	}
	return strings.Join(escaped, ",") + "\n"
}

// parseCSV 解析既有 CSV 并校验表头。
func parseCSV(file string) (any, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty CSV file")
	}
	expected := strings.Join(csvHeader, ",")
	if strings.TrimRight(lines[0], "\n") != expected {
		return nil, fmt.Errorf("CSV header mismatch in %s: expected %s, got %s",
			file, expected, lines[0])
	}
	return lines, nil
}

// saveCSV 追加结果行。
func saveCSV(r *fileReport, file string, result Result) {
	elapsed := ""
	if result.Elapsed > 0 {
		elapsed = fmt.Sprintf("%.3f", result.Elapsed)
	}
	row := csvRow([]string{
		result.URL,
		fmt.Sprintf("%d", result.Status),
		fmt.Sprintf("%d", result.Length),
		result.Type,
		result.Redirect,
		elapsed,
	})
	_ = appendText(file, row)
}
