package httpx

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
)

// RequestError 表示请求失败（对应 dirsearch 的 RequestException）。
type RequestError struct {
	// Message 是错误描述。
	Message string
}

// Error 实现 error 接口。
func (e *RequestError) Error() string {
	return e.Message
}

// newRequestError 构造请求错误。
func newRequestError(format string, args ...any) *RequestError {
	return &RequestError{Message: fmt.Sprintf(format, args...)}
}

// AuthType 枚举支持的认证类型。
type AuthType int

// 认证类型常量。
const (
	AuthNone AuthType = iota
	AuthBasic
	AuthDigest
	AuthBearer
	AuthJWT
	AuthNTLM
)

// ParseAuthType 将认证类型字符串转换为枚举值。
func ParseAuthType(name string) AuthType {
	switch strings.ToLower(name) {
	case "basic":
		return AuthBasic
	case "digest":
		return AuthDigest
	case "bearer":
		return AuthBearer
	case "jwt":
		return AuthJWT
	case "ntlm":
		return AuthNTLM
	default:
		return AuthNone
	}
}

// Credentials 保存认证凭据。
type Credentials struct {
	// Username 与 Password 用于 basic/digest。
	Username string
	Password string
	// Token 用于 bearer/jwt。
	Token string
	// Type 是认证类型。
	Type AuthType
}

// ParseCredentials 解析凭据字符串：
// bearer/jwt 视整串为 token，其余按 user:password 拆分（无冒号时密码为空）。
func ParseCredentials(credType, credential string) Credentials {
	c := Credentials{Type: ParseAuthType(credType)}
	if c.Type == AuthBearer || c.Type == AuthJWT {
		c.Token = credential
		return c
	}
	if idx := strings.Index(credential, ":"); idx >= 0 {
		c.Username = credential[:idx]
		c.Password = credential[idx+1:]
	} else {
		c.Username = credential
	}
	return c
}

// ApplyAuth 将认证信息写入请求头。
// 返回 (headers, err)；NTLM 等不支持的类型返回错误。
func ApplyAuth(headers map[string]string, cred Credentials) error {
	switch cred.Type {
	case AuthBasic:
		token := base64.StdEncoding.EncodeToString(
			[]byte(cred.Username + ":" + cred.Password))
		headers["authorization"] = "Basic " + token
	case AuthBearer, AuthJWT:
		headers["authorization"] = "Bearer " + cred.Token
	case AuthDigest:
		// Digest 认证需要 401 挑战后动态生成，由请求器处理
	case AuthNone:
		// 无认证
	default:
		return newRequestError("unsupported authentication type: ntlm")
	}
	return nil
}

// DigestChallenge 表示 WWW-Authenticate 挑战头解析结果。
type DigestChallenge struct {
	Realm     string
	Nonce     string
	QOP       string
	Algorithm string
	Opaque    string
}

// ParseDigestChallenge 解析 WWW-Authenticate: Digest ... 头。
func ParseDigestChallenge(header string) (*DigestChallenge, error) {
	lower := strings.ToLower(header)
	if !strings.HasPrefix(lower, "digest ") {
		return nil, fmt.Errorf("not a digest challenge")
	}
	challenge := &DigestChallenge{Algorithm: "MD5"}
	for _, part := range splitChallenge(strings.TrimPrefix(header[6:], " ")) {
		key, value, _ := strings.Cut(part, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.Trim(strings.TrimSpace(value), "\"")
		switch key {
		case "realm":
			challenge.Realm = value
		case "nonce":
			challenge.Nonce = value
		case "qop":
			challenge.QOP = value
		case "algorithm":
			challenge.Algorithm = value
		case "opaque":
			challenge.Opaque = value
		}
	}
	if challenge.Nonce == "" {
		return nil, fmt.Errorf("missing nonce in digest challenge")
	}
	return challenge, nil
}

// splitChallenge 按逗号拆分挑战参数（忽略引号内的逗号）。
func splitChallenge(s string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	for _, c := range s {
		switch {
		case c == '"':
			inQuote = !inQuote
			current.WriteRune(c)
		case c == ',' && !inQuote:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(c)
		}
	}
	if strings.TrimSpace(current.String()) != "" {
		parts = append(parts, current.String())
	}
	return parts
}

// DigestAuthHeader 生成 Digest 认证的 Authorization 头。
// 实现 RFC 2617（qop=auth），与 Python urllib3 的行为兼容。
func DigestAuthHeader(method, uri string, cred Credentials, challenge *DigestChallenge, nc int) string {
	ha1 := md5hex(cred.Username + ":" + challenge.Realm + ":" + cred.Password)
	ha2 := md5hex(method + ":" + uri)
	ncValue := fmt.Sprintf("%08x", nc)
	cnonce := randomCNonce()

	var response string
	if strings.Contains(challenge.QOP, "auth") {
		response = md5hex(ha1 + ":" + challenge.Nonce + ":" + ncValue + ":" +
			cnonce + ":auth:" + ha2)
	} else {
		response = md5hex(ha1 + ":" + challenge.Nonce + ":" + ha2)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Digest username=\"%s\", realm=\"%s\", nonce=\"%s\", uri=\"%s\"",
		cred.Username, challenge.Realm, challenge.Nonce, uri))
	if strings.Contains(challenge.QOP, "auth") {
		b.WriteString(fmt.Sprintf(", qop=auth, nc=%s, cnonce=\"%s\"", ncValue, cnonce))
	}
	b.WriteString(fmt.Sprintf(", response=\"%s\", algorithm=%s", response, challenge.Algorithm))
	if challenge.Opaque != "" {
		b.WriteString(fmt.Sprintf(", opaque=\"%s\"", challenge.Opaque))
	}
	return b.String()
}

// md5hex 计算 MD5 十六进制摘要。
func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// randomCNonce 生成随机客户端 nonce。
func randomCNonce() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = "0123456789abcdef"[rand.Intn(16)]
	}
	return string(b)
}

// AddProxyAuthentication 在代理 URL 中补充认证信息（若尚无 userinfo）。
// 与 dirsearch 的 add_proxy_authentication 一致。
func AddProxyAuthentication(proxy, credential string) string {
	if credential == "" {
		return proxy
	}
	parsed, err := url.Parse(proxy)
	if err != nil || parsed.User != nil {
		return proxy
	}
	username, password, _ := strings.Cut(credential, ":")
	userinfo := url.QueryEscape(username)
	if password != "" || strings.Contains(credential, ":") {
		userinfo += ":" + url.QueryEscape(password)
	}
	return strings.Replace(proxy, "://", "://"+userinfo+"@", 1)
}

// NTLMPlaceholder 保留 NTLM 支持的占位说明（当前构建不支持）。
const NTLMPlaceholder = "ntlm requires the NTLM handshake which is not available in this build"

// ensureSHA1Used 保持导入整洁（SHA-1 供未来 NTLM 消息签名使用）。
var _ = sha1.Size
