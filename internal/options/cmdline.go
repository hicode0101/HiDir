package options

import (
	"fmt"
	"strings"
)

// optSpec 描述一个命令行选项的规格。
type optSpec struct {
	Short   string // 短选项（可为空）
	Long    string // 长选项
	Dest    string // 目标字段名（对应 dirsearch 的 dest）
	HasArg  bool   // 是否带参数
	Metavar string
	Help    string
	Group   string
}

// groups 是帮助输出的分组顺序。
var groups = []string{
	"Mandatory",
	"Dictionary Settings",
	"General Settings",
	"Advanced Filtering",
	"Request Settings",
	"Connection Settings",
	"Advanced Settings",
	"View Settings",
	"Output Settings",
}

// optionSpecs 定义全部命令行选项（与 dirsearch 的 parse_arguments 一致）。
var optionSpecs = []optSpec{
	// Mandatory
	{Short: "-u", Long: "--url", Dest: "urls", HasArg: true, Metavar: "URL",
		Help: "Target URL(s), can use multiple flags", Group: "Mandatory"},
	{Short: "-l", Long: "--urls-file", Dest: "urls_file", HasArg: true, Metavar: "PATH",
		Help: "URL list file", Group: "Mandatory"},
	{Long: "--stdin", Dest: "stdin_urls",
		Help: "Read URL(s) from STDIN", Group: "Mandatory"},
	{Long: "--cidr", Dest: "cidr", HasArg: true,
		Help: "Target CIDR", Group: "Mandatory"},
	{Long: "--raw", Dest: "raw_file", HasArg: true, Metavar: "PATH",
		Help: "Load raw HTTP request from file (use '--scheme' flag to set the scheme)", Group: "Mandatory"},
	{Long: "--nmap-report", Dest: "nmap_report", HasArg: true, Metavar: "PATH",
		Help: "Load targets from nmap report (Ensure the inclusion of the -sV flag during nmap scan for comprehensive results)", Group: "Mandatory"},
	{Short: "-s", Long: "--session", Dest: "session_file", HasArg: true,
		Help: "Session file", Group: "Mandatory"},
	{Long: "--session-id", Dest: "session_id", HasArg: true, Metavar: "ID",
		Help: "Load session by numeric id (use --list-sessions to see ids)", Group: "Mandatory"},
	{Long: "--config", Dest: "config", HasArg: true, Metavar: "PATH",
		Help: "Path to configuration file (Default: 'DIRSEARCH_CONFIG' environment variable, otherwise 'config.ini')", Group: "Mandatory"},

	// Dictionary Settings
	{Short: "-w", Long: "--wordlists", Dest: "wordlists", HasArg: true,
		Help: "Wordlist files or directories contain wordlists (separated by commas)", Group: "Dictionary Settings"},
	{Long: "--wordlist-categories", Dest: "wordlist_categories", HasArg: true,
		Help: "Comma-separated wordlist category names (e.g. common,conf,web). Use 'all' to include all bundled categories", Group: "Dictionary Settings"},
	{Long: "--wordlist-backend", Dest: "wordlist_backend", HasArg: true, Metavar: "BACKEND",
		Help: "Wordlist generation backend: auto, python, native (default: auto)", Group: "Dictionary Settings"},
	{Long: "--wordlist-status", Dest: "wordlist_status",
		Help: "Show resolved wordlist files and generated entry count, then exit", Group: "Dictionary Settings"},
	{Long: "--wordlist-max-size", Dest: "wordlist_max_size", HasArg: true, Metavar: "COUNT",
		Help: "Maximum generated wordlist entries before aborting (default: 500000)", Group: "Dictionary Settings"},
	{Short: "-e", Long: "--extensions", Dest: "extensions", HasArg: true,
		Help: "Extension list, separated by commas (e.g. php,asp); use quoted '*' for common extensions", Group: "Dictionary Settings"},
	{Short: "-f", Long: "--force-extensions", Dest: "force_extensions",
		Help: "Add extensions to the end of every wordlist entry. By default dirsearch only replaces the %EXT% keyword with extensions", Group: "Dictionary Settings"},
	{Long: "--overwrite-extensions", Dest: "overwrite_extensions",
		Help: "Overwrite other extensions in the wordlist with your extensions (selected via `-e`)", Group: "Dictionary Settings"},
	{Long: "--exclude-extensions", Dest: "exclude_extensions", HasArg: true, Metavar: "EXTENSIONS",
		Help: "Exclude extension list, separated by commas (e.g. asp,jsp)", Group: "Dictionary Settings"},
	{Long: "--prefixes", Dest: "prefixes", HasArg: true,
		Help: "Add custom prefixes to all wordlist entries (separated by commas)", Group: "Dictionary Settings"},
	{Long: "--suffixes", Dest: "suffixes", HasArg: true,
		Help: "Add custom suffixes to all wordlist entries, ignore directories (separated by commas)", Group: "Dictionary Settings"},
	{Short: "-U", Long: "--uppercase", Dest: "uppercase",
		Help: "Uppercase wordlist", Group: "Dictionary Settings"},
	{Short: "-L", Long: "--lowercase", Dest: "lowercase",
		Help: "Lowercase wordlist", Group: "Dictionary Settings"},
	{Short: "-C", Long: "--capital", Dest: "capital",
		Help: "Capital wordlist", Group: "Dictionary Settings"},

	// General Settings
	{Short: "-t", Long: "--threads", Dest: "thread_count", HasArg: true, Metavar: "THREADS",
		Help: "Number of threads", Group: "General Settings"},
	{Long: "--list-sessions", Dest: "list_sessions",
		Help: "List resumable sessions and exit", Group: "General Settings"},
	{Long: "--sessions-dir", Dest: "sessions_dir", HasArg: true, Metavar: "PATH",
		Help: "Directory to search for resumable sessions (default: HiDir path /sessions, or $HOME/.hidir/sessions when bundled)", Group: "General Settings"},
	{Short: "-a", Long: "--async", Dest: "async_mode",
		Help: "Enable asynchronous mode", Group: "General Settings"},
	{Long: "--sync, --no-async", Dest: "no_async",
		Help: "Use synchronous Python mode", Group: "General Settings"},
	{Short: "-r", Long: "--recursive", Dest: "recursive",
		Help: "Brute-force recursively", Group: "General Settings"},
	{Long: "--deep-recursive", Dest: "deep_recursive",
		Help: "Perform recursive scan on every directory depth (e.g. api/users -> api/)", Group: "General Settings"},
	{Long: "--force-recursive", Dest: "force_recursive",
		Help: "Do recursive brute-force for every found path, not only directories", Group: "General Settings"},
	{Short: "-R", Long: "--max-recursion-depth", Dest: "recursion_depth", HasArg: true, Metavar: "DEPTH",
		Help: "Maximum recursion depth (0 means unlimited)", Group: "General Settings"},
	{Long: "--recursion-status", Dest: "recursion_status_codes", HasArg: true, Metavar: "CODES",
		Help: "Valid status codes to perform recursive scan, support ranges (separated by commas)", Group: "General Settings"},
	{Long: "--filter-threshold", Dest: "filter_threshold", HasArg: true, Metavar: "THRESHOLD",
		Help: "Maximum number of results with duplicate responses before getting filtered out", Group: "General Settings"},
	{Long: "--subdirs", Dest: "subdirs", HasArg: true, Metavar: "SUBDIRS",
		Help: "Scan sub-directories of the given URL[s] (separated by commas)", Group: "General Settings"},
	{Long: "--exclude-subdirs", Dest: "exclude_subdirs", HasArg: true, Metavar: "SUBDIRS",
		Help: "Exclude the following subdirectories during recursive scan (separated by commas)", Group: "General Settings"},
	{Short: "-i", Long: "--include-status", Dest: "include_status_codes", HasArg: true, Metavar: "CODES",
		Help: "Include status codes, separated by commas, support ranges (e.g. 200,300-399)", Group: "General Settings"},
	{Short: "-x", Long: "--exclude-status", Dest: "exclude_status_codes", HasArg: true, Metavar: "CODES",
		Help: "Exclude status codes, separated by commas, support ranges (e.g. 301,500-599)", Group: "General Settings"},
	{Long: "--exclude-sizes", Dest: "exclude_sizes", HasArg: true, Metavar: "SIZES",
		Help: "Exclude responses by sizes, separated by commas (e.g. 0,0B,4KB)", Group: "General Settings"},
	{Long: "--exclude-text", Dest: "exclude_texts", HasArg: true, Metavar: "TEXTS",
		Help: "Exclude responses by text, can use multiple flags", Group: "General Settings"},
	{Long: "--exclude-regex", Dest: "exclude_regex", HasArg: true, Metavar: "REGEX",
		Help: "Exclude responses by regular expression", Group: "General Settings"},
	{Long: "--exclude-redirect", Dest: "exclude_redirect", HasArg: true, Metavar: "STRING",
		Help: "Exclude responses if this regex (or text) matches redirect URL (e.g. '/index.html')", Group: "General Settings"},
	{Long: "--exclude-response", Dest: "exclude_response", HasArg: true, Metavar: "PATH",
		Help: "Exclude responses similar to response of this page, path as input (e.g. 404.html)", Group: "General Settings"},
	{Long: "--skip-on-status", Dest: "skip_on_status", HasArg: true, Metavar: "CODES",
		Help: "Skip target whenever hit one of these status codes, separated by commas, support ranges", Group: "General Settings"},
	{Long: "--min-response-size", Dest: "minimum_response_size", HasArg: true, Metavar: "LENGTH",
		Help: "Minimum response length (e.g. 1024,1KB)", Group: "General Settings"},
	{Long: "--max-response-size", Dest: "maximum_response_size", HasArg: true, Metavar: "LENGTH",
		Help: "Maximum response length (e.g. 1024,1KB)", Group: "General Settings"},
	{Long: "--max-time", Dest: "max_time", HasArg: true, Metavar: "SECONDS",
		Help: "Maximum runtime for the scan", Group: "General Settings"},
	{Long: "--target-max-time", Dest: "target_max_time", HasArg: true, Metavar: "SECONDS",
		Help: "Maximum runtime for a target", Group: "General Settings"},
	{Long: "--exit-on-error", Dest: "exit_on_error",
		Help: "Exit whenever an error occurs", Group: "General Settings"},

	// Advanced Filtering
	{Long: "--auto-calibration", Dest: "auto_calibration",
		Help: "Force extra wildcard calibration from the beginning", Group: "Advanced Filtering"},
	{Long: "--matcher-mode, --mmode", Dest: "matcher_mode", HasArg: true, Metavar: "MODE",
		Help: "Advanced matcher operator: and, or", Group: "Advanced Filtering"},
	{Long: "--filter-mode, --fmode", Dest: "filter_mode", HasArg: true, Metavar: "MODE",
		Help: "Advanced filter operator: and, or", Group: "Advanced Filtering"},
	{Long: "--match-status, --mc", Dest: "match_status_codes", HasArg: true, Metavar: "CODES",
		Help: "Advanced matcher for status codes, separated by commas, support ranges", Group: "Advanced Filtering"},
	{Long: "--filter-status, --fc", Dest: "filter_status_codes", HasArg: true, Metavar: "CODES",
		Help: "Advanced filter for status codes, separated by commas, support ranges", Group: "Advanced Filtering"},
	{Long: "--match-size, --ms", Dest: "match_sizes", HasArg: true, Metavar: "SIZES",
		Help: "Advanced matcher for response length, separated by commas, support ranges", Group: "Advanced Filtering"},
	{Long: "--filter-size, --fs", Dest: "filter_sizes", HasArg: true, Metavar: "SIZES",
		Help: "Advanced filter for response length, separated by commas, support ranges", Group: "Advanced Filtering"},
	{Long: "--match-words, --mw", Dest: "match_words", HasArg: true, Metavar: "WORDS",
		Help: "Advanced matcher for response word count, separated by commas, support ranges", Group: "Advanced Filtering"},
	{Long: "--filter-words, --fw", Dest: "filter_words", HasArg: true, Metavar: "WORDS",
		Help: "Advanced filter for response word count, separated by commas, support ranges", Group: "Advanced Filtering"},
	{Long: "--match-lines, --ml", Dest: "match_lines", HasArg: true, Metavar: "LINES",
		Help: "Advanced matcher for response line count, separated by commas, support ranges", Group: "Advanced Filtering"},
	{Long: "--filter-lines, --fl", Dest: "filter_lines", HasArg: true, Metavar: "LINES",
		Help: "Advanced filter for response line count, separated by commas, support ranges", Group: "Advanced Filtering"},
	{Long: "--match-regex, --mr", Dest: "match_regex", HasArg: true, Metavar: "REGEX",
		Help: "Advanced matcher for response body regular expression", Group: "Advanced Filtering"},
	{Long: "--filter-regex, --fr", Dest: "filter_regex", HasArg: true, Metavar: "REGEX",
		Help: "Advanced filter for response body regular expression", Group: "Advanced Filtering"},
	{Long: "--match-header", Dest: "match_headers", HasArg: true, Metavar: "TEXT",
		Help: "Advanced matcher for response headers by text, can use multiple flags", Group: "Advanced Filtering"},
	{Long: "--filter-header", Dest: "filter_headers", HasArg: true, Metavar: "TEXT",
		Help: "Advanced filter for response headers by text, can use multiple flags", Group: "Advanced Filtering"},
	{Long: "--match-header-regex", Dest: "match_header_regex", HasArg: true, Metavar: "REGEX",
		Help: "Advanced matcher for response headers regular expression", Group: "Advanced Filtering"},
	{Long: "--filter-header-regex", Dest: "filter_header_regex", HasArg: true, Metavar: "REGEX",
		Help: "Advanced filter for response headers regular expression", Group: "Advanced Filtering"},
	{Long: "--match-time, --mt", Dest: "match_time", HasArg: true, Metavar: "TIME",
		Help: "Advanced matcher for elapsed milliseconds, e.g. >100 or <100", Group: "Advanced Filtering"},
	{Long: "--filter-time, --ft", Dest: "filter_time", HasArg: true, Metavar: "TIME",
		Help: "Advanced filter for elapsed milliseconds, e.g. >100 or <100", Group: "Advanced Filtering"},

	// Request Settings
	{Short: "-m", Long: "--http-method", Dest: "http_method", HasArg: true, Metavar: "METHOD",
		Help: "HTTP method (default: GET)", Group: "Request Settings"},
	{Long: "--request-backend", Dest: "request_backend", HasArg: true, Metavar: "BACKEND",
		Help: "Request backend: python, native (default: python)", Group: "Request Settings"},
	{Short: "-d", Long: "--data", Dest: "data", HasArg: true,
		Help: "HTTP request data", Group: "Request Settings"},
	{Long: "--data-file", Dest: "data_file", HasArg: true, Metavar: "PATH",
		Help: "Read request body from file without encoding or newline conversion", Group: "Request Settings"},
	{Short: "-H", Long: "--header", Dest: "headers", HasArg: true,
		Help: "HTTP request header, can use multiple flags", Group: "Request Settings"},
	{Long: "--headers-file", Dest: "headers_file", HasArg: true, Metavar: "PATH",
		Help: "File contains HTTP request headers", Group: "Request Settings"},
	{Short: "-F", Long: "--follow-redirects", Dest: "follow_redirects",
		Help: "Follow HTTP redirects", Group: "Request Settings"},
	{Long: "--random-agent", Dest: "random_agents",
		Help: "Choose a random User-Agent for each request", Group: "Request Settings"},
	{Long: "--auth", Dest: "auth", HasArg: true, Metavar: "CREDENTIAL",
		Help: "Authentication credential (e.g. user:password or bearer token)", Group: "Request Settings"},
	{Long: "--auth-type", Dest: "auth_type", HasArg: true, Metavar: "TYPE",
		Help: "Authentication type (basic, digest, bearer, ntlm, jwt)", Group: "Request Settings"},
	{Long: "--cert-file", Dest: "cert_file", HasArg: true, Metavar: "PATH",
		Help: "File contains client-side certificate", Group: "Request Settings"},
	{Long: "--key-file", Dest: "key_file", HasArg: true, Metavar: "PATH",
		Help: "File contains client-side certificate private key (unencrypted)", Group: "Request Settings"},
	{Long: "--user-agent", Dest: "user_agent", HasArg: true,
		Help: "", Group: "Request Settings"},
	{Long: "--cookie", Dest: "cookie", HasArg: true,
		Help: "", Group: "Request Settings"},

	// Connection Settings
	{Long: "--timeout", Dest: "timeout", HasArg: true,
		Help: "Connection timeout in seconds (greater than 0)", Group: "Connection Settings"},
	{Long: "--delay", Dest: "delay", HasArg: true,
		Help: "Delay between requests in seconds (0 or greater)", Group: "Connection Settings"},
	{Short: "-p", Long: "--proxy", Dest: "proxies", HasArg: true, Metavar: "PROXY",
		Help: "Proxy URL (HTTP/SOCKS), can use multiple flags", Group: "Connection Settings"},
	{Long: "--proxies-file", Dest: "proxies_file", HasArg: true, Metavar: "PATH",
		Help: "File contains proxy servers", Group: "Connection Settings"},
	{Long: "--proxy-auth", Dest: "proxy_auth", HasArg: true, Metavar: "CREDENTIAL",
		Help: "Proxy authentication credential", Group: "Connection Settings"},
	{Long: "--replay-proxy", Dest: "replay_proxy", HasArg: true, Metavar: "PROXY",
		Help: "Proxy to replay with found paths", Group: "Connection Settings"},
	{Long: "--tor", Dest: "tor",
		Help: "Use Tor network as proxy", Group: "Connection Settings"},
	{Long: "--scheme", Dest: "scheme", HasArg: true, Metavar: "SCHEME",
		Help: "Scheme for raw request or if there is no scheme in the URL (Default: auto-detect)", Group: "Connection Settings"},
	{Long: "--max-rate", Dest: "max_rate", HasArg: true, Metavar: "RATE",
		Help: "Maximum requests per second (0 means unlimited)", Group: "Connection Settings"},
	{Short: "", Long: "--retries", Dest: "max_retries", HasArg: true, Metavar: "RETRIES",
		Help: "Number of retries for failed requests (0 or greater)", Group: "Connection Settings"},
	{Long: "--ip", Dest: "ip", HasArg: true,
		Help: "Server IP address", Group: "Connection Settings"},
	{Long: "--interface", Dest: "network_interface", HasArg: true,
		Help: "Network interface to use", Group: "Connection Settings"},

	// Advanced Settings
	{Long: "--crawl", Dest: "crawl",
		Help: "Crawl for new paths in responses", Group: "Advanced Settings"},
	{Long: "--find-backup", Dest: "find_backup",
		Help: "Look for backups of discovered files", Group: "Advanced Settings"},

	// View Settings
	{Long: "--full-url", Dest: "full_url",
		Help: "Full URLs in the output (enabled automatically in quiet mode)", Group: "View Settings"},
	{Long: "--redirects-history", Dest: "redirects_history",
		Help: "Show redirects history", Group: "View Settings"},
	{Long: "--no-color", Dest: "color",
		Help: "No colored output", Group: "View Settings"},
	{Short: "-q", Long: "--quiet-mode", Dest: "quiet",
		Help: "Quiet mode", Group: "View Settings"},
	{Long: "--disable-cli", Dest: "disable_cli",
		Help: "Turn off command-line output", Group: "View Settings"},
	{Short: "-v", Long: "--verbose", Dest: "verbose",
		Help: "Show verbose output with response time and content type", Group: "View Settings"},

	// Output Settings
	{Short: "-O", Long: "--output-formats", Dest: "output_formats", HasArg: true, Metavar: "FORMAT",
		Help: "Report formats, separated by commas (Available: simple, plain, json, xml, md, csv, html, sqlite)", Group: "Output Settings"},
	{Short: "-o", Long: "--output-file", Dest: "output_file", HasArg: true, Metavar: "PATH",
		Help: "Output file location", Group: "Output Settings"},
	{Long: "--mysql-url", Dest: "mysql_url", HasArg: true, Metavar: "URL",
		Help: "Database URL for MySQL output (Format: mysql://[username:password@]host[:port]/database-name)", Group: "Output Settings"},
	{Long: "--postgres-url", Dest: "postgres_url", HasArg: true, Metavar: "URL",
		Help: "Database URL for PostgreSQL output (Format: postgres://[username:password@]host[:port]/database-name)", Group: "Output Settings"},
	{Long: "--save-response", Dest: "save_response", HasArg: true, Metavar: "PATH",
		Help: "Save matched response bodies into a directory", Group: "Output Settings"},
	{Long: "--save-response-jsonl", Dest: "save_response_jsonl", HasArg: true, Metavar: "PATH",
		Help: "Append matched responses to a JSONL file", Group: "Output Settings"},
	{Long: "--log", Dest: "log_file", HasArg: true, Metavar: "PATH",
		Help: "Log file", Group: "Output Settings"},
}

