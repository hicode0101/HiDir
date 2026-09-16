package httpx

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"hidir/internal/settings"
	"hidir/internal/utils"
)

// RequesterConfig 汇总请求器的全部配置。
type RequesterConfig struct {
	// Method 是 HTTP 方法。
	Method string
	// Data 是请求体。
	Data []byte
	// Headers 是请求头集合（小写键）。
	Headers map[string]string
	// Timeout 是单请求超时秒数。
	Timeout float64
	// MaxRetries 是失败重试次数。
	MaxRetries int
	// MaxRate 是每秒最大请求数（0 不限）。
	MaxRate int
	// Proxies 是代理列表。
	Proxies []string
	// ProxyAuth 是代理认证凭据。
	ProxyAuth string
	// FollowRedirects 是否跟随重定向。
	FollowRedirects bool
	// Agents 是随机 User-Agent 列表（--random-agent）。
	Agents []string
	// DNSOverride 是 host:port 到 IP 的映射（--ip）。
	DNSOverride map[string]string
	// CertFile/KeyFile 是客户端证书。
	CertFile string
	KeyFile  string
	// Auth 是认证凭据。
	Auth Credentials
}

// Requester 是扫描使用的 HTTP 客户端。
type Requester struct {
	// URL 是当前目标的基础 URL（以 / 结尾）。
	URL string
	// Query 是附加在所有请求后的查询串。
	Query string
	// Config 是请求器配置。
	Config RequesterConfig

	client           *http.Client
	rateLimiter      *RateLimiter
	staticAuthHeader string // Basic/Bearer/JWT 的静态认证头
	digestHeader     string // 缓存的 Digest Authorization 头
	digestTried      bool   // 当前目标是否已尝试过 Digest 握手
	digestURIPath    string
}

// NewRequester 基于配置构建请求器。
func NewRequester(cfg RequesterConfig) (*Requester, error) {
	if cfg.Method == "" {
		cfg.Method = http.MethodGet
	}
	if cfg.Headers == nil {
		cfg.Headers = map[string]string{}
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10
	}
	r := &Requester{Config: cfg, rateLimiter: &RateLimiter{}}

	// 客户端证书
	var certificates []tls.Certificate
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, newRequestError("Failed to load client certificate: %s", err)
		}
		certificates = []tls.Certificate{cert}
	}

	// SOCKS4 系列代理不受支持
	for _, proxy := range cfg.Proxies {
		scheme := proxyScheme(proxy)
		if scheme == "socks4" || scheme == "socks4a" {
			return nil, newRequestError(
				"SOCKS4 proxies are not supported; use SOCKS5 instead")
		}
	}

	transport := &http.Transport{
		Proxy:                 r.proxyResolver(),
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true, Certificates: certificates},
		MaxIdleConnsPerHost:   256,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: time.Duration(cfg.Timeout * float64(time.Second)),
	}
	if len(cfg.DNSOverride) > 0 {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			if override, ok := cfg.DNSOverride[strings.ToLower(addr)]; ok {
				addr = override
			}
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, addr)
		}
	}

	// Basic/Bearer/JWT 认证头静态生成
	switch cfg.Auth.Type {
	case AuthBasic:
		token := base64.StdEncoding.EncodeToString(
			[]byte(cfg.Auth.Username + ":" + cfg.Auth.Password))
		r.staticAuthHeader = "Basic " + token
	case AuthBearer, AuthJWT:
		r.staticAuthHeader = "Bearer " + cfg.Auth.Token
	}

	jar, _ := cookiejar.New(nil)
	r.client = &http.Client{
		Transport: transport,
		Jar:       jar,
	}
	r.configureRedirects()
	return r, nil
}

// redirectRecorder 用于在重定向过程中记录历史 URL。
type redirectRecorder struct {
	history *[]string
}

// contextRedirectKey 是上下文中记录器的键。
type contextRedirectKey struct{}

