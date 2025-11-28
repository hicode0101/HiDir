package parse

import (
	"strings"
)

// HeadersParser 解析HTTP请求头
type HeadersParser struct {
	headers string
}

// NewHeadersParser 创建新的HeadersParser实例
func NewHeadersParser(headers string) *HeadersParser {
	return &HeadersParser{headers: headers}
}

// Parse 解析请求头为键值对
func (hp *HeadersParser) Parse() map[string]string {
	result := make(map[string]string)

	lines := strings.Split(hp.headers, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}

	return result
}

// ParseHeaders 解析请求头字符串为键值对
func ParseHeaders(headers string) map[string]string {
	parser := NewHeadersParser(headers)
	return parser.Parse()
}
