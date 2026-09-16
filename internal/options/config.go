package options

import (
	"encoding/json"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"hidir/assets"
)

// ConfigFile 表示解析后的 INI 配置文件（兼容 Python configparser）。
type ConfigFile struct {
	sections map[string]map[string]string
}

// ReadConfigFile 读取并解析 INI 配置文件；文件不存在时返回空配置。
// 支持 [section]、key = value、# 与 ; 注释。
func ReadConfigFile(path string) *ConfigFile {
	data, err := os.ReadFile(path)
	if err != nil {
		return &ConfigFile{sections: map[string]map[string]string{}}
	}
	return parseConfigBytes(data)
}

// parseConfigBytes 解析 INI 配置字节流。
func parseConfigBytes(data []byte) *ConfigFile {
	cfg := &ConfigFile{sections: map[string]map[string]string{}}
	section := ""
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(rawLine, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if _, ok := cfg.sections[section]; !ok {
				cfg.sections[section] = map[string]string{}
			}
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		if section == "" {
			continue
		}
		key = strings.TrimSpace(key)
		// 行内注释：仅当 # 前有空格时才算注释（与 configparser 默认行为一致）
		if idx := strings.Index(value, " #"); idx >= 0 {
			value = value[:idx]
		}
		cfg.sections[section][key] = strings.TrimSpace(value)
	}
	return cfg
}

// get 取出原始字符串值。
func (c *ConfigFile) get(section, key string) (string, bool) {
	v, ok := c.sections[section][key]
	return v, ok
}

// safeGet 取字符串，缺失返回默认值。
func (c *ConfigFile) safeGet(section, key, def string) string {
	if v, ok := c.get(section, key); ok {
		return v
	}
	return def
}

// safeGetInt 取整型，缺失或非法返回默认值。
func (c *ConfigFile) safeGetInt(section, key string, def int) int {
	v, ok := c.get(section, key)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}

// safeGetFloat 取浮点，缺失或非法返回默认值。
func (c *ConfigFile) safeGetFloat(section, key string, def float64) float64 {
	v, ok := c.get(section, key)
	if !ok {
		return def
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return def
	}
	return f
}

// safeGetBool 取布尔值（true/false/1/0/yes/no/on/off，大小写不敏感）。
func (c *ConfigFile) safeGetBool(section, key string, def bool) bool {
	v, ok := c.get(section, key)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(v)))
	if err != nil {
		return def
	}
	return b
}

// safeGetList 取列表：JSON 数组或单个字符串（与 dirsearch 的 safe_getlist 一致）。
func (c *ConfigFile) safeGetList(section, key string, def []string) []string {
	v, ok := c.get(section, key)
	if !ok {
		return def
	}
	var list []string
	if err := json.Unmarshal([]byte(v), &list); err != nil {
		list = []string{v}
	}
	return list
}

