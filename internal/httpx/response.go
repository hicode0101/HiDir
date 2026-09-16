// Package httpx 实现 HiDir 的 HTTP 请求层：
// 请求构造、重试、代理、认证、限速与响应解析，
// 对应 dirsearch 的 lib/connection/requester.py 与 response.py。
package httpx

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"hidir/internal/settings"
	"hidir/internal/utils"
)

// Response 表示一次 HTTP 响应的解析结果。
type Response struct {
	// Datetime 是响应接收时间（YYYY-MM-DD HH:MM:SS）。
	Datetime string
	// URL 是请求的完整 URL。
	URL string
	// FullPath 是从 URL 解析出的路径。
	FullPath string
	// Path 是去除查询串与片段后的干净路径。
	Path string
	// Status 是 HTTP 状态码。
	Status int
	// Headers 是小写键的响应头集合。
	Headers map[string]string
	// Redirect 是 Location 头（未跟随跳转时）。
	Redirect string
	// History 是重定向历史 URL 列表。
	History []string
	// Elapsed 是请求耗时（秒）。
	Elapsed float64
	// Content 是解码后的文本内容（二进制响应为空）。
	Content string
	// Body 是原始响应体字节。
	Body []byte
	// BodyComplete 表示响应体是否完整捕获。
	BodyComplete bool

	// normalized 缓存归一化内容；digest 缓存响应体摘要。
	normalized         string
	textNormalizedDone bool
	digest             []byte
}

// NewResponse 基于基本字段构建响应并完成路径解析。
func NewResponse(rawURL string, status int, headers map[string]string, body []byte) *Response {
	resp := &Response{
		Datetime:     time.Now().Format("2006-01-02 15:04:05"),
		URL:          rawURL,
		Status:       status,
		Headers:      headers,
		Redirect:     headers["location"],
		Body:         body,
		BodyComplete: true,
	}
	resp.FullPath = utils.ParsePath(rawURL)
	resp.Path = utils.CleanPath(resp.FullPath, false, false)
	return resp
}

// Type 返回 Content-Type（去掉参数部分），缺失时为 "unknown"。
func (r *Response) Type() string {
	if ct, ok := r.Headers["content-type"]; ok {
		return strings.TrimSpace(strings.Split(ct, ";")[0])
	}
	return settings.Unknown
}

// Length 返回响应长度：优先 Content-Length 头，否则响应体长度。
func (r *Response) Length() int64 {
	return utils.GetResponseLength(r.Headers["content-length"], len(r.Body))
}

// Size 返回人类可读的响应大小。
func (r *Response) Size() string {
	return utils.GetReadableSize(r.Length())
}

// Text 返回响应文本（解码内容或原始字节的 UTF-8 近似）。
func (r *Response) Text() string {
	if r.Content != "" {
		return r.Content
	}
	return strings.ToValidUTF8(string(r.Body), string([]byte{0xEF, 0xBF, 0xBD}))
}

// Words 返回响应文本的词数。
func (r *Response) Words() int {
	return len(strings.Fields(r.Text()))
}

// Lines 返回响应文本的行数。
func (r *Response) Lines() int {
	if r.Text() == "" {
		return 0
	}
	return strings.Count(r.Text(), "\n") + 1
}

// normalizedContentCache 缓存归一化内容。
func (r *Response) NormalizedContent() string {
	if r.normalized == "" && r.textNormalizedDone == false {
		r.normalized = utils.NormalizeDynamicContent(r.Text())
		r.textNormalizedDone = true
	}
	return r.normalized
}

// bodyDigest 计算响应体 SHA-256 摘要。
func (r *Response) BodyDigest() []byte {
	if r.digest == nil {
		sum := sha256.Sum256(r.Body)
		r.digest = sum[:]
	}
	return r.digest
}

// HasSameBody 判断两个响应体是否完全相同。
func (r *Response) HasSameBody(other *Response) bool {
	if r == other {
		return true
	}
	return r.BodyComplete && other.BodyComplete &&
		hex.EncodeToString(r.BodyDigest()) == hex.EncodeToString(other.BodyDigest())
}

// EqualResponse 判断两个响应是否等价（状态码、跳转、响应体一致）。
func (r *Response) EqualResponse(other *Response) bool {
	return r.Status == other.Status &&
		r.Redirect == other.Redirect &&
		r.HasSameBody(other)
}

// FilterFingerprint 返回与路径无关的过滤指纹（用于 --filter-threshold）。
func (r *Response) FilterFingerprint() string {
	body := r.Content
	if body != "" {
		body = strings.ReplaceAll(body,
			utils.CleanPath(r.FullPath, false, false), "")
	} else {
		body = hex.EncodeToString(r.BodyDigest())
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d|%s", r.Status, body)))
	return hex.EncodeToString(sum[:8])
}

// ResponseFingerprint 返回响应综合指纹（状态码、类型、跳转、归一化内容），
// 用于自动校准判定。与 dirsearch 的 response_fingerprint 语义一致。
func (r *Response) ResponseFingerprint() string {
	path := utils.CleanPath(r.FullPath, false, false)
	path = strings.Trim(path, "/")
	body := r.NormalizedContent()
	redirect := utils.CleanPath(r.Redirect, false, false)
	if path != "" {
		body = strings.ReplaceAll(body, path, "__PATH__")
		redirect = strings.ReplaceAll(redirect, path, "__PATH__")
	}
	// 用归一化内容的前 4096 字符与长度桶构建稳定指纹
	chunk := body
	if len(chunk) > 4096 {
		chunk = chunk[:4096]
	}
	sum := sha256.Sum256([]byte(chunk))
	return fmt.Sprintf("%d|%s|%s|%d|%s",
		r.Status, r.Type(), redirect, len(body)/64, hex.EncodeToString(sum[:8]))
}

// HeadersText 渲染响应头为多行文本。
func (r *Response) HeadersText() string {
	keys := make([]string, 0, len(r.Headers))
	for k := range r.Headers {
		keys = append(keys, k)
	}
	sortStrings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(r.Headers[k])
		b.WriteString("\n")
	}
	return b.String()
}

// DeclaredCharset 从 Content-Type 中提取 charset 参数。
func DeclaredCharset(contentType string) string {
	for _, parameter := range strings.Split(contentType, ";")[1:] {
		name, value, found := strings.Cut(parameter, "=")
		if found && strings.ToLower(strings.Trim(strings.TrimSpace(name), "'\"")) == "charset" {
			return strings.Trim(strings.TrimSpace(value), "'\"")
		}
	}
	return ""
}

// sortStrings 简单排序。
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
