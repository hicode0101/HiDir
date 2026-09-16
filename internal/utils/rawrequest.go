package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// RawRequest 表示从原始 HTTP 请求文件解析出的请求要素。
type RawRequest struct {
	// URL 是根据请求行与 Host 头还原的目标 URL。
	URL string
	// Method 是 HTTP 方法。
	Method string
	// Headers 是解析出的请求头。
	Headers map[string]string
	// Body 是请求体（可能为 nil）。
	Body []byte
}

// ParseRawContent 解析原始 HTTP 请求文本，行为与 dirsearch 的 parse_raw_content 一致：
// 支持 absolute-form（GET http://host/path）与 origin-form（GET /path + Host 头）。
func ParseRawContent(content []byte, scheme string) (*RawRequest, error) {
	head, body := splitHeadBody(content)

	lines := strings.Split(strings.ReplaceAll(string(head), "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return nil, fmt.Errorf("the raw request is formatively invalid")
	}

	// 解析请求行：METHOD TARGET [VERSION]
	requestLine := strings.Fields(strings.TrimSpace(lines[0]))
	if len(requestLine) < 2 {
		return nil, fmt.Errorf("the raw request is formatively invalid")
	}
	method := requestLine[0]
	target := requestLine[1]

	headers, err := ParseHeaders(strings.Join(lines[1:], "\n"))
	if err != nil {
		return nil, fmt.Errorf("the raw request is formatively invalid")
	}

	targetURL, err := targetFromRequestLine(target, headers, scheme)
	if err != nil {
		return nil, err
	}

	return &RawRequest{
		URL:     targetURL,
		Method:  method,
		Headers: headers,
		Body:    body,
	}, nil
}

// splitHeadBody 以空行（\r\n\r\n 或 \n\n）切分头部与请求体。
func splitHeadBody(content []byte) (head []byte, body []byte) {
	text := content
	for _, sep := range []string{"\r\n\r\n", "\n\n"} {
		if idx := strings.Index(string(text), sep); idx >= 0 {
			return text[:idx], []byte(string(text[idx+len(sep):]))
		}
	}
	return []byte(strings.Trim(string(text), "\r\n")), nil
}

// targetFromRequestLine 根据请求行目标与 Host 头还原完整 URL。
func targetFromRequestLine(target string, headers Headers, scheme string) (string, error) {
	// absolute-form：请求行中已包含完整 URL
	if parsed, err := url.Parse(target); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return normalizeRawTarget(parsed, true), nil
	}

	host := headers["host"]
	if host == "" {
		return "", fmt.Errorf("can't find the Host header in the raw request")
	}

	// origin-form：使用 Host 头拼接
	parsed, err := url.Parse(host)
	if err != nil {
		return "", fmt.Errorf("the raw request is formatively invalid")
	}
	_ = parsed

	path, query, _ := strings.Cut(target, "?")
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	result := host + path
	if query != "" {
		result += "?" + query
	}
	if scheme != "" {
		if !strings.Contains(scheme, "://") {
			result = scheme + "://" + result
		} else {
			result = scheme + result
		}
	}
	return result, nil
}

// normalizeRawTarget 输出规整后的绝对 URL。
func normalizeRawTarget(parsed *url.URL, absolute bool) string {
	path := parsed.Path
	if path == "" {
		path = "/"
	}
	out := url.URL{Scheme: parsed.Scheme, Host: parsed.Host, Path: path, RawQuery: parsed.RawQuery}
	if !absolute {
		out.Scheme = ""
	}
	return out.String()
}