// parsedArgs 保存命令行解析的中间结果（未合并 config）。
type parsedArgs struct {
	opts  *Options
	color *bool // --no-color 显式置 false
}

// specIndex 按选项名（含短/长/别名）索引规格。
func buildSpecIndex() map[string]*optSpec {
	index := make(map[string]*optSpec)
	for i := range optionSpecs {
		spec := &optionSpecs[i]
		for _, name := range strings.Split(spec.Long, ", ") {
			index[name] = spec
		}
		if spec.Short != "" {
			index[spec.Short] = spec
		}
	}
	return index
}

var specIndex = buildSpecIndex()

// ParseArguments 解析命令行参数（不含程序名），返回未合并配置的 Options。
// 出错时返回 *ExitError。
func ParseArguments(args []string) (*Options, error) {
	opts := NewDefaults()
	state := &parsedArgs{opts: opts}

	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "--help-all":
			return nil, &ExitError{Message: HelpAll, Code: 0}
		case arg == "-hh":
			return nil, &ExitError{Message: HelpAll, Code: 0}
		case arg == "-h" || arg == "--help":
			return nil, &ExitError{Message: HelpCommon, Code: 0}
		case arg == "--version":
			return nil, &ExitError{Message: fmt.Sprintf("HiDir v%s", Version), Code: 0}
		case strings.HasPrefix(arg, "-"):
			consumed, err := state.applyOption(args, i)
			if err != nil {
				return nil, err
			}
			i += consumed
		default:
			return nil, cliErrf(
				"unexpected positional argument(s); quote shell wildcards such as '*' when passing option values")
		}
	}
	return opts, nil
}

