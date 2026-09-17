// Package controller 实现扫描主控流程：
// 目标解析、目录队列、递归、时间限制、交互暂停与会话持久化，
// 对应 dirsearch 的 lib/controller/controller.py。
package controller

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"hidir/internal/httpx"
	"hidir/internal/options"
	"hidir/internal/report"
	"hidir/internal/scanner"
	"hidir/internal/settings"
	"hidir/internal/utils"
	"hidir/internal/view"
	"hidir/internal/wordlist"
)

// InterruptType 表示扫描中断类型。
type InterruptType int

// 中断类型常量。
const (
	// InterruptQuit 表示用户请求退出（保存/不保存）。
	InterruptQuit InterruptType = iota
	// InterruptSkipTarget 表示跳过当前目标。
	InterruptSkipTarget
	// InterruptNextDir 表示跳到下一个扫描目录。
	InterruptNextDir
)

// Interrupt 携带中断类型与提示消息。
type Interrupt struct {
	// Type 是中断类型。
	Type InterruptType
	// Message 是要展示的消息。
	Message string
}

// Error 实现 error 接口。
func (e *Interrupt) Error() string { return e.Message }

// OutputEntry 保存一段历史输出。
type OutputEntry struct {
	// StartTime 是该段输出对应运行的启动时间。
	StartTime string `json:"start_time"`
	// Output 是输出内容。
	Output string `json:"output"`
}

// sessionPayload 是会话文件的 JSON 结构。
type sessionPayload struct {
	// Version 是会话格式版本。
	Version int `json:"version"`
	// Args 是原始命令行参数（恢复时重新解析）。
	Args []string `json:"args"`
	// StartTime 是扫描开始时间。
	StartTime string `json:"start_time"`
	// TargetURL 是当前目标。
	TargetURL string `json:"url"`
	// Controller 保存计数器与剩余队列。
	Controller struct {
		JobsProcessed     int      `json:"jobs_processed"`
		Errors            int      `json:"errors"`
		ConsecutiveErrors int      `json:"consecutive_errors"`
		DirectoriesLeft   []string `json:"directories_left"`
		URLsLeft          []string `json:"urls_left"`
	} `json:"controller"`
	// OutputHistory 是历史输出。
	OutputHistory []OutputEntry `json:"output_history"`
}

// Controller 是扫描主控制器。
type Controller struct {
	// Options 是解析后的扫描选项。
	Options *options.Options
	// ResourceFS 是词表数据库资源。
	ResourceFS *utils.ResourceFS
	// UI 是终端输出。
	UI *view.CLI

	dictionary *wordlist.Dictionary
	blacklists wordlist.Blacklists
	reporter   *report.Manager
	requester  *httpx.Requester
	fuzzer     *scanner.Fuzzer

	directories       []string
	passedUrls        map[string]bool
	jobsProcessed     int
	errors            int
	consecutiveErrors int
	startTime         time.Time
	targetStartTime   time.Time
	basePath          string
	targetURL         string

	stateMutex sync.Mutex
	// currentInterrupt 是待处理的中断。
	currentInterrupt *Interrupt

	// sigChan 是 Ctrl+C/终止信号通道（由 Run 创建），暂停期间用于二次 Ctrl+C 强退。
	sigChan chan os.Signal
	// progressHold 为 true 时冻结进度条刷新，避免覆盖交互菜单。
	progressHold atomic.Bool

	// SessionFile 是当前会话文件（可能为空）。
	SessionFile string
	// SessionNew 指示恢复的会话另存为新文件。
	SessionNew bool
	// OutputHistory 是历史输出（恢复会话时使用）。
	OutputHistory []OutputEntry
	// OldSession 指示当前运行来自会话恢复。
	OldSession bool

	// stdin 用于交互菜单读取。
	stdin *bufio.Reader
	// QuietInput 供测试注入输入。
	QuietInput func() string
}

// NewController 创建控制器。
func NewController(opts *options.Options, rfs *utils.ResourceFS, ui *view.CLI) *Controller {
	return &Controller{
		Options:    opts,
		ResourceFS: rfs,
		UI:         ui,
		passedUrls: map[string]bool{},
		stdin:      bufio.NewReader(os.Stdin),
	}
}

// StartTime 返回扫描开始时间。
func (c *Controller) StartTime() string {
	if c.startTime.IsZero() {
		return time.Now().Format("2006-01-02 15:04:05")
	}
	return c.startTime.Format("2006-01-02 15:04:05")
}

// CommandString 返回脱敏后的命令行（用于报告头部）。
func (c *Controller) CommandString() string {
	return utils.RedactCommand(os.Args)
}

