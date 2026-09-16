package options

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"hidir/internal/filters"
	"hidir/internal/settings"
	"hidir/internal/utils"
	"hidir/internal/wordlist"
)

// Filesystem 抽象 ParseOptions 所需的文件操作，便于测试注入。
type Filesystem struct {
	// Exists 判断路径是否存在。
	Exists func(path string) bool
	// IsFile 判断路径是否为普通文件。
	IsFile func(path string) bool
	// ReadBytes 读取文件全部内容。
	ReadBytes func(path string) ([]byte, error)
	// ReadLines 按行读取文件。
	ReadLines func(path string) ([]string, error)
	// Abs 返回绝对路径。
	Abs func(path string) string
	// ReadStdin 读取标准输入内容。
	ReadStdin func() (string, error)
	// ExpandHome 展开 ~ 路径（可选）。
	ExpandHome func(path string) string
}

// OSFilesystem 返回基于真实文件系统的实现。
func OSFilesystem() *Filesystem {
	return &Filesystem{
		Exists: func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		IsFile: func(path string) bool {
			info, err := os.Stat(path)
			return err == nil && info.Mode().IsRegular()
		},
		ReadBytes: os.ReadFile,
		ReadLines: func(path string) ([]string, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			text := strings.ReplaceAll(string(data), "\r\n", "\n")
			lines := strings.Split(text, "\n")
			// 去掉末尾空行（与 Python splitlines 行为一致）
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
			return lines, nil
		},
		Abs: func(path string) string {
			abs, err := filepath.Abs(path)
			if err != nil {
				return path
			}
			return abs
		},
		ReadStdin: func() (string, error) {
			buf := make([]byte, 0, 4096)
			tmp := make([]byte, 4096)
			for {
				n, err := os.Stdin.Read(tmp)
				buf = append(buf, tmp[:n]...)
				if err != nil {
					break
				}
			}
			return string(buf), nil
		},
		ExpandHome: func(path string) string {
			if strings.HasPrefix(path, "~") {
				home, err := os.UserHomeDir()
				if err == nil {
					return filepath.Join(home, strings.TrimPrefix(path, "~"))
				}
			}
			return path
		},
	}
}

