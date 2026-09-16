// Package utils 提供 HiDir 各模块共用的工具函数，
// 对应 dirsearch 的 lib/utils 与 lib/parse 中的通用逻辑。
package utils

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// URLSafeChars 是 safequote 时不会被百分号编码的字符（ASCII 标点），与 dirsearch 一致。
const URLSafeChars = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

// SafeQuote 对路径进行百分号编码，保留字母数字、`_.-~` 以及所有 ASCII 标点，
// 行为与 Python 的 quote(s, safe=string.punctuation) 一致。
func SafeQuote(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreserved(c) || strings.IndexByte(URLSafeChars, c) >= 0 {
			b.WriteByte(c)
			continue
		}
		b.WriteString(fmt.Sprintf("%%%02X", c))
	}
	return b.String()
}

// isUnreserved 判断字节是否为 RFC 3986 非保留字符（含波浪号）。
func isUnreserved(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_' || c == '~'
}

// StripAndUniquify 去除每个元素的空白、丢弃空元素并按顺序去重。
func StripAndUniquify(items []string) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

// SplitCSV 将逗号分隔字符串拆分并 StripAndUniquify。
func SplitCSV(value string) []string {
	if value == "" {
		return nil
	}
	return StripAndUniquify(strings.Split(value, ","))
}

// LstripOnce 只移除一次字符串开头的 pattern 前缀。
func LstripOnce(s, pattern string) string {
	if pattern != "" && strings.HasPrefix(s, pattern) {
		return s[len(pattern):]
	}
	return s
}

// RstripOnce 只移除一次字符串结尾的 pattern 后缀。
func RstripOnce(s, pattern string) string {
	if pattern != "" && strings.HasSuffix(s, pattern) {
		return s[:len(s)-len(pattern)]
	}
	return s
}

// GetValidFilename 将 Windows 文件名中的非法字符替换为下划线。
func GetValidFilename(s string) string {
	replacer := strings.NewReplacer(
		"\"", "_", "*", "_", "<", "_", ">", "_", "?", "_",
		"\\", "_", "|", "_", "/", "_", ":", "_",
	)
	return replacer.Replace(s)
}

// roundHalfEven 实现 Python round(n/d) 的银行家舍入语义。
func roundHalfEven(n, d int64) int64 {
	if d == 0 {
		return 0
	}
	sign := int64(1)
	if n < 0 {
		sign = -1
		n = -n
	}
	q := n / d
	r := n % d
	switch {
	case r*2 > d:
		q++
	case r*2 == d:
		if q%2 == 1 {
			q++
		}
	}
	return sign * q
}

// GetReadableSize 将字节数转换为人类可读大小（如 4KB），
// 舍入行为与 dirsearch 的 get_readable_size（Python round 银行家舍入）一致。
func GetReadableSize(num int64) string {
	const base = int64(1024)
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	n := num
	for _, unit := range units {
		if -base < n && n < base {
			return fmt.Sprintf("%d%s", n, unit)
		}
		n = roundHalfEven(n, base)
	}
	return fmt.Sprintf("%dTB", n)
}

// GetResponseLength 依据 Content-Length 头返回响应长度，缺失时回退到响应体长度。
func GetResponseLength(contentLength string, bodyLength int) int64 {
	if contentLength == "" {
		return int64(bodyLength)
	}
	length, err := strconv.ParseInt(strings.TrimSpace(contentLength), 10, 64)
	if err != nil || length < 0 {
		return int64(bodyLength)
	}
	return length
}

// textChars 是判断二进制内容的参考字符表（与 dirsearch 的 TEXT_CHARS 一致）：
// 控制字符中仅保留 BS/HT/LF/FF/CR/ESC，以及 0x20-0xFF（不含 DEL）。
var textChars [256]bool

func init() {
	for _, b := range []byte{7, 8, 9, 10, 12, 13, 27} {
		textChars[b] = true
	}
	for b := 0x20; b < 0x100; b++ {
		if b != 0x7F {
			textChars[b] = true
		}
	}
}

// IsBinary 检查响应体中是否含有二进制字节（与 Python bytes.translate 判定一致）。
func IsBinary(body []byte) bool {
	for _, b := range body {
		if !textChars[b] {
			return true
		}
	}
	return false
}

// IsIPv6 判断字符串是否为 IPv6 地址。
func IsIPv6(s string) bool {
	return strings.Count(s, ":") >= 2
}

// IPRange 将 CIDR 网段展开为 IP 列表（支持 IPv4 与 IPv6）。
func IPRange(subnet string) ([]string, error) {
	_, network, err := net.ParseCIDR(strings.TrimSpace(subnet))
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %s", subnet)
	}
	var ips []string
	for ip := append(net.IP(nil), network.IP...); network.Contains(ip); incIP(ip) {
		ips = append(ips, ip.String())
		// 防御超大网段（如 IPv6 ::/0），限制展开数量
		if len(ips) >= 1<<20 {
			break
		}
	}
	return ips, nil
}

// incIP 将 IP 自增一（用于遍历网段）。
func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}
