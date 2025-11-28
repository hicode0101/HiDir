package common

// 认证类型
var AUTHENTICATION_TYPES = []string{"basic", "bearer"}

// 常见扩展名
var COMMON_EXTENSIONS = []string{"php", "html", "htm", "asp", "aspx", "jsp", "jspx", "do", "action", "cgi", "pl", "py", "rb", "lua", "swf", "fla", "xml"}

// 默认Tor代理
var DEFAULT_TOR_PROXIES = []string{"socks5://127.0.0.1:9050"}

// 输出格式
var OUTPUT_FORMATS = []string{"simple", "plain", "json", "xml", "md", "csv", "html"}

// 默认HTTP头
var DEFAULT_HEADERS = map[string]string{
	"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	"Accept-Language":           "en-US,en;q=0.5",
	"Accept-Encoding":           "gzip, deflate",
	"Connection":                "keep-alive",
	"Upgrade-Insecure-Requests": "1",
}

// 标准端口
var STANDARD_PORTS = map[string]int{
	"http":  80,
	"https": 443,
}

// 默认测试前缀
var DEFAULT_TEST_PREFIXES = []string{"_", "."}

// 默认测试后缀
var DEFAULT_TEST_SUFFIXES = []string{"_", "."}

// 通配符测试标记
const WILDCARD_TEST_POINT_MARKER = "_dirsearch_"

// 最大连续请求错误数
const MAX_CONSECUTIVE_REQUEST_ERRORS = 5

// 暂停等待超时
const PAUSING_WAIT_TIMEOUT = 30

// 未知
const UNKNOWN = "unknown"

// 扩展名识别正则
const EXTENSION_RECOGNITION_REGEX = `\.([a-zA-Z0-9]+)$`