// Setup 初始化字典、黑名单、请求器与报告。
func (c *Controller) Setup() error {
	lineReader := func(path string) ([]string, error) {
		return c.ResourceFS.GetLines(path)
	}

	// 黑名单词表
	blacklistParams := wordlist.GenerateParams{
		Extensions:        c.Options.Extensions,
		ExcludeExtensions: c.Options.ExcludeExtensions,
		MaxSize:           c.Options.WordlistMaxSize,
		IsBlacklist:       true,
	}
	c.blacklists = wordlist.GetBlacklists(c.ResourceFS, blacklistParams, lineReader)

	// 原始请求文件：从文件还原 URL、方法、头与请求体
	if c.Options.RawFile != "" {
		if err := c.applyRawRequest(); err != nil {
			return err
		}
	} else {
		// 合并默认请求头
		merged := map[string]string{}
		for k, v := range settings.DefaultHeaders {
			merged[k] = v
		}
		for k, v := range c.Options.Headers {
			merged[k] = v
		}
		c.Options.Headers = merged
	}

	// 生成词表
	loader := wordlist.NewCategoryLoader(c.ResourceFS)
	params := wordlist.GenerateParams{
		Extensions:          c.Options.Extensions,
		ForceExtensions:     c.Options.ForceExtensions,
		OverwriteExtensions: c.Options.OverwriteExtensions,
		ExcludeExtensions:   c.Options.ExcludeExtensions,
		Prefixes:            c.Options.Prefixes,
		Suffixes:            c.Options.Suffixes,
		Lowercase:           c.Options.Lowercase,
		Uppercase:           c.Options.Uppercase,
		Capitalization:      c.Options.Capitalization,
		MaxSize:             c.Options.WordlistMaxSize,
	}
	entries, err := wordlist.Generate(c.Options.Wordlists, params, lineReader, loader)
	if err != nil {
		if limit, ok := err.(*wordlist.LimitError); ok {
			return fmt.Errorf("%s", limit)
		}
		return err
	}
	c.dictionary = wordlist.NewDictionary(entries)

	c.startTime = time.Now()
	c.directories = nil
	c.passedUrls = map[string]bool{}

	// 构建请求器
	requester, err := c.buildRequester()
	if err != nil {
		return err
	}
	c.requester = requester

	// 横幅与配置展示
	c.UI.Header(settings.Banner)
	c.UI.Config(c.Options.Extensions, c.Options.Prefixes, c.Options.Suffixes,
		c.Options.HTTPMethod, c.Options.ThreadCount, len(entries))

	// 报告管理器
	meta := report.StartInfo{
		StartTime: c.StartTime(),
		Command:   c.CommandString(),
	}
	filePaths := map[string]string{}
	for _, format := range c.Options.OutputFormats {
		switch format {
		case "mysql":
			filePaths[format] = c.Options.MysqlURL
		case "postgresql":
			filePaths[format] = c.Options.PostgresURL
		default:
			filePaths[format] = c.Options.OutputFile
		}
	}
	c.reporter = report.NewManager(c.Options.OutputFormats, meta, filePaths, c.Options.OutputTable)
	c.reporter.Warning = func(message string) {
		c.UI.NewLineSave(message, true)
	}

	if c.Options.LogFile != "" {
		c.UI.LogFile(c.Options.LogFile)
	}
	return nil
}

// applyRawRequest 从原始请求文件还原请求参数。
func (c *Controller) applyRawRequest() error {
	data, err := os.ReadFile(c.Options.RawFile)
	if err != nil {
		return err
	}
	scheme := c.Options.Scheme
	raw, err := utils.ParseRawContent(data, scheme)
	if err != nil {
		return err
	}
	c.Options.URLs = []string{raw.URL}
	c.Options.HTTPMethod = strings.ToUpper(raw.Method)
	c.Options.Headers = raw.Headers
	if raw.Body != nil {
		c.Options.Data = raw.Body
	}
	return nil
}

// buildRequester 根据选项构建 HTTP 请求器。
func (c *Controller) buildRequester() (*httpx.Requester, error) {
	var agents []string
	if c.Options.RandomAgents {
		lines, err := c.ResourceFS.GetLines("user-agents.txt")
		if err == nil {
			for _, line := range lines {
				if line != "" {
					agents = append(agents, line)
				}
			}
		}
	}

	// --ip 覆盖 DNS
	dnsOverride := map[string]string{}
	if c.Options.IP != "" {
		// 目标尚未确定，运行时通过 SetIP 补充
	}

	auth := httpx.ParseCredentials(c.Options.AuthType, c.Options.Auth)
	if auth.Type == httpx.AuthNTLM {
		return nil, fmt.Errorf("NTLM authentication is not supported in this build")
	}

	timeout := 10.0
	if c.Options.Timeout != nil {
		timeout = *c.Options.Timeout
	}
	maxRetries := 1
	if c.Options.MaxRetries != nil {
		maxRetries = *c.Options.MaxRetries
	}
	maxRate := 0
	if c.Options.MaxRate != nil {
		maxRate = *c.Options.MaxRate
	}

	return httpx.NewRequester(httpx.RequesterConfig{
		Method:          c.Options.HTTPMethod,
		Data:            c.Options.Data,
		Headers:         c.Options.Headers,
		Timeout:         timeout,
		MaxRetries:      maxRetries,
		MaxRate:         maxRate,
		Proxies:         c.Options.Proxies,
		ProxyAuth:       c.Options.ProxyAuth,
		FollowRedirects: c.Options.FollowRedirects,
		Agents:          agents,
		DNSOverride:     dnsOverride,
		CertFile:        c.Options.CertFile,
		KeyFile:         c.Options.KeyFile,
		Auth:            auth,
	})
}

