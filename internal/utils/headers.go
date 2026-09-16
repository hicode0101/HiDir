package utils

import (
	"strings"
)

// Headers 表示大小写不敏感的 HTTP 头集合。
type Headers map[string]string

// NewHeaders 将任意键值规范化为小写键的 Header 集合。
func NewHeaders(pairs map[string]string) Headers {
	out := make(Headers, len(pairs))
	for k, v := range pairs {
		out[strings.ToLower(k)] = v
	}
	return out
}

// ParseHeaders 解析 "Key: Value" 形式的头文本（支持多行），
// 行为与 dirsearch 的 HeadersParser（基于 email.parser）一致：
// 折行（以空白开头）会合并到上一行，非法行（不含冒号）会被忽略。
func ParseHeaders(text string) (Headers, error) {
	headers := Headers{}
	if strings.TrimSpace(text) == "" {
		return headers, nil
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var lastKey string
	for _, line := range lines {
		switch {
		case line == "":
			lastKey = ""
			continue
		case line[0] == ' ' || line[0] == '\t':
			// 折行：延续上一个头
			if lastKey != "" {
				headers[lastKey] += " " + strings.TrimSpace(line)
			}
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			return nil, &HeaderFormatError{Line: line}
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if existing, ok := headers[key]; ok {
			headers[key] = existing + ", " + value
		} else {
			headers[key] = value
		}
		lastKey = key
	}
	return headers, nil
}

// HeaderFormatError 表示非法的头格式（缺少冒号）。
type HeaderFormatError struct {
	// Line 是出错的原始行。
	Line string
}

// Error 实现 error 接口。
func (e *HeaderFormatError) Error() string {
	return "invalid header format: " + e.Line
}

// HeadersText 将 Header 集合渲染为 "key: value" 多行文本。
func HeadersText(h map[string]string) string {
	var b strings.Builder
	for k, v := range h {
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(v)
		b.WriteString("\n")
	}
	return b.String()
}
