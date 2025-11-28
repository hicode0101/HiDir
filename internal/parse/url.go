package parse

import (
	"net/url"
	"strings"
)

// CleanPath 清理路径
func CleanPath(path string) string {
	// 移除重复的斜杠
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	// 移除末尾的斜杠（如果有）
	path = strings.TrimSuffix(path, "/")

	return path
}

// ParsePath 解析URL路径
func ParsePath(urlStr string) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	return parsed.Path
}

// DetectScheme 检测URL的协议
func DetectScheme(host string, port int) string {
	// 这里简化实现，实际应该尝试连接来检测
	if port == 443 {
		return "https"
	}
	return "http"
}