// SetTarget 解析并校验目标 URL（scheme 检测、端口、凭据）。
func (c *Controller) SetTarget(target string) error {
	// 无 scheme 时补齐占位
	if !strings.Contains(target, "://") {
		scheme := c.Options.Scheme
		if scheme == "" {
			scheme = settings.Unknown
		}
		target = scheme + "://" + target
	}
	target = utils.EnsureTrailingPathSlash(target)

	parsed, err := url.Parse(target)
	if err != nil {
		return &Interrupt{Message: fmt.Sprintf("Invalid target URL: %s", target)}
	}

	if parsed.Scheme == settings.Unknown && (len(c.Options.Proxies) > 0 || c.Options.Tor) {
		return &Interrupt{Message: "Cannot auto-detect the scheme when using a proxy or Tor. " +
			"Specify http:// or https:// in the target, or use --scheme"}
	}

	c.basePath = utils.LstripOnce(parsed.Path, "/")

	// URL 内嵌凭据
	credential := ""
	if parsed.User != nil {
		credential = parsed.User.Username()
		if password, ok := parsed.User.Password(); ok {
			credential += ":" + password
		}
	}

	if parsed.Scheme != settings.Unknown && parsed.Scheme != "https" && parsed.Scheme != "http" {
		return &Interrupt{Message: fmt.Sprintf("Unsupported URI scheme: %s", parsed.Scheme)}
	}

	// 端口解析与默认值
	// scheme 未定时不能提前回填默认端口（对应 dirsearch 中 port=None 的语义），
	// 否则探测阶段会拿到 0 号端口，且无法进入"未提供端口"的回退分支。
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme != settings.Unknown {
			port = fmt.Sprintf("%d", settings.StandardPorts[parsed.Scheme])
		}
	} else if !validPort(port) {
		return &Interrupt{Message: fmt.Sprintf("Invalid port number: %s", port)}
	}

	// scheme 检测
	scheme := parsed.Scheme
	if scheme == settings.Unknown {
		if c.Options.Scheme != "" {
			scheme = c.Options.Scheme
		} else if port == "" {
			// scheme 与端口均未提供：用 443 探测协议，再按探测结果回填标准端口
			scheme = DetectScheme(parsed.Hostname(), "443", c.Options.IP)
			port = fmt.Sprintf("%d", settings.StandardPorts[scheme])
		} else {
			scheme = DetectScheme(parsed.Hostname(), port, c.Options.IP)
		}
	}

	host := parsed.Hostname()
	urlHost := host
	if strings.Contains(host, ":") {
		urlHost = "[" + host + "]"
	}
	c.targetURL = scheme + "://" + urlHost
	if port != fmt.Sprintf("%d", settings.StandardPorts[scheme]) {
		c.targetURL += ":" + port
	}
	c.targetURL += "/"

	// 配置请求器
	c.requester.SetURL(c.targetURL)
	c.requester.SetQuery(parsed.RawQuery)
	if c.Options.IP != "" {
		c.requester.Config.DNSOverride = map[string]string{
			strings.ToLower(host + ":" + port): c.Options.IP,
		}
	}
	if credential != "" {
		// 目标内嵌凭据按 basic 认证处理
		c.requester.Config.Auth = httpx.ParseCredentials("basic", credential)
	}
	return nil
}

// validPort 校验端口范围。
func validPort(port string) bool {
	n := 0
	if port == "" {
		return false
	}
	for _, ch := range port {
		if ch < '0' || ch > '9' {
			return false
		}
		n = n*10 + int(ch-'0')
	}
	return n > 0 && n < 65536
}

// DetectScheme 通过 TLS 连接探测目标协议。
func DetectScheme(host, port, connectHost string) string {
	timeout := settings.SocketTimeout
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return "http"
	}
	defer conn.Close()
	tlsConn := tls.Client(conn, &tls.Config{InsecureSkipVerify: true, ServerName: host})
	conn.SetDeadline(time.Now().Add(timeout))
	if err := tlsConn.Handshake(); err != nil {
		return "http"
	}
	return "https"
}