// applyOption 处理 args[i] 上的选项，返回消费的参数个数。
func (p *parsedArgs) applyOption(args []string, i int) (int, error) {
	arg := args[i]

	var name, inlineValue string
	hasInline := false
	if strings.HasPrefix(arg, "--") {
		// 支持 --option=value 形式（含 --mmode=and）
		if eq := strings.Index(arg, "="); eq >= 0 {
			name = arg[:eq]
			inlineValue = arg[eq+1:]
			hasInline = true
		} else {
			name = arg
		}
	} else {
		// 短选项：仅识别精确匹配（-u、-hh 已提前处理）
		name = arg[:2]
	}

	spec, ok := specIndex[name]
	if !ok {
		return 0, cliErrf("no such option: %s", name)
	}

	// 处理别名（如 --sync/--no-async 同一个 dest）
	longName := spec.Long
	value := inlineValue
	consumed := 1

	if spec.HasArg {
		if !hasInline {
			if i+1 >= len(args) {
				return 0, cliErrf("option %s requires argument", name)
			}
			value = args[i+1]
			consumed = 2
		}
	} else if hasInline {
		return 0, cliErrf("--%s option does not take a value", strings.TrimLeft(name, "-"))
	}

	if err := p.set(spec, value, longName); err != nil {
		return 0, err
	}
	return consumed, nil
}

