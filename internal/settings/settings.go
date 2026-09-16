// Package settings 定义 HiDir 的全局常量与默认配置。
// 这些常量与 dirsearch 的 lib/core/settings.py 保持兼容。
package settings

import (
	"runtime"
	"time"
)

// Version 是 HiDir 的版本号，格式与 dirsearch 相同：<major>.<minor>.<revision>
const Version = "1.0.0"

// Banner 是启动时打印的 ASCII Logo（风格与 dirsearch 保持一致）。
const Banner = `
  _|. _ _  _  _  _ _|_    v` + Version + `
 (_||| _) (/_(_|| (_| )
`

// DefaultEncoding 是默认的响应解码字符集。
const DefaultEncoding = "utf-8"

// FileBasedOutputFormats 是支持的基于文件的报告格式。
// 与 dirsearch 的 FILE_BASED_OUTPUT_FORMATS 一致。
var FileBasedOutputFormats = []string{"simple", "plain", "json", "xml", "md", "csv", "html", "sqlite"}

// CommonExtensions 是 -e "*" 时使用的常见扩展名列表。
var CommonExtensions = []string{"php", "jsp", "asp", "aspx", "do", "action", "cgi", "html", "htm", "js", "tar.gz"}

// ArchiveExtensions 是归档类扩展名（用于备份文件探测）。
var ArchiveExtensions = []string{"zip", "tar", "tar.gz", "tgz", "gz", "7z", "rar", "bak"}

// BackupExtensions 是备份类扩展名（归档扩展名 + 常见备份后缀）。
var BackupExtensions = append(append([]string{}, ArchiveExtensions...),
	"bkp", "bkup", "old", "swn", "swp")

// MediaExtensions 是媒体类扩展名，overwrite-extensions 与爬虫会跳过这些扩展名。
var MediaExtensions = []string{
	"webm", "mkv", "avi", "ts", "mov", "qt", "amv", "mp4", "m4p", "m4v",
	"mp3", "swf", "mpg", "mpeg", "jpg", "jpeg", "pjpeg", "png", "woff",
	"svg", "webp", "bmp", "pdf", "wav", "vtt",
}

// ExcludeOverwriteExtensions 是 overwrite-extensions 时不会被覆盖的扩展名
// （媒体扩展名 + 常见静态资源扩展名）。
var ExcludeOverwriteExtensions = append(append([]string{}, MediaExtensions...),
	"axd", "cache", "coffee", "conf", "config", "css", "dll", "lock", "log",
	"key", "pub", "properties", "ini", "jar", "js", "json", "toml", "txt",
	"xml", "yaml", "yml")

// DBEngines 是词表模板 %DB%/%DB_ENGINE% 占位符支持的数据库引擎。
var DBEngines = []string{"mysql", "postgres", "postgresql", "sqlite", "mariadb", "mongodb", "redis"}

// AuthenticationTypes 是支持的认证类型列表。
var AuthenticationTypes = []string{"basic", "digest", "bearer", "ntlm", "jwt"}

// ProxySchemes 是合法的代理 URL 前缀。
var ProxySchemes = []string{"http://", "https://", "socks5://", "socks5h://", "socks4://", "socks4a://"}

// StandardPorts 是各协议的默认端口。
var StandardPorts = map[string]int{"http": 80, "https": 443}

// DefaultTestPrefixes 是默认测试的通配符前缀（除用户指定的 prefixes 之外）。
var DefaultTestPrefixes = []string{".", ".ht"}

// DefaultTestSuffixes 是默认测试的通配符后缀（除用户指定的 suffixes 之外）。
var DefaultTestSuffixes = []string{"/", "~"}

// DefaultTorProxies 是 --tor 选项使用的 Tor SOCKS5 代理列表。
var DefaultTorProxies = []string{"socks5://127.0.0.1:9050", "socks5://127.0.0.1:9150"}

// DefaultHeaders 是每个请求默认携带的 HTTP 头（可被用户头覆盖）。
var DefaultHeaders = map[string]string{
	"user-agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36",
	"accept":          "*/*",
	"accept-encoding": "*",
	"keep-alive":      "timeout=15, max=1000",
	"cache-control":   "max-age=0",
}

// 以下为扫描过程中的关键标记与阈值常量，与 dirsearch 保持一致。

// ReflectedPathMarker 用于通配符重定向正则生成时替换被反射的路径。
const ReflectedPathMarker = "__REFLECTED_PATH__"

// WildcardTestPointMarker 是通配符测试路径中的替换点标记。
const WildcardTestPointMarker = "__WILDCARD_POINT__"

// ExtensionTag 是词表中扩展名占位符关键字。
const ExtensionTag = "%ext%"

// ExtensionRecognitionRegex 用于识别路径是否已带扩展名。
const ExtensionRecognitionRegex = `(?i)\w+([.][a-zA-Z0-9]{2,5}){1,3}~?$`

// RobotsTxtRegex 用于从 robots.txt 中提取路径。
const RobotsTxtRegex = `(?:Allow|Disallow): /(.*)`

// Unknown 表示未知协议 scheme 占位符。
const Unknown = "unknown"

// DummyWord 是动态内容检测中使用的哑词。
const DummyWord = "dummyasdf"

// SocketTimeout 是协议探测（scheme 检测）时的连接超时时间。
const SocketTimeout = 6 * time.Second

// MaxResponseSize 是单个响应体捕获的最大字节数（80MB）。
const MaxResponseSize = 80 * 1024 * 1024

// MaxConsecutiveRequestErrors 是连续请求错误的最大次数，超过后跳过当前目标。
const MaxConsecutiveRequestErrors = 75

// ThreadedWorkerShutdownTimeout 是同步 worker 停止时的最长等待时间。
const ThreadedWorkerShutdownTimeout = 2 * time.Second

// AutoCalibrationExtraSamples 是自动校准时额外采样的次数。
const AutoCalibrationExtraSamples = 2

// AmbiguousSimilarityThreshold 是模糊通配符判定时的相似度阈值。
const AmbiguousSimilarityThreshold = 0.9

// AmbiguousSimilarityMaxContentLength 是模糊判定允许的最大内容长度。
const AmbiguousSimilarityMaxContentLength = 262144

// AutoCalibrationDuplicateThreshold 是重复响应自动校准的默认阈值。
const AutoCalibrationDuplicateThreshold = 8

// AutoCalibrationForcedThreshold 是 --auto-calibration 开启时的阈值。
const AutoCalibrationForcedThreshold = 3

// AutoCalibrationMinContentLength 是参与自动校准指纹的最小内容长度。
const AutoCalibrationMinContentLength = 32

// CrawlAttributes 是 HTML 爬取时关注的属性列表。
var CrawlAttributes = []string{
	"action", "cite", "data", "formaction", "href", "longdesc",
	"poster", "src", "srcset", "xmlns",
}

// CrawlTags 是 HTML 爬取时关注的标签列表。
var CrawlTags = []string{
	"a", "area", "base", "blockquote", "button", "embed", "form", "frame",
	"frameset", "html", "iframe", "img", "input", "ins", "noframes",
	"object", "q", "script", "source",
}

// IsWindows 报告当前是否运行在 Windows 平台。
var IsWindows = runtime.GOOS == "windows"
