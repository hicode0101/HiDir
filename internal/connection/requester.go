package connection

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Response 存储HTTP响应

type Response struct {
	Status   int
	Content  string
	Length   int64
	Headers  map[string][]string
	Path     string
	FullPath string
	Redirect string
	History  []string
}

// Requester 处理HTTP请求
type Requester struct {
	client    *http.Client
	url       string
	auth      string
	authType  string
	proxyAuth string
	headers   map[string]string
	data      string
	method    string
}

// NewRequester 创建新的Requester实例
func NewRequester() *Requester {
	// 创建自定义的Transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 跳过证书验证
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}

	// 创建HTTP客户端
	client := &http.Client{
		Transport: transport,
		Timeout:   7500 * time.Millisecond, // 默认超时
	}

	return &Requester{
		client:  client,
		headers: make(map[string]string),
		method:  "GET",
	}
}

// SetURL 设置目标URL
func (r *Requester) SetURL(url string) {
	r.url = url
}

// SetAuth 设置认证信息
func (r *Requester) SetAuth(authType, auth string) {
	r.authType = authType
	r.auth = auth
}

// SetProxyAuth 设置代理认证信息
func (r *Requester) SetProxyAuth(proxyAuth string) {
	r.proxyAuth = proxyAuth
}

// SetHeaders 设置请求头
func (r *Requester) SetHeaders(headers map[string]string) {
	r.headers = headers
}

// SetData 设置请求数据
func (r *Requester) SetData(data string) {
	r.data = data
}

// SetMethod 设置HTTP方法
func (r *Requester) SetMethod(method string) {
	r.method = method
}

// Request 发送HTTP请求
func (r *Requester) Request(path string, proxy ...string) (*Response, error) {
	// 构建完整URL
	fullPath := r.url
	if !strings.HasSuffix(fullPath, "/") {
		fullPath += "/"
	}
	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}
	fullPath += path

	// 创建请求
	req, err := http.NewRequest(r.method, fullPath, strings.NewReader(r.data))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}

	// 设置认证
	if r.auth != "" {
		switch r.authType {
		case "basic":
			parts := strings.SplitN(r.auth, ":", 2)
			if len(parts) == 2 {
				req.SetBasicAuth(parts[0], parts[1])
			}
		case "bearer":
			req.Header.Set("Authorization", "Bearer "+r.auth)
		}
	}

	// 设置代理
	if len(proxy) > 0 && proxy[0] != "" {
		proxyURL, err := url.Parse(proxy[0])
		if err == nil {
			req.URL.Host = proxyURL.Host
			req.URL.Scheme = proxyURL.Scheme
		}
	}

	// 发送请求
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应内容
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 处理重定向
	redirect := ""
	history := []string{}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		redirect = resp.Header.Get("Location")
		if redirect != "" {
			history = append(history, fullPath)
		}
	}

	// 构建响应对象
	response := &Response{
		Status:   resp.StatusCode,
		Content:  string(content),
		Length:   int64(len(content)),
		Headers:  resp.Header,
		Path:     path,
		FullPath: fullPath,
		Redirect: redirect,
		History:  history,
	}

	return response, nil
}