// appendValue 向追加型字段写入。
func appendValue(dst []string, value string) []string {
	return append(dst, value)
}

// set 将解析出的值写入 Options 对应字段。
func (p *parsedArgs) set(spec *optSpec, value, longName string) error {
	o := p.opts
	_ = longName
	switch spec.Dest {
	case "urls":
		o.URLs = appendValue(o.URLs, value)
	case "urls_file":
		o.URLsFile = value
	case "stdin_urls":
		o.StdinURLs = true
	case "cidr":
		o.CIDR = value
	case "raw_file":
		o.RawFile = value
	case "nmap_report":
		o.NmapReport = value
	case "session_file":
		o.SessionFile = value
	case "session_id":
		o.SessionID = value
	case "config":
		o.Config = value
	case "wordlists":
		o.WordlistsRaw = value
	case "wordlist_categories":
		o.WordlistCategories = value
	case "wordlist_backend":
		o.WordlistBackend = value
	case "wordlist_status":
		o.WordlistStatus = true
	case "wordlist_max_size":
		n, err := parseIntArg(value)
		if err != nil {
			return cliErrf("option --wordlist-max-size: invalid integer value: '%s'", value)
		}
		o.WordlistMaxSize = n
	case "extensions":
		o.ExtensionsRaw = value
	case "force_extensions":
		o.ForceExtensions = true
	case "overwrite_extensions":
		o.OverwriteExtensions = true
	case "exclude_extensions":
		o.ExcludeExtensionsRaw = value
	case "prefixes":
		o.PrefixesRaw = value
	case "suffixes":
		o.SuffixesRaw = value
	case "uppercase":
		o.Uppercase = true
	case "lowercase":
		o.Lowercase = true
	case "capital":
		o.Capitalization = true
	case "thread_count":
		n, err := parseIntArg(value)
		if err != nil {
			return cliErrf("option -t/--threads: invalid integer value: '%s'", value)
		}
		o.ThreadCount = n
	case "list_sessions":
		o.ListSessions = true
	case "sessions_dir":
		o.SessionsDir = value
	case "async_mode":
		t := true
		o.AsyncMode = &t
	case "no_async":
		f := false
		o.AsyncMode = &f
	case "recursive":
		o.Recursive = true
	case "deep_recursive":
		o.DeepRecursive = true
	case "force_recursive":
		o.ForceRecursive = true
	case "recursion_depth":
		n, err := parseIntArg(value)
		if err != nil {
			return cliErrf("option -R/--max-recursion-depth: invalid integer value: '%s'", value)
		}
		o.RecursionDepth = &n
	case "recursion_status_codes":
		o.RecursionStatusRaw = value
	case "filter_threshold":
		n, err := parseIntArg(value)
		if err != nil {
			return cliErrf("option --filter-threshold: invalid integer value: '%s'", value)
		}
		o.FilterThreshold = &n
	case "subdirs":
		o.SubdirsRaw = value
	case "exclude_subdirs":
		o.ExcludeSubdirsRaw = value
	case "include_status_codes":
		o.IncludeStatusCodesRaw = value
	case "exclude_status_codes":
		o.ExcludeStatusCodesRaw = value
	case "exclude_sizes":
		o.ExcludeSizesRaw = value
	case "exclude_texts":
		o.ExcludeTextsRaw = appendValue(o.ExcludeTextsRaw, value)
	case "exclude_regex":
		o.ExcludeRegex = value
	case "exclude_redirect":
		o.ExcludeRedirect = value
	case "exclude_response":
		o.ExcludeResponse = value
	case "skip_on_status":
		o.SkipOnStatusRaw = value
	case "minimum_response_size":
		o.MinResponseSizeRaw = value
	case "maximum_response_size":
		o.MaxResponseSizeRaw = value
	case "max_time":
		n, err := parseIntArg(value)
		if err != nil {
			return cliErrf("option --max-time: invalid integer value: '%s'", value)
		}
		o.MaxTime = &n
	case "target_max_time":
		n, err := parseIntArg(value)
		if err != nil {
			return cliErrf("option --target-max-time: invalid integer value: '%s'", value)
		}
		o.TargetMaxTime = &n
	case "exit_on_error":
		o.ExitOnError = true
	case "auto_calibration":
		o.AutoCalibration = true
	case "matcher_mode":
		o.MatcherMode = value
	case "filter_mode":
		o.FilterMode = value
	case "match_status_codes":
		o.MatchStatusRaw = value
	case "filter_status_codes":
		o.FilterStatusRaw = value
	case "match_sizes":
		o.MatchSizesRaw = value
	case "filter_sizes":
		o.FilterSizesRaw = value
	case "match_words":
		o.MatchWordsRaw = value
	case "filter_words":
		o.FilterWordsRaw = value
	case "match_lines":
		o.MatchLinesRaw = value
	case "filter_lines":
		o.FilterLinesRaw = value
	case "match_regex":
		o.MatchRegex = value
	case "filter_regex":
		o.FilterRegex = value
	case "match_headers":
		o.MatchHeadersRaw = appendValue(o.MatchHeadersRaw, value)
	case "filter_headers":
		o.FilterHeadersRaw = appendValue(o.FilterHeadersRaw, value)
	case "match_header_regex":
		o.MatchHeaderRegex = value
	case "filter_header_regex":
		o.FilterHeaderRegex = value
	case "match_time":
		o.MatchTimeRaw = value
	case "filter_time":
		o.FilterTimeRaw = value
	case "http_method":
		o.HTTPMethod = value
	case "request_backend":
		o.RequestBackend = value
	case "data":
		o.DataRaw = value
	case "data_file":
		o.DataFile = value
	case "headers":
		o.HeadersRaw = appendValue(o.HeadersRaw, value)
	case "headers_file":
		o.HeadersFile = value
	case "follow_redirects":
		o.FollowRedirects = true
	case "random_agents":
		o.RandomAgents = true
	case "auth":
		o.Auth = value
	case "auth_type":
		o.AuthType = value
	case "cert_file":
		o.CertFile = value
	case "key_file":
		o.KeyFile = value
	case "user_agent":
		o.UserAgent = value
	case "cookie":
		o.Cookie = value
	case "timeout":
		f, err := parseFloatArg(value)
		if err != nil {
			return cliErrf("option --timeout: invalid floating-point value: '%s'", value)
		}
		o.Timeout = &f
	case "delay":
		f, err := parseFloatArg(value)
		if err != nil {
			return cliErrf("option --delay: invalid floating-point value: '%s'", value)
		}
		o.Delay = &f
	case "proxies":
		o.ProxiesRaw = appendValue(o.ProxiesRaw, value)
	case "proxies_file":
		o.ProxiesFile = value
	case "proxy_auth":
		o.ProxyAuth = value
	case "replay_proxy":
		o.ReplayProxy = value
	case "tor":
		o.Tor = true
	case "scheme":
		o.Scheme = value
	case "max_rate":
		n, err := parseIntArg(value)
		if err != nil {
			return cliErrf("option --max-rate: invalid integer value: '%s'", value)
		}
		o.MaxRate = &n
	case "max_retries":
		n, err := parseIntArg(value)
		if err != nil {
			return cliErrf("option --retries: invalid integer value: '%s'", value)
		}
		o.MaxRetries = &n
	case "ip":
		o.IP = value
	case "network_interface":
		o.NetworkInterface = value
	case "crawl":
		o.Crawl = true
	case "find_backup":
		o.FindBackup = true
	case "full_url":
		o.FullURL = true
	case "redirects_history":
		o.RedirectsHistory = true
	case "color":
		f := false
		p.color = &f
		o.Color = &f
	case "quiet":
		o.Quiet = true
	case "disable_cli":
		o.DisableCLI = true
	case "verbose":
		o.Verbose = true
	case "output_formats":
		o.OutputFormatsRaw = value
	case "output_file":
		o.OutputFile = value
	case "mysql_url":
		o.MysqlURL = value
	case "postgres_url":
		o.PostgresURL = value
	case "save_response":
		o.SaveResponse = value
	case "save_response_jsonl":
		o.SaveResponseJSONL = value
	case "log_file":
		o.LogFile = value
	default:
		return cliErrf("no such option: %s", spec.Long)
	}
	return nil
}

// parseIntArg 解析整型参数。
func parseIntArg(value string) (int, error) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// parseFloatArg 解析浮点参数。
func parseFloatArg(value string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(strings.TrimSpace(value), "%g", &f)
	if err != nil {
		return 0, err
	}
	return f, nil
}

// ParseStatusCodes 解析 "200,300-399" 形式的状态码集合。
// 供 options 校验与测试使用；非法值返回错误。
func ParseStatusCodes(value string) (IntSet, error) {
	set := IntSet{}
	if value == "" {
		return set, nil
	}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			start, end, found := strings.Cut(part, "-")
			startN, err1 := parseIntArg(start)
			endN, err2 := parseIntArg(end)
			if !found || err1 != nil || err2 != nil {
				return nil, fmt.Errorf("Invalid status code or status code range: %s", part)
			}
			for i := startN; i <= endN; i++ {
				set.Add(i)
			}
			continue
		}
		n, err := parseIntArg(part)
		if err != nil {
			return nil, fmt.Errorf("Invalid status code or status code range: %s", part)
		}
		set.Add(n)
	}
	return set, nil
}
