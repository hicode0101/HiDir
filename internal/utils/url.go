package utils

import (
	"net/url"
	"strings"
)

// defaultPorts 与 dirsearch 的 _DEFAULT_PORTS 一致。
var defaultPorts = map[string]int{"http": 80, "https": 443}

// origin 提取 URL 的 (scheme, host, port) 三元组用于同源比较。
func origin(value string) (scheme, host string, port int, ok bool) {
	parsed, err := url.Parse(value)
	if err != nil {
		return "", "", 0, false
	}
	scheme = strings.ToLower(parsed.Scheme)
	host = strings.ToLower(parsed.Hostname())
	if scheme == "" || parsed.Host == "" {
		return "", "", 0, false
	}
	port = portOrDefault(parsed.Port(), scheme)
	return scheme, host, port, true
}

// portOrDefault 将端口字符串转换为整数，空值时回退到协议默认端口。
func portOrDefault(portStr, scheme string) int {
	if portStr == "" {
		return defaultPorts[scheme]
	}
	port := 0
	for _, c := range portStr {
		if c < '0' || c > '9' {
			return -1
		}
		port = port*10 + int(c-'0')
	}
	return port
}

// SameOrigin 判断两个 URL 是否同源。
func SameOrigin(first, second string) bool {
	s1, h1, p1, ok1 := origin(first)
	s2, h2, p2, ok2 := origin(second)
	return ok1 && ok2 && s1 == s2 && h1 == h2 && p1 == p2
}

// SameOriginPath 若 location 与 baseURL 同源，则返回 location 的路径部分；
// 否则返回空字符串。
func SameOriginPath(baseURL, location string) string {
	parsed, err := url.Parse(location)
	if err != nil {
		return ""
	}
	resolved := ResolveURL(baseURL, location)
	if !SameOrigin(baseURL, resolved) {
		return ""
	}
	_ = parsed
	return ParsePath(resolved)
}

// CleanPath 去除路径中的查询串与 DOM 片段。
func CleanPath(path string, keepQueries, keepFragment bool) string {
	if !keepFragment {
		if idx := strings.Index(path, "#"); idx >= 0 {
			path = path[:idx]
		}
	}
	if !keepQueries {
		if idx := strings.Index(path, "?"); idx >= 0 {
			path = path[:idx]
		}
	}
	return path
}

// ParsePath 从完整 URL 中提取路径部分；输入已是路径时原样返回（去掉开头一个 /）。
// 与 dirsearch 的 parse_path 逻辑一致。
func ParsePath(value string) string {
	// 尝试按 scheme://host/path 拆分
	idx := strings.Index(value, "//")
	if idx > 0 {
		scheme := value[:idx]
		if !strings.HasSuffix(scheme, ":") || strings.Contains(scheme, "/") {
			// "://" 之前不是合法 scheme，按路径处理
			return LstripOnce(value, "/")
		}
		rest := value[idx+2:]
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
		return ""
	}
	if strings.HasPrefix(value, "//") {
		return LstripOnce(value, "/")
	}
	return LstripOnce(value, "/")
}

// EnsureTrailingPathSlash 保证 URL 路径以 "/" 结尾。
func EnsureTrailingPathSlash(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	return parsed.String()
}

// AppendQueryString 在不含查询串的路径后追加查询串（保留片段）。
func AppendQueryString(value, query string) string {
	if query == "" || strings.Contains(value, "?") {
		return value
	}
	path, fragment, hasFragment := strings.Cut(value, "#")
	result := path + "?" + query
	if hasFragment {
		result += "#" + fragment
	}
	return result
}

// ResolveURL 以 base 为基准解析相对/绝对引用（等价于 Python 的 urljoin）。
func ResolveURL(base, ref string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return ref
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return baseURL.ResolveReference(refURL).String()
}

// JoinRequestTarget 将 base URL 的路径与请求路径拼接为 origin-form 请求目标，
// 与 dirsearch 的 _join_request_target 一致。
func JoinRequestTarget(baseURL, quotedPath string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "/" + LstripOnce(quotedPath, "/")
	}
	basePath := parsed.Path
	if basePath == "" {
		basePath = "/"
	}
	var target string
	if basePath == "/" {
		target = "/"
	} else {
		target = strings.TrimSuffix(basePath, "/") + "/"
	}
	return target + LstripOnce(quotedPath, "/")
}

// MergePath 模拟浏览器点击相对链接时的路径合并行为。
func MergePath(rawURL, path string) string {
	parts := strings.Split(rawURL, "/")
	resolved := ResolveURL("/", path)
	resolved = LstripOnce(resolved, "/")
	parts[len(parts)-1] = resolved
	return strings.Join(parts, "/")
}
