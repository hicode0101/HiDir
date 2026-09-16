package utils

import (
	"regexp"
	"strings"
)

// 爬虫相关正则与集合，移植自 dirsearch 的 lib/utils/crawl.py。
var (
	// escapedSlashRegex 处理 JSON/JS 中转义的斜杠（\/、\u002f、\x2f）。
	escapedSlashRegex = regexp.MustCompile(`(?i)\\(?:/|u002f|x2f)`)

	// robotsPathRegex 从 robots.txt 提取 Allow/Disallow 路径。
	robotsPathRegex = regexp.MustCompile(`(?:Allow|Disallow): /(.*)`)

	// htmlTagRegex 粗粒度匹配 HTML 标签（开标签）。
	htmlTagRegex = regexp.MustCompile(`(?is)<([a-zA-Z][a-zA-Z0-9]*)\b([^>]*)>`)

	// htmlAttrRegex 匹配标签内的属性键值对（双引号/单引号/无引号）。
	htmlAttrRegex = regexp.MustCompile(`(?is)([a-zA-Z_:@][a-zA-Z0-9_.:-]*)\s*=\s*("([^"]*)"|'([^']*)'|([^\s"'=<>` + "`" + `]+))`)
)

// CrawlResult 过滤爬取结果：去掉空路径、媒体后缀路径并去重。
func CrawlFilter(paths []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, path := range paths {
		path = CleanPath(path, true, false)
		resourcePath := path
		if idx := strings.Index(path, "?"); idx >= 0 {
			resourcePath = path[:idx]
		}
		if path == "" || hasMediaSuffix(resourcePath) {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

// hasMediaSuffix 判断路径是否以媒体扩展名结尾（大小写不敏感）。
func hasMediaSuffix(path string) bool {
	lower := strings.ToLower(path)
	for _, suffix := range mediaSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

// mediaSuffixes 是媒体扩展名后缀列表（避免与 settings 包循环依赖而内联）。
var mediaSuffixes = []string{
	".webm", ".mkv", ".avi", ".ts", ".mov", ".qt", ".amv", ".mp4", ".m4p",
	".m4v", ".mp3", ".swf", ".mpg", ".mpeg", ".jpg", ".jpeg", ".pjpeg",
	".png", ".woff", ".svg", ".webp", ".bmp", ".pdf", ".wav", ".vtt",
}

// Crawl 根据响应类型爬取同源路径。
// contentType 用于区分 HTML / robots.txt / 纯文本三种模式；
// fullPath 是响应路径（如 "robots.txt"）。
func Crawl(respURL, fullPath, contentType, content string) []string {
	parts := strings.SplitN(respURL, "/", 4)
	scope := strings.Join(parts[:3], "/") + "/"

	if strings.Contains(contentType, "text/html") {
		return CrawlHTML(respURL, scope, content)
	}
	if fullPath == "robots.txt" {
		return CrawlRobots(content)
	}
	return CrawlText(scope, content)
}

// CrawlText 在纯文本内容中提取以 scope 为前缀的路径。
func CrawlText(scope, content string) []string {
	content = escapedSlashRegex.ReplaceAllString(content, "/")
	scopeRegex, err := regexp.Compile("(?i)" + regexp.QuoteMeta(scope))
	if err != nil {
		return nil
	}
	var paths []string
	for _, loc := range scopeRegex.FindAllStringIndex(content, -1) {
		preceding := byte(0)
		if loc[0] > 0 {
			preceding = content[loc[0]-1]
		}
		var quote byte
		if preceding == '"' || preceding == '\'' || preceding == '`' {
			quote = preceding
		}
		var b strings.Builder
		for i := loc[1]; i < len(content); i++ {
			c := content[i]
			if (quote != 0 && c == quote) || !isTextURLChar(c) {
				break
			}
			b.WriteByte(c)
		}
		path := b.String()
		if quote == 0 {
			path = trimUnquotedURL(path)
		}
		if path != "" {
			paths = append(paths, path)
		}
	}
	return CrawlFilter(paths)
}

// isTextURLChar 判断字符是否属于 URL 文本字符集（RFC 3986 + 数组括号）。
func isTextURLChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	return strings.IndexByte("-._~%!$&'()*+,;=:@/?#[]", c) >= 0
}

// trimUnquotedURL 去除未加引号 URL 尾部的多余括号与标点。
func trimUnquotedURL(path string) string {
	path = strings.TrimRight(path, ".,")
	for _, pair := range [2][2]string{{"(", ")"}, {"[", "]"}} {
		excess := strings.Count(path, pair[1]) - strings.Count(path, pair[0])
		if excess < 0 {
			excess = 0
		}
		trimmed := strings.TrimRight(path, pair[1])
		trailing := len(path) - len(trimmed)
		if n := min(excess, trailing); n > 0 {
			path = path[:len(path)-n]
		}
	}
	return path
}

// CrawlRobots 从 robots.txt 内容中提取路径。
func CrawlRobots(content string) []string {
	matches := robotsPathRegex.FindAllStringSubmatch(content, -1)
	var paths []string
	for _, m := range matches {
		if len(m) > 1 {
			paths = append(paths, m[1])
		}
	}
	return CrawlFilter(paths)
}

// CrawlHTML 从 HTML 内容中提取同源路径（模拟 BeautifulSoup 遍历）。
func CrawlHTML(respURL, scope, content string) []string {
	baseURL := respURL
	var results []string

	for _, tag := range htmlTagRegex.FindAllStringSubmatch(content, -1) {
		tagName := strings.ToLower(tag[1])
		attrs := tag[2]

		// 处理 <base href="...">：后续相对路径以此为基准
		if tagName == "base" {
			if href := attrValue(attrs, "href"); href != "" {
				if resolved := safeResolve(baseURL, BrowserURLValue(strings.TrimSpace(href))); resolved != "" {
					baseURL = resolved
				}
			}
			continue
		}

		if !isCrawlTag(tagName) {
			continue
		}

		for _, attr := range htmlAttrRegex.FindAllStringSubmatch(attrs, -1) {
			attrName := strings.ToLower(attr[1])
			if !isCrawlAttribute(attrName) {
				continue
			}
			value := attr[3]
			if value == "" {
				value = attr[4]
			}
			if value == "" {
				value = attr[5]
			}
			if value == "" {
				continue
			}
			if attrName == "srcset" {
				for _, candidate := range SrcsetURLs(value) {
					if path := sameOriginCrawlPath(scope, baseURL, candidate); path != "" {
						results = append(results, path)
					}
				}
				continue
			}
			if path := sameOriginCrawlPath(scope, baseURL, value); path != "" {
				results = append(results, path)
			}
		}
	}
	return CrawlFilter(results)
}

// isCrawlTag 判断标签是否在爬取标签列表中。
func isCrawlTag(name string) bool {
	for _, t := range []string{
		"a", "area", "base", "blockquote", "button", "embed", "form", "frame",
		"frameset", "html", "iframe", "img", "input", "ins", "noframes",
		"object", "q", "script", "source",
	} {
		if t == name {
			return true
		}
	}
	return false
}

// isCrawlAttribute 判断属性是否在爬取属性列表中。
func isCrawlAttribute(name string) bool {
	for _, a := range []string{
		"action", "cite", "data", "formaction", "href", "longdesc",
		"poster", "src", "srcset", "xmlns",
	} {
		if a == name {
			return true
		}
	}
	return false
}

// attrValue 从标签属性文本中提取指定属性值。
func attrValue(attrs, name string) string {
	for _, attr := range htmlAttrRegex.FindAllStringSubmatch(attrs, -1) {
		if strings.EqualFold(attr[1], name) {
			if attr[3] != "" {
				return attr[3]
			}
			if attr[4] != "" {
				return attr[4]
			}
			return attr[5]
		}
	}
	return ""
}

// safeResolve 解析相对引用并校验合法性，失败时返回空字符串。
func safeResolve(base, ref string) string {
	resolved := ResolveURL(base, ref)
	scheme, _, _, ok := origin(resolved)
	if !ok {
		return ""
	}
	if scheme == "data" || scheme == "javascript" {
		return ""
	}
	return resolved
}

// sameOriginCrawlPath 若 value 与 scope 同源则返回其路径，否则返回空。
func sameOriginCrawlPath(scope, baseURL, value string) string {
	resolved := safeResolve(baseURL, BrowserURLValue(strings.TrimSpace(value)))
	if resolved == "" {
		return ""
	}
	return SameOriginPath(scope, resolved)
}

// BrowserURLValue 规范化 URL 路径中的反斜杠（浏览器行为），
// 查询串与片段保持原样。
func BrowserURLValue(value string) string {
	query := strings.Index(value, "?")
	fragment := strings.Index(value, "#")
	pathEnd := len(value)
	for _, pos := range []int{query, fragment} {
		if pos >= 0 && pos < pathEnd {
			pathEnd = pos
		}
	}
	return strings.ReplaceAll(value[:pathEnd], "\\", "/") + value[pathEnd:]
}

// SrcsetURLs 解析 srcset 属性中的 URL 列表。
func SrcsetURLs(value string) []string {
	var out []string
	runes := []byte(value)
	pos, length := 0, len(runes)
	for pos < length {
		for pos < length && (isASCIIWhitespace(runes[pos]) || runes[pos] == ',') {
			pos++
		}
		if pos >= length {
			return out
		}
		start := pos
		for pos < length && !isASCIIWhitespace(runes[pos]) {
			pos++
		}
		u := string(runes[start:pos])
		if strings.HasSuffix(u, ",") {
			u = strings.TrimRight(u, ",")
			if u != "" {
				out = append(out, u)
			}
			continue
		}
		if u != "" {
			out = append(out, u)
		}
		// 跳过描述符（如 2x 或 100w），直到遇到不在括号内的逗号
		parentheses := 0
		for pos < length {
			c := runes[pos]
			pos++
			if c == '(' {
				parentheses++
			} else if c == ')' && parentheses > 0 {
				parentheses--
			} else if c == ',' && parentheses == 0 {
				break
			}
		}
	}
	return out
}

// isASCIIWhitespace 判断 HTML 空白字符。
func isASCIIWhitespace(c byte) bool {
	return c == '\t' || c == '\n' || c == '\f' || c == '\r' || c == ' '
}