// ParseOptions 执行完整的选项解析管线：
// 命令行解析 -> 配置文件合并 -> 目标解析 -> 各项校验。
// 出错时返回 *ExitError；帮助/版本信息也通过 ExitError 返回（Code=0）。
func ParseOptions(args []string, fs *Filesystem, rfs *utils.ResourceFS) (*Options, error) {
	opt, err := ParseArguments(args)
	if err != nil {
		return nil, err
	}

	// 读取配置文件：CLI --config > HIDIR_CONFIG/DIRSEARCH_CONFIG 环境变量 >
	// 当前目录 config.ini > 可执行文件目录 config.ini > 内嵌默认配置
	configPath := opt.Config
	if configPath == "" {
		configPath = os.Getenv("HIDIR_CONFIG")
	}
	if configPath == "" {
		configPath = os.Getenv("DIRSEARCH_CONFIG")
	}
	if configPath == "" {
		configPath = "config.ini"
	}
	opt.Config = configPath
	cfg := ReadConfigFile(configPath)
	if len(cfg.sections) == 0 {
		// 依次尝试可执行文件目录与内嵌默认配置
		if exe, exeErr := os.Executable(); exeErr == nil {
			cfg = ReadConfigFile(filepath.Join(filepath.Dir(exe), "config.ini"))
		}
	}
	if len(cfg.sections) == 0 {
		cfg = ReadEmbeddedConfig()
	}
	opt = mergeConfig(opt, cfg)

	// --list-sessions / 会话恢复在目标校验之前处理（与 dirsearch 一致）
	if opt.ListSessions || opt.SessionFile != "" || opt.SessionID != "" {
		return opt, nil
	}

	opt.HTTPMethod = strings.ToUpper(opt.HTTPMethod)

	// ---- 目标解析 ----
	switch {
	case opt.URLsFile != "":
		if !checkFile(fs, opt.URLsFile) {
			return nil, fileAccessError(fs, opt.URLsFile)
		}
		lines, err := fs.ReadLines(opt.URLsFile)
		if err != nil {
			return nil, errf("%s", err)
		}
		opt.URLs = lines
	case opt.CIDR != "":
		ips, err := utils.IPRange(opt.CIDR)
		if err != nil {
			return nil, errf("%s", err)
		}
		opt.URLs = ips
	case opt.StdinURLs:
		text, err := fs.ReadStdin()
		if err != nil {
			return nil, errf("%s", err)
		}
		text = strings.ReplaceAll(text, "\r\n", "\n")
		var lines []string
		for _, line := range strings.Split(text, "\n") {
			lines = append(lines, strings.TrimSuffix(line, "\r"))
		}
		opt.URLs = lines
	case opt.RawFile != "":
		if !checkFile(fs, opt.RawFile) {
			return nil, fileAccessError(fs, opt.RawFile)
		}
	case opt.NmapReport != "":
		targets, err := utils.ParseNmapReport(opt.NmapReport)
		if err != nil {
			return nil, errf("Error while parsing Nmap report: %s", err)
		}
		opt.URLs = targets
	case len(opt.URLs) == 0 && !opt.WordlistStatus:
		return nil, errf("URL target is missing, try using -u <url>")
	}

	// 过滤注释行并去重（raw 模式除外）
	if opt.WordlistStatus && len(opt.URLs) == 0 {
		opt.URLs = []string{}
	} else if opt.RawFile == "" {
		var filtered []string
		for _, u := range opt.URLs {
			if strings.HasPrefix(u, "#") {
				continue
			}
			filtered = append(filtered, u)
		}
		opt.URLs = utils.StripAndUniquify(filtered)
	}

	if len(opt.ExtensionsRaw) == 0 {
		WarningPrinter("WARNING: No extension was specified!")
	}

	// ---- 词表解析 ----
	wordlists, err := wordlist.ResolveFiles(rfs, wordlist.ResolveInput{
		WordlistsRaw:  opt.WordlistsRaw,
		CategoriesRaw: opt.WordlistCategories,
		DefaultFile:   "dicc.txt",
	})
	if err != nil {
		return nil, errf("%s", err)
	}
	opt.Wordlists = wordlists

	if opt.ThreadCount < 1 {
		return nil, errf("Threads number must be greater than zero")
	}
	if opt.WordlistMaxSize < 1 {
		return nil, errf("--wordlist-max-size must be greater than zero")
	}

	// ---- 数值校验 ----
	if math.IsNaN(*opt.Timeout) || math.IsInf(*opt.Timeout, 0) || *opt.Timeout <= 0 {
		return nil, errf("--timeout must be finite and greater than zero")
	}
	if math.IsNaN(*opt.Delay) || math.IsInf(*opt.Delay, 0) || *opt.Delay < 0 {
		return nil, errf("--delay must be finite and zero or greater")
	}
	for _, item := range []struct {
		value  int
		option string
	}{
		{*opt.MaxRetries, "--retries"},
		{*opt.MaxRate, "--max-rate"},
		{*opt.RecursionDepth, "--max-recursion-depth"},
	} {
		if item.value < 0 {
			return nil, errf("%s must be zero or greater", item.option)
		}
	}

	if opt.WordlistBackend != "auto" && opt.WordlistBackend != "python" && opt.WordlistBackend != "native" {
		return nil, errf("--wordlist-backend must be one of: auto, python, native")
	}
	if opt.RequestBackend != "python" && opt.RequestBackend != "native" {
		return nil, errf("--request-backend must be one of: python, native")
	}

	// ---- 代理 ----
	if opt.Tor {
		opt.Proxies = append([]string{}, settings.DefaultTorProxies...)
	} else if opt.ProxiesFile != "" {
		if !checkFile(fs, opt.ProxiesFile) {
			return nil, fileAccessError(fs, opt.ProxiesFile)
		}
		lines, err := fs.ReadLines(opt.ProxiesFile)
		if err != nil {
			return nil, errf("%s", err)
		}
		opt.Proxies = lines
	}

	// ---- 请求体与证书 ----
	if opt.DataFile != "" {
		if !checkFile(fs, opt.DataFile) {
			return nil, fileAccessError(fs, opt.DataFile)
		}
		data, err := fs.ReadBytes(opt.DataFile)
		if err != nil {
			return nil, errf("%s", err)
		}
		opt.Data = data
	} else if opt.DataRaw != "" {
		opt.Data = []byte(opt.DataRaw)
	}

	if opt.CertFile != "" && !checkFile(fs, opt.CertFile) {
		return nil, fileAccessError(fs, opt.CertFile)
	}
	if opt.KeyFile != "" && !checkFile(fs, opt.KeyFile) {
		return nil, fileAccessError(fs, opt.KeyFile)
	}

	// ---- 请求头 ----
	headers := map[string]string{}
	if opt.HeadersFile != "" {
		if !checkFile(fs, opt.HeadersFile) {
			return nil, fileAccessError(fs, opt.HeadersFile)
		}
		data, err := fs.ReadBytes(opt.HeadersFile)
		if err != nil {
			return nil, errf("Error in headers file: %s", err)
		}
		parsed, perr := utils.ParseHeaders(string(data))
		if perr != nil {
			return nil, errf("Error in headers file: %s", perr)
		}
		for k, v := range parsed {
			headers[k] = v
		}
	}
	if len(opt.HeadersRaw) > 0 {
		parsed, perr := utils.ParseHeaders(strings.Join(opt.HeadersRaw, "\n"))
		if perr != nil {
			return nil, errf("Invalid headers")
		}
		for k, v := range parsed {
			headers[k] = v
		}
	}
	opt.Headers = headers
	if opt.UserAgent != "" {
		opt.Headers["user-agent"] = opt.UserAgent
	}
	if opt.Cookie != "" {
		opt.Headers["cookie"] = opt.Cookie
	}

	// ---- 其余原始字段的拷贝 ----
	var excludeTexts []string
	for _, text := range opt.ExcludeTextsRaw {
		if text != "" {
			excludeTexts = append(excludeTexts, text)
		}
	}
	opt.ExcludeTexts = excludeTexts
	var matchHeaders []string
	for _, header := range opt.MatchHeadersRaw {
		if header != "" {
			matchHeaders = append(matchHeaders, header)
		}
	}
	opt.MatchHeaders = matchHeaders
	var filterHeaders []string
	for _, header := range opt.FilterHeadersRaw {
		if header != "" {
			filterHeaders = append(filterHeaders, header)
		}
	}
	opt.FilterHeaders = filterHeaders
	if len(opt.ProxiesRaw) > 0 {
		opt.Proxies = opt.ProxiesRaw
	}

	// ---- 状态码解析 ----
	for _, item := range []struct {
		raw    string
		target *IntSet
	}{
		{opt.IncludeStatusCodesRaw, &opt.IncludeStatus},
		{opt.ExcludeStatusCodesRaw, &opt.ExcludeStatus},
		{opt.RecursionStatusRaw, &opt.RecursionStatus},
		{opt.SkipOnStatusRaw, &opt.SkipOnStatus},
		{opt.MatchStatusRaw, &opt.MatchStatus},
		{opt.FilterStatusRaw, &opt.FilterStatus},
	} {
		set, perr := ParseStatusCodes(item.raw)
		if perr != nil {
			return nil, errf("%s", perr)
		}
		*item.target = set
	}

	// ---- 高级过滤解析 ----
	if opt.MatchSizes, err = parseRanges(opt.MatchSizesRaw, "--match-size"); err != nil {
		return nil, err
	}
	if opt.FilterSizes, err = parseRanges(opt.FilterSizesRaw, "--filter-size"); err != nil {
		return nil, err
	}
	if opt.MatchWords, err = parseRanges(opt.MatchWordsRaw, "--match-words"); err != nil {
		return nil, err
	}
	if opt.FilterWords, err = parseRanges(opt.FilterWordsRaw, "--filter-words"); err != nil {
		return nil, err
	}
	if opt.MatchLines, err = parseRanges(opt.MatchLinesRaw, "--match-lines"); err != nil {
		return nil, err
	}
	if opt.FilterLines, err = parseRanges(opt.FilterLinesRaw, "--filter-lines"); err != nil {
		return nil, err
	}
	if opt.MatchTime, err = parseTimes(opt.MatchTimeRaw, "--match-time"); err != nil {
		return nil, err
	}
	if opt.FilterTime, err = parseTimes(opt.FilterTimeRaw, "--filter-time"); err != nil {
		return nil, err
	}
	if opt.MatcherMode != "and" && opt.MatcherMode != "or" {
		return nil, errf("--matcher-mode must be either 'and' or 'or'")
	}
	if opt.FilterMode != "and" && opt.FilterMode != "or" {
		return nil, errf("--filter-mode must be either 'and' or 'or'")
	}
	for _, item := range []struct {
		pattern string
		option  string
	}{
		{opt.ExcludeRegex, "--exclude-regex"},
		{opt.ExcludeRedirect, "--exclude-redirect"},
		{opt.MatchRegex, "--match-regex"},
		{opt.FilterRegex, "--filter-regex"},
		{opt.MatchHeaderRegex, "--match-header-regex"},
		{opt.FilterHeaderRegex, "--filter-header-regex"},
	} {
		if err := filters.ValidateRegex(item.pattern, item.option); err != nil {
			return nil, errf("%s", err)
		}
	}

	// ---- 前缀/后缀/子目录 ----
	opt.Prefixes = utils.SplitCSV(opt.PrefixesRaw)
	opt.Suffixes = utils.SplitCSV(opt.SuffixesRaw)
	opt.Subdirs = normalizeSubdirs(opt.SubdirsRaw)
	opt.ExcludeSubdirs = normalizeSubdirs(opt.ExcludeSubdirsRaw)

	// ---- 响应大小 ----
	opt.ExcludeSizes, err = filters.ParseSizeList(opt.ExcludeSizesRaw)
	if err != nil {
		return nil, errf("--exclude-sizes: %s", err)
	}
	if opt.MinResponseSize, err = filters.ParseSize(opt.MinResponseSizeRaw); err != nil {
		return nil, errf("--min-response-size: %s", err)
	}
	if opt.MaxResponseSize, err = filters.ParseSize(opt.MaxResponseSizeRaw); err != nil {
		return nil, errf("--max-response-size: %s", err)
	}

	// ---- 扩展名 ----
	if opt.ExtensionsRaw == "*" {
		opt.Extensions = append([]string{}, commonExtensions...)
	} else {
		var extensions []string
		for _, ext := range utils.SplitCSV(opt.ExtensionsRaw) {
			extensions = append(extensions, strings.TrimLeft(ext, "."))
		}
		opt.Extensions = extensions
	}
	var excludeExtensions []string
	for _, ext := range utils.SplitCSV(opt.ExcludeExtensionsRaw) {
		excludeExtensions = append(excludeExtensions, strings.TrimLeft(ext, "."))
	}
	opt.ExcludeExtensions = excludeExtensions

	// ---- 认证 ----
	if opt.Auth != "" && opt.AuthType == "" {
		return nil, errf("Please select the authentication type with --auth-type")
	} else if opt.AuthType != "" && opt.Auth == "" {
		return nil, errf("No authentication credential found")
	} else if opt.Auth != "" && !contains(authTypes, opt.AuthType) {
		return nil, errf("'%s' is not in available authentication types: %s",
			opt.AuthType, strings.Join(authTypes, ", "))
	}

	// 扩展名与排除扩展名不能交集
	extensionSet := map[string]bool{}
	for _, ext := range opt.Extensions {
		extensionSet[ext] = true
	}
	for _, ext := range opt.ExcludeExtensions {
		if extensionSet[ext] {
			return nil, errf("Exclude extension list can not contain any extension that has already in the extension list")
		}
	}

	// ---- 输出 ----
	var formats []string
	for _, format := range strings.Split(opt.OutputFormatsRaw, ",") {
		format = strings.TrimSpace(format)
		if format != "" {
			formats = append(formats, format)
		}
	}
	opt.OutputFormats = formats

	var invalid []string
	for _, format := range opt.OutputFormats {
		if !contains(outputFormatsAll, format) {
			invalid = append(invalid, format)
		}
	}
	if len(invalid) > 0 {
		sort.Strings(invalid)
		return nil, errf("Invalid output format(s): %s", strings.Join(invalid, ", "))
	}

	if len(opt.OutputFormats) == 0 && opt.OutputFile != "" {
		return nil, errf("Please provide output formats (use '-O')")
	}

	// 多格式共用一个输出文件时必须使用 {format}/{extension} 变量
	hasFormatVar := strings.Contains(opt.OutputFile, "{format}")
	hasExtensionVar := strings.Contains(opt.OutputFile, "{extension}")
	if opt.OutputFile != "" && !hasFormatVar && len(opt.OutputFormats) > 1 {
		plainAndSimple := false
		count := map[string]bool{}
		for _, f := range opt.OutputFormats {
			count[f] = true
		}
		if count["plain"] && count["simple"] {
			plainAndSimple = true
		}
		if !hasExtensionVar || plainAndSimple {
			return nil, errf("Found at least 2 output formats sharing the same output file, " +
				"make sure you use '{format}' and '{extension} variables in your output file")
		}
	}

	if opt.MysqlURL != "" {
		opt.OutputFormats = append(opt.OutputFormats, "mysql")
	}
	if opt.PostgresURL != "" {
		opt.OutputFormats = append(opt.OutputFormats, "postgresql")
	}

	if opt.LogFile != "" {
		opt.LogFile = fs.Abs(opt.LogFile)
	}
	if opt.OutputFile != "" {
		opt.OutputFile = fs.Abs(opt.OutputFile)
	}
	if opt.SaveResponse != "" {
		opt.SaveResponse = fs.Abs(opt.SaveResponse)
	}
	if opt.SaveResponseJSONL != "" {
		opt.SaveResponseJSONL = fs.Abs(opt.SaveResponseJSONL)
	}

	return opt, nil
}