// configureRedirects 设置重定向策略。
func (r *Requester) configureRedirects() {
	r.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !r.Config.FollowRedirects {
			return http.ErrUseLastResponse
		}
		if len(via) >= 30 {
			return newRequestError("stopped after 30 redirects")
		}
		// 记录重定向前的 URL 历史
		if value := req.Context().Value(contextRedirectKey{}); value != nil {
			recorder := value.(*redirectRecorder)
			if len(via) >= 1 {
				*recorder.history = append(*recorder.history, via[len(via)-1].URL.String())
			}
		}
		return nil
	}
}

// proxyScheme 提取代理协议（缺省 http）。
func proxyScheme(proxy string) string {
	if !strings.Contains(proxy, "://") {
		return "http"
	}
	return strings.ToLower(strings.SplitN(proxy, "://", 2)[0])
}

// proxyResolver 返回每请求随机选择代理的函数。
func (r *Requester) proxyResolver() func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		if len(r.Config.Proxies) == 0 {
			return nil, nil
		}
		proxy := r.Config.Proxies[rand.Intn(len(r.Config.Proxies))]
		if !hasAnyScheme(proxy) {
			proxy = "http://" + proxy
		}
		proxy = AddProxyAuthentication(proxy, r.Config.ProxyAuth)
		// socks5h 与 socks5 等价处理
		proxy = strings.Replace(proxy, "socks5h://", "socks5://", 1)
		return url.Parse(proxy)
	}
}

// hasAnyScheme 判断 URL 是否带协议前缀。
func hasAnyScheme(proxy string) bool {
	for _, scheme := range []string{"http://", "https://", "socks5://", "socks5h://", "socks4://", "socks4a://"} {
		if strings.HasPrefix(proxy, scheme) {
			return true
		}
	}
	return false
}

// SetURL 设置目标基础 URL。
func (r *Requester) SetURL(rawURL string) {
	r.URL = rawURL
	r.digestTried = false
	r.digestHeader = ""
}

// SetQuery 设置附加查询串。
func (r *Requester) SetQuery(query string) { r.Query = query }

// Rate 返回当前请求速率。
func (r *Requester) Rate() int { return r.rateLimiter.Rate() }

// RequestPath 附加查询串到路径。
func (r *Requester) RequestPath(path string) string {
	return utils.AppendQueryString(path, r.Query)
}