// mergeConfig 用配置文件补齐命令行中未提供的选项，返回合并后的 Options。
// 优先级与 dirsearch 的 merge_config 一致：命令行 > 配置文件 > 默认值。
func mergeConfig(opt *Options, cfg *ConfigFile) *Options {
	// ---- General ----
	if opt.ThreadCount == 0 {
		opt.ThreadCount = cfg.safeGetInt("general", "threads", 25)
	}
	if opt.AsyncMode == nil {
		v := cfg.safeGetBool("general", "async", true)
		opt.AsyncMode = &v
	}
	if opt.FilterThreshold == nil {
		v := cfg.safeGetInt("general", "filter-threshold", 0)
		opt.FilterThreshold = &v
	}
	if opt.IncludeStatusCodesRaw == "" {
		opt.IncludeStatusCodesRaw = cfg.safeGet("general", "include-status", "")
	}
	if opt.ExcludeStatusCodesRaw == "" {
		opt.ExcludeStatusCodesRaw = cfg.safeGet("general", "exclude-status", "")
	}
	if opt.ExcludeSizesRaw == "" {
		opt.ExcludeSizesRaw = cfg.safeGet("general", "exclude-sizes", "")
	}
	if len(opt.ExcludeTextsRaw) == 0 {
		opt.ExcludeTextsRaw = cfg.safeGetList("general", "exclude-texts", nil)
	}
	if opt.ExcludeRegex == "" {
		opt.ExcludeRegex = cfg.safeGet("general", "exclude-regex", "")
	}
	if opt.ExcludeRedirect == "" {
		opt.ExcludeRedirect = cfg.safeGet("general", "exclude-redirect", "")
	}
	if opt.ExcludeResponse == "" {
		opt.ExcludeResponse = cfg.safeGet("general", "exclude-response", "")
	}
	if !opt.Recursive {
		opt.Recursive = cfg.safeGetBool("general", "recursive", false)
	}
	if !opt.DeepRecursive {
		opt.DeepRecursive = cfg.safeGetBool("general", "deep-recursive", false)
	}
	if !opt.ForceRecursive {
		opt.ForceRecursive = cfg.safeGetBool("general", "force-recursive", false)
	}
	if opt.RecursionDepth == nil {
		v := cfg.safeGetInt("general", "max-recursion-depth", 0)
		opt.RecursionDepth = &v
	}
	if opt.RecursionStatusRaw == "" {
		opt.RecursionStatusRaw = cfg.safeGet("general", "recursion-status", "100-999")
	}
	if opt.SubdirsRaw == "" {
		opt.SubdirsRaw = cfg.safeGet("general", "subdirs", "")
	}
	if opt.ExcludeSubdirsRaw == "" {
		opt.ExcludeSubdirsRaw = cfg.safeGet("general", "exclude-subdirs", "")
	}
	if opt.SkipOnStatusRaw == "" {
		opt.SkipOnStatusRaw = cfg.safeGet("general", "skip-on-status", "")
	}
	if !opt.AutoCalibration {
		opt.AutoCalibration = cfg.safeGetBool("general", "auto-calibration", false)
	}
	if opt.MatcherMode == "" {
		opt.MatcherMode = cfg.safeGet("advanced-filtering", "matcher-mode", "or")
	}
	if opt.FilterMode == "" {
		opt.FilterMode = cfg.safeGet("advanced-filtering", "filter-mode", "or")
	}
	if opt.MatchStatusRaw == "" {
		opt.MatchStatusRaw = cfg.safeGet("advanced-filtering", "match-status", "")
	}
	if opt.FilterStatusRaw == "" {
		opt.FilterStatusRaw = cfg.safeGet("advanced-filtering", "filter-status", "")
	}
	if opt.MatchSizesRaw == "" {
		opt.MatchSizesRaw = cfg.safeGet("advanced-filtering", "match-size", "")
	}
	if opt.FilterSizesRaw == "" {
		opt.FilterSizesRaw = cfg.safeGet("advanced-filtering", "filter-size", "")
	}
	if opt.MatchWordsRaw == "" {
		opt.MatchWordsRaw = cfg.safeGet("advanced-filtering", "match-words", "")
	}
	if opt.FilterWordsRaw == "" {
		opt.FilterWordsRaw = cfg.safeGet("advanced-filtering", "filter-words", "")
	}
	if opt.MatchLinesRaw == "" {
		opt.MatchLinesRaw = cfg.safeGet("advanced-filtering", "match-lines", "")
	}
	if opt.FilterLinesRaw == "" {
		opt.FilterLinesRaw = cfg.safeGet("advanced-filtering", "filter-lines", "")
	}
	if opt.MatchRegex == "" {
		opt.MatchRegex = cfg.safeGet("advanced-filtering", "match-regex", "")
	}
	if opt.FilterRegex == "" {
		opt.FilterRegex = cfg.safeGet("advanced-filtering", "filter-regex", "")
	}
	if len(opt.MatchHeadersRaw) == 0 {
		opt.MatchHeadersRaw = cfg.safeGetList("advanced-filtering", "match-header", nil)
	}
	if len(opt.FilterHeadersRaw) == 0 {
		opt.FilterHeadersRaw = cfg.safeGetList("advanced-filtering", "filter-header", nil)
	}
	if opt.MatchHeaderRegex == "" {
		opt.MatchHeaderRegex = cfg.safeGet("advanced-filtering", "match-header-regex", "")
	}
	if opt.FilterHeaderRegex == "" {
		opt.FilterHeaderRegex = cfg.safeGet("advanced-filtering", "filter-header-regex", "")
	}
	if opt.MatchTimeRaw == "" {
		opt.MatchTimeRaw = cfg.safeGet("advanced-filtering", "match-time", "")
	}
	if opt.FilterTimeRaw == "" {
		opt.FilterTimeRaw = cfg.safeGet("advanced-filtering", "filter-time", "")
	}
	if opt.MaxTime == nil {
		v := cfg.safeGetInt("general", "max-time", 0)
		opt.MaxTime = &v
	}
	if opt.TargetMaxTime == nil {
		v := cfg.safeGetInt("general", "target-max-time", 0)
		opt.TargetMaxTime = &v
	}
	if !opt.ExitOnError {
		opt.ExitOnError = cfg.safeGetBool("general", "exit-on-error", false)
	}

	// ---- Dictionary ----
	if opt.WordlistsRaw == "" {
		opt.WordlistsRaw = cfg.safeGet("dictionary", "wordlists", "")
	}
	if opt.WordlistCategories == "" {
		opt.WordlistCategories = cfg.safeGet("dictionary", "wordlist-categories", "")
	}
	if opt.WordlistBackend == "auto" && cfgHas(cfg, "dictionary", "wordlist-backend") {
		opt.WordlistBackend = cfg.safeGet("dictionary", "wordlist-backend", "auto")
	}
	if opt.WordlistMaxSize == 0 {
		opt.WordlistMaxSize = cfg.safeGetInt("dictionary", "wordlist-max-size", 500000)
	}
	if opt.ExtensionsRaw == "" {
		opt.ExtensionsRaw = cfg.safeGet("dictionary", "default-extensions", "")
	}
	if !opt.ForceExtensions {
		opt.ForceExtensions = cfg.safeGetBool("dictionary", "force-extensions", false)
	}
	if !opt.OverwriteExtensions {
		opt.OverwriteExtensions = cfg.safeGetBool("dictionary", "overwrite-extensions", false)
	}
	if opt.ExcludeExtensionsRaw == "" {
		opt.ExcludeExtensionsRaw = cfg.safeGet("dictionary", "exclude-extensions", "")
	}
	if opt.PrefixesRaw == "" {
		opt.PrefixesRaw = cfg.safeGet("dictionary", "prefixes", "")
	}
	if opt.SuffixesRaw == "" {
		opt.SuffixesRaw = cfg.safeGet("dictionary", "suffixes", "")
	}
	if !opt.Lowercase {
		opt.Lowercase = cfg.safeGetBool("dictionary", "lowercase", false)
	}
	if !opt.Uppercase {
		opt.Uppercase = cfg.safeGetBool("dictionary", "uppercase", false)
	}
	if !opt.Capitalization {
		opt.Capitalization = cfg.safeGetBool("dictionary", "capital", false)
	}

	// ---- Request ----
	if opt.HTTPMethod == "" {
		opt.HTTPMethod = cfg.safeGet("request", "http-method", "get")
	}
	if opt.RequestBackend == "" {
		opt.RequestBackend = cfg.safeGet("request", "request-backend", "python")
	}
	if len(opt.HeadersRaw) == 0 {
		opt.HeadersRaw = cfg.safeGetList("request", "headers", nil)
	}
	if opt.HeadersFile == "" {
		opt.HeadersFile = cfg.safeGet("request", "headers-file", "")
	}
	if !opt.FollowRedirects {
		opt.FollowRedirects = cfg.safeGetBool("request", "follow-redirects", false)
	}
	if !opt.RandomAgents {
		opt.RandomAgents = cfg.safeGetBool("request", "random-user-agents", false)
	}
	if opt.UserAgent == "" {
		opt.UserAgent = cfg.safeGet("request", "user-agent", "")
	}
	if opt.Cookie == "" {
		opt.Cookie = cfg.safeGet("request", "cookie", "")
	}

	// ---- Connection ----
	if opt.Delay == nil {
		v := cfg.safeGetFloat("connection", "delay", 0)
		opt.Delay = &v
	}
	if opt.Timeout == nil {
		v := cfg.safeGetFloat("connection", "timeout", 7.5)
		opt.Timeout = &v
	}
	if opt.MaxRetries == nil {
		v := cfg.safeGetInt("connection", "max-retries", 1)
		opt.MaxRetries = &v
	}
	if opt.MaxRate == nil {
		v := cfg.safeGetInt("connection", "max-rate", 0)
		opt.MaxRate = &v
	}
	if len(opt.ProxiesRaw) == 0 {
		opt.ProxiesRaw = cfg.safeGetList("connection", "proxies", nil)
	}
	if opt.ProxiesFile == "" {
		opt.ProxiesFile = cfg.safeGet("connection", "proxies-file", "")
	}
	if opt.Scheme == "" {
		s := cfg.safeGet("connection", "scheme", "")
		if s == "http" || s == "https" {
			opt.Scheme = s
		}
	}
	if opt.ReplayProxy == "" {
		opt.ReplayProxy = cfg.safeGet("connection", "replay-proxy", "")
	}
	if opt.NetworkInterface == "" {
		opt.NetworkInterface = cfg.safeGet("connection", "network-interface", "")
	}

	// ---- Advanced ----
	if !opt.Crawl {
		opt.Crawl = cfg.safeGetBool("advanced", "crawl", false)
	}
	if !opt.FindBackup {
		opt.FindBackup = cfg.safeGetBool("advanced", "find-backup", false)
	}

	// ---- View ----
	if !opt.FullURL {
		opt.FullURL = cfg.safeGetBool("view", "full-url", false)
	}
	if opt.Color == nil {
		// 命令行未使用 --no-color 时以配置文件为准（默认开启）
		v := cfg.safeGetBool("view", "color", true)
		opt.Color = &v
	}
	if !opt.Quiet {
		opt.Quiet = cfg.safeGetBool("view", "quiet-mode", false)
	}
	if !opt.DisableCLI {
		opt.DisableCLI = cfg.safeGetBool("view", "disable-cli", false)
	}
	if !opt.Verbose {
		opt.Verbose = cfg.safeGetBool("view", "verbose", false)
	}
	if !opt.RedirectsHistory {
		opt.RedirectsHistory = cfg.safeGetBool("view", "show-redirects-history", false)
	}

	// ---- Output ----
	if opt.OutputFile == "" {
		opt.OutputFile = cfg.safeGet("output", "output-file", "")
	}
	if opt.MysqlURL == "" {
		opt.MysqlURL = cfg.safeGet("output", "mysql-url", "")
	}
	if opt.PostgresURL == "" {
		opt.PostgresURL = cfg.safeGet("output", "postgres-url", "")
	}
	opt.OutputTable = cfg.safeGet("output", "output-sql-table", "")
	if opt.OutputFormatsRaw == "" {
		opt.OutputFormatsRaw = cfg.safeGet("output", "output-formats", "plain")
	}
	if opt.SaveResponse == "" {
		opt.SaveResponse = cfg.safeGet("output", "save-response", "")
	}
	if opt.SaveResponseJSONL == "" {
		opt.SaveResponseJSONL = cfg.safeGet("output", "save-response-jsonl", "")
	}
	if opt.LogFile == "" {
		opt.LogFile = cfg.safeGet("output", "log-file", "")
	}
	opt.LogFileSize = cfg.safeGetInt("output", "log-file-size", 0)

	return opt
}

// cfgHas 判断配置项是否存在。
func cfgHas(cfg *ConfigFile, section, key string) bool {
	_, ok := cfg.get(section, key)
	return ok
}

// ReadEmbeddedConfig 读取二进制内嵌的默认配置文件。
func ReadEmbeddedConfig() *ConfigFile {
	data, err := fs.ReadFile(assets.FS, assets.DBDir+"/config.ini")
	if err != nil {
		return &ConfigFile{sections: map[string]map[string]string{}}
	}
	return parseConfigBytes(data)
}
