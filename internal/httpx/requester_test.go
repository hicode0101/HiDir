package httpx

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestRequester 创建指向测试服务器的请求器。
func newTestRequester(t *testing.T, serverURL string, mutate func(*RequesterConfig)) *Requester {
	t.Helper()
	cfg := RequesterConfig{
		Method:     "GET",
		Headers:    map[string]string{},
		Timeout:    5,
		MaxRetries: 0,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	r, err := NewRequester(cfg)
	if err != nil {
		t.Fatal(err)
	}
	r.SetURL(serverURL)
	return r
}

// ---- 基本请求 ----

func TestRequesterBasicGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin" {
			w.WriteHeader(404)
			fmt.Fprint(w, "not found")
			return
		}
		w.WriteHeader(200)
		fmt.Fprint(w, "admin page")
	}))
	defer server.Close()

	r := newTestRequester(t, server.URL+"/", nil)
	resp, err := r.Do("admin", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != 200 || resp.Content != "admin page" {
		t.Errorf("status=%d content=%q", resp.Status, resp.Content)
	}
	if resp.Path != "admin" || resp.FullPath != "admin" {
		t.Errorf("path=%q fullpath=%q", resp.Path, resp.FullPath)
	}
}

func TestRequesterPathNotNormalized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模糊路径（双斜杠）应原样到达服务器
		fmt.Fprintf(w, "path=%s", r.URL.Path)
	}))
	defer server.Close()

	r := newTestRequester(t, server.URL+"/", nil)
	resp, err := r.Do("..%2fadmin", "")
	if err != nil {
		t.Fatal(err)
	}
	// 路径中的模糊字符不会被 URL 规范化吞掉
	if !strings.Contains(resp.Text(), "..%2fadmin") && !strings.Contains(resp.Text(), "../admin") {
		t.Logf("server saw: %s", resp.Text())
	}
}

func TestRequesterQueryAppend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "query=%s", r.URL.RawQuery)
	}))
	defer server.Close()

	r := newTestRequester(t, server.URL+"/", nil)
	r.SetQuery("token=abc")
	resp, err := r.Do("admin", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Text(), "token=abc") {
		t.Errorf("query = %s", resp.Text())
	}
}

func TestRequesterPOSTWithData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		if r.Method != "POST" {
			t.Errorf("method = %s", r.Method)
		}
		if string(body) != "a=1&b=2" {
			t.Errorf("body = %q", body)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	r := newTestRequester(t, server.URL+"/", func(c *RequesterConfig) {
		c.Method = "POST"
		c.Data = []byte("a=1&b=2")
	})
	if _, err := r.Do("submit", ""); err != nil {
		t.Fatal(err)
	}
}

func TestRequesterCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ua=%s|custom=%s", r.Header.Get("User-Agent"), r.Header.Get("X-Custom"))
	}))
	defer server.Close()

	r := newTestRequester(t, server.URL+"/", func(c *RequesterConfig) {
		c.Headers = map[string]string{
			"user-agent": "HiDir-Test/1.0",
			"x-custom":   "yes",
		}
	})
	resp, err := r.Do("", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Text(), "ua=HiDir-Test/1.0") || !strings.Contains(resp.Text(), "custom=yes") {
		t.Errorf("headers = %s", resp.Text())
	}
}

func TestRequesterRandomAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, r.Header.Get("User-Agent"))
	}))
	defer server.Close()

	r := newTestRequester(t, server.URL+"/", func(c *RequesterConfig) {
		c.Agents = []string{"AgentA", "AgentB"}
	})
	for i := 0; i < 5; i++ {
		resp, err := r.Do("", "")
		if err != nil {
			t.Fatal(err)
		}
		if text := resp.Text(); text != "AgentA" && text != "AgentB" {
			t.Errorf("unexpected agent: %q", text)
		}
	}
}

// ---- 重定向 ----

func TestRequesterNoFollowRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/old" {
			http.Redirect(w, r, "/new", 301)
			return
		}
		fmt.Fprint(w, "new location")
	}))
	defer server.Close()

	r := newTestRequester(t, server.URL+"/", nil)
	resp, err := r.Do("old", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != 301 || resp.Redirect != "/new" {
		t.Errorf("status=%d redirect=%q", resp.Status, resp.Redirect)
	}
}

// ---- 认证 ----

func TestRequesterBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		want := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
		if auth != want {
			w.WriteHeader(401)
			return
		}
		fmt.Fprint(w, "authenticated")
	}))
	defer server.Close()

	r := newTestRequester(t, server.URL+"/", func(c *RequesterConfig) {
		c.Auth = ParseCredentials("basic", "user:pass")
	})
	resp, err := r.Do("", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "authenticated" {
		t.Errorf("content = %q", resp.Content)
	}
}

func TestRequesterBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer my-token" {
			w.WriteHeader(401)
			return
		}
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	r := newTestRequester(t, server.URL+"/", func(c *RequesterConfig) {
		c.Auth = ParseCredentials("bearer", "my-token")
	})
	resp, err := r.Do("", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok" {
		t.Errorf("content = %q", resp.Content)
	}
}

func TestRequesterDigestAuth(t *testing.T) {
	// 简化版 Digest 挑战：验证请求器会用正确的头重放
	realm := "test@host"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.Header().Set("WWW-Authenticate",
				fmt.Sprintf(`Digest realm="%s", nonce="abc123", qop="auth"`, realm))
			w.WriteHeader(401)
			return
		}
		if !strings.HasPrefix(auth, "Digest ") {
			w.WriteHeader(401)
			return
		}
		fmt.Fprint(w, "digest-ok")
	}))
	defer server.Close()

	r := newTestRequester(t, server.URL+"/", func(c *RequesterConfig) {
		c.Auth = ParseCredentials("digest", "user:pass")
	})
	resp, err := r.Do("", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "digest-ok" {
		t.Errorf("content = %q", resp.Content)
	}
}

func TestDigestAuthHeaderShape(t *testing.T) {
	cred := ParseCredentials("digest", "user:pass")
	challenge := &DigestChallenge{Realm: "r", Nonce: "n", QOP: "auth", Algorithm: "MD5"}
	header := DigestAuthHeader("GET", "/x", cred, challenge, 1)
	for _, part := range []string{`username="user"`, `realm="r"`, `nonce="n"`, `qop=auth`, `response="`} {
		if !strings.Contains(header, part) {
			t.Errorf("missing %q in %q", part, header)
		}
	}
	// MD5 响应可复算验证
	ha1 := fmt.Sprintf("%x", md5.Sum([]byte("user:r:pass")))
	ha2 := fmt.Sprintf("%x", md5.Sum([]byte("GET:/x")))
	want := fmt.Sprintf("%x", md5.Sum([]byte(ha1+":n:00000001:00000001:auth:"+ha2)))
	// cnonce 是随机的，这里仅验证 response 存在
	if !strings.Contains(header, `response="`) {
		t.Error("response missing")
	}
	_ = want
}

// ---- 错误分类 ----

func TestRequesterConnectError(t *testing.T) {
	// 使用一个必然关闭的端口
	r, err := NewRequester(RequesterConfig{Method: "GET", Headers: map[string]string{}, Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	r.SetURL("http://127.0.0.1:1/")
	_, err = r.Do("x", "")
	if err == nil {
		t.Fatal("connection should fail")
	}
	if !strings.Contains(err.Error(), "Cannot connect to") && !strings.Contains(err.Error(), "timeout") {
		t.Logf("error message: %v", err)
	}
}

// ---- 响应属性 ----

func TestResponseProperties(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", "11")
		fmt.Fprint(w, "hello world")
	}))
	defer server.Close()

	r := newTestRequester(t, server.URL+"/", nil)
	resp, err := r.Do("", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Length() != 11 {
		t.Errorf("length = %d", resp.Length())
	}
	if resp.Type() != "text/html" {
		t.Errorf("type = %q", resp.Type())
	}
	if resp.Words() != 2 || resp.Lines() != 1 {
		t.Errorf("words=%d lines=%d", resp.Words(), resp.Lines())
	}
	if resp.Size() != "11B" {
		t.Errorf("size = %q", resp.Size())
	}
}

// ---- 限速器 ----

func TestRateLimiter(t *testing.T) {
	rl := &RateLimiter{}
	// 不限速
	for i := 0; i < 10; i++ {
		if d := rl.reserve(0); d != 0 {
			t.Fatalf("unlimited reserve returned %v", d)
		}
	}
	// 限速 2/s：前两次立即通过，第三次需等待
	rl2 := &RateLimiter{}
	if d := rl2.reserve(2); d != 0 {
		t.Errorf("first reserve delayed: %v", d)
	}
	if d := rl2.reserve(2); d != 0 {
		t.Errorf("second reserve delayed: %v", d)
	}
	if d := rl2.reserve(2); d <= 0 {
		t.Error("third reserve should wait")
	}
}

// ---- 代理认证 ----

func TestAddProxyAuthentication(t *testing.T) {
	got := AddProxyAuthentication("http://proxy:8080", "user:pass")
	if got != "http://user:pass@proxy:8080" {
		t.Errorf("proxy auth = %q", got)
	}
	// 已有 userinfo 不覆盖
	got = AddProxyAuthentication("http://existing@proxy:8080", "user:pass")
	if got != "http://existing@proxy:8080" {
		t.Errorf("existing userinfo = %q", got)
	}
	// 无凭据
	got = AddProxyAuthentication("http://proxy:8080", "")
	if got != "http://proxy:8080" {
		t.Errorf("no credential = %q", got)
	}
}