// checkFile 校验文件存在、是普通文件且可读。
func checkFile(fs *Filesystem, path string) bool {
	return fs.Exists(path) && fs.IsFile(path)
}

// fileAccessError 生成与 dirsearch 一致的文件访问错误。
func fileAccessError(fs *Filesystem, path string) *ExitError {
	if !fs.Exists(path) {
		return errf("%s does not exist", path)
	}
	if !fs.IsFile(path) {
		return errf("%s is not a file", path)
	}
	return errf("%s cannot be read", path)
}

// parseRanges 解析数值范围并包装错误信息。
func parseRanges(value, optionName string) ([]filters.NumericRange, error) {
	ranges, err := filters.ParseNumericRanges(value)
	if err != nil {
		return nil, errf("%s: %s", optionName, err)
	}
	return ranges, nil
}

// parseTimes 解析时间过滤并包装错误信息。
func parseTimes(value, optionName string) ([]filters.TimeFilter, error) {
	times, err := filters.ParseTimeFilters(value)
	if err != nil {
		return nil, errf("%s: %s", optionName, err)
	}
	return times, nil
}

// normalizeSubdirs 规范化子目录列表，语义与 dirsearch 一致：
// 先补齐尾部斜杠再去重去空白，最后去掉头部斜杠。
// 空字符串会产生一个 "" 条目（对应根目录扫描任务）。
func normalizeSubdirs(raw string) []string {
	var subdirs []string
	for _, subdir := range strings.Split(raw, ",") {
		subdir = strings.TrimSpace(subdir)
		if !strings.HasSuffix(subdir, "/") {
			subdir += "/"
		}
		subdirs = append(subdirs, subdir)
	}
	subdirs = utils.StripAndUniquify(subdirs)
	for i, subdir := range subdirs {
		subdirs[i] = utils.LstripOnce(subdir, "/")
	}
	return subdirs
}

// contains 列表包含判断。
func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// WarningPrinter 是警告输出函数（默认输出到 stderr，测试可替换）。
var WarningPrinter = func(message string) {
	os.Stderr.WriteString(message + "\n")
}