// Close 释放底层连接。
func (r *Requester) Close() {
	if transport, ok := r.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

// Do 发送一次请求（带重试与限速）。
// path 不以 "/" 开头；proxy 非空时强制使用该代理（用于 --replay-proxy）。
func (r *Requester) Do(path string, proxy string) (*Response, error) {
	r.rateLimiter.Wait(r.Config.MaxRate)

	requestPath := r.RequestPath(path)
	quoted := utils.SafeQuote(requestPath)
	fullURL := joinURL(r.URL, quoted)
	target := utils.JoinRequestTarget(r.URL, quoted)

	var lastErr error
	for attempt := 0; attempt < r.Config.MaxRetries+1; attempt++ {
		resp, err := r.doWithDigest(fullURL, target, quoted, proxy)
		if err == nil {
			return resp, nil
		}
		if requestErr, ok := err.(*RequestError); ok {
			// 认证/代理类的不可恢复错误直接返回
			if strings.Contains(requestErr.Message, "authentication") {
				return nil, err
			}
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = newRequestError("There was a problem in the request to: %s", fullURL)
	}
	return nil, lastErr
}

// doWithDigest 处理 Digest 认证的 401 挑战握手。
func (r *Requester) doWithDigest(fullURL, target, quotedPath, proxy string) (*Response, error) {
	resp, err := r.doOnce(fullURL, target, quotedPath, proxy, r.digestHeader)
	if err != nil {
		return nil, err
	}

	// Digest 握手：401 + WWW-Authenticate 时重放一次
	if resp.Status == 401 && r.Config.Auth.Type == AuthDigest && !r.digestTried {
		challengeHeader := resp.Headers["www-authenticate"]
		challenge, cerr := ParseDigestChallenge(challengeHeader)
		if cerr != nil {
			return resp, nil
		}
		r.digestTried = true
		r.digestHeader = DigestAuthHeader(r.Config.Method, "/"+quotedPath, r.Config.Auth, challenge, 1)
		return r.doOnce(fullURL, target, quotedPath, proxy, r.digestHeader)
	}
	return resp, nil
}

// doOnce 执行单次请求与响应解析。
func (r *Requester) doOnce(fullURL, target, quotedPath, proxy, authHeader string) (*Response, error) {
	timeout := time.Duration(r.Config.Timeout * float64(time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var bodyReader io.Reader
	if len(r.Config.Data) > 0 {
		bodyReader = bytes.NewReader(r.Config.Data)
	}

	req, err := http.NewRequestWithContext(ctx, r.Config.Method, fullURL, bodyReader)
	if err != nil {
		return nil, newRequestError("Invalid URL: %s", fullURL)
	}

	// 组装请求头
	headers := make(map[string]string, len(r.Config.Headers)+2)
	for k, v := range r.Config.Headers {
		headers[k] = v
	}
	if len(r.Config.Agents) > 0 {
		headers["user-agent"] = r.Config.Agents[rand.Intn(len(r.Config.Agents))]
	}
	if len(r.Config.Data) > 0 {
		if _, ok := headers["content-type"]; !ok {
			headers["content-type"] = utils.GuessMimetype(r.Config.Data)
		}
	}
	if authHeader == "" {
		authHeader = r.staticAuthHeader
	}
	if authHeader != "" {
		headers["authorization"] = authHeader
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 记录重定向历史
	history := &[]string{}
	req = req.WithContext(context.WithValue(ctx, contextRedirectKey{}, &redirectRecorder{history}))

	// 代理认证失败检测
	if proxy != "" {
		req = req.WithContext(context.WithValue(req.Context(), proxyContextKey{}, true))
	}

	start := time.Now()
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, r.classifyError(err, fullURL)
	}
	defer resp.Body.Close()

	if proxy != "" && resp.StatusCode == 407 {
		return nil, newRequestError("Proxy authentication required")
	}

	parsed, err := parseHTTPResponse(fullURL, resp, start, *history)
	if err != nil {
		return nil, err
	}
	// 保留原始（含模糊字符）的路径信息
	parsed.URL = fullURL
	parsed.FullPath = utils.ParsePath(target)
	parsed.Path = utils.CleanPath(parsed.FullPath, false, false)
	return parsed, nil
}

// proxyContextKey 标记请求是否经由显式代理。
type proxyContextKey struct{}

// classifyError 将底层错误转换为可读的 RequestError（消息与 dirsearch 一致）。
func (r *Requester) classifyError(err error, fullURL string) error {
	msg := err.Error()
	lower := strings.ToLower(msg)
	parsed, _ := url.Parse(fullURL)

	switch {
	case strings.Contains(lower, "proxyconnect"), strings.Contains(lower, "proxy"):
		if strings.Contains(lower, "authentication") || strings.Contains(lower, "407") {
			return newRequestError("Proxy authentication required")
		}
		return newRequestError("Cannot establish the proxy connection")
	case strings.Contains(lower, "no such host"), strings.Contains(lower, "lookup"),
		strings.Contains(lower, "nxdomain"):
		return newRequestError("Couldn't resolve DNS")
	case strings.Contains(lower, "context deadline exceeded"), strings.Contains(lower, "timeout"),
		strings.Contains(lower, "timed out"):
		return newRequestError("Request timeout: %s", fullURL)
	case strings.Contains(lower, "certificate"):
		return newRequestError("SSL certificate verification failed: %s", fullURL)
	case strings.Contains(lower, "tls"), strings.Contains(lower, "ssl"), strings.Contains(lower, "handshake"):
		return newRequestError("SSL error: %s", msg)
	case strings.Contains(lower, "too many redirects"):
		return newRequestError("Too many redirects: %s", fullURL)
	case strings.Contains(lower, "connection refused"), strings.Contains(lower, "connection reset"),
		strings.Contains(lower, "connect:"), strings.Contains(lower, "connectex"),
		strings.Contains(lower, "actively refused"), strings.Contains(lower, "refused the connection"),
		strings.Contains(lower, "unreachable"):
		return newRequestError("Cannot connect to: %s", parsed.Host)
	default:
		return newRequestError("There was a problem in the request to: %s", fullURL)
	}
}

// parseHTTPResponse 读取并解析 HTTP 响应体。
func parseHTTPResponse(fullURL string, resp *http.Response, start time.Time, history []string) (*Response, error) {
	limit := int64(settings.MaxResponseSize)
	buf := &bytes.Buffer{}
	if _, err := io.CopyN(buf, resp.Body, limit); err != nil && err != io.EOF {
		return nil, newRequestError("Failed to read response body: %s", fullURL)
	}
	data := buf.Bytes()
	complete := int64(len(data)) < limit

	headers := map[string]string{}
	for key, values := range resp.Header {
		headers[strings.ToLower(key)] = strings.Join(values, ", ")
	}

	response := NewResponse(fullURL, resp.StatusCode, headers, data)
	response.BodyComplete = complete
	response.History = history
	response.Elapsed = time.Since(start).Seconds()

	decodeContent(response)
	return response, nil
}

// decodeContent 按字符集解码响应体（二进制内容置空，与 dirsearch 一致）。
func decodeContent(resp *Response) {
	contentType := resp.Headers["content-type"]
	charset := DeclaredCharset(contentType)
	isTextual := isTextualMediaType(contentType)
	knownCharset := charset != "" && isKnownCharset(charset)

	if utils.IsBinary(resp.Body) && !(isTextual && knownCharset) {
		resp.Content = ""
		return
	}
	resp.Content = decodeBody(resp.Body, charset)
}

// isTextualMediaType 判断是否为文本类媒体类型。
func isTextualMediaType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if strings.HasPrefix(ct, "text/") {
		return true
	}
	if ct == "application/json" || ct == "application/xml" {
		return true
	}
	return strings.HasSuffix(ct, "+json") || strings.HasSuffix(ct, "+xml")
}

// isKnownCharset 判断字符集是否受支持。
func isKnownCharset(charset string) bool {
	switch normalizeCharset(charset) {
	case "utf-8", "iso-8859-1", "us-ascii", "windows-1252":
		return true
	}
	return false
}

// normalizeCharset 规范化字符集名称。
func normalizeCharset(charset string) string {
	return strings.ToLower(strings.TrimSpace(charset))
}

// decodeBody 以指定字符集解码，未支持时按 UTF-8 宽松解码。
func decodeBody(body []byte, charset string) string {
	switch normalizeCharset(charset) {
	case "iso-8859-1", "windows-1252", "latin-1":
		// Latin-1：字节直接映射到码点
		runes := make([]rune, len(body))
		for i, b := range body {
			runes[i] = rune(b)
		}
		return string(runes)
	default:
		return strings.ToValidUTF8(string(body), string([]byte{0xEF, 0xBF, 0xBD}))
	}
}

// joinURL 拼接基础 URL 与请求路径。
func joinURL(base, path string) string {
	if strings.HasSuffix(base, "/") && strings.HasPrefix(path, "/") {
		return base + strings.TrimPrefix(path, "/")
	}
	if !strings.HasSuffix(base, "/") && !strings.HasPrefix(path, "/") {
		return base + "/" + path
	}
	return base + path
}
