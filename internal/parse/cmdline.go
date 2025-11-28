package parse

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/yourusername/hidir/internal/common"
)

// Options 存储命令行参数

type Options struct {
	// 必选参数
	URLs        []string
	URLFile     string
	StdinURLs   bool
	CIDR        string
	RawFile     string
	SessionFile string
	Config      string

	// 字典设置
	Wordlists           string
	Extensions          string
	ForceExtensions     bool
	OverwriteExtensions bool
	ExcludeExtensions   string
	RemoveExtensions    bool
	Prefixes            string
	Suffixes            string
	Uppercase           bool
	Lowercase           bool
	Capitalization      bool

	// 通用设置
	ThreadCount          int
	Recursive            bool
	DeepRecursive        bool
	ForceRecursive       bool
	RecursionDepth       int
	RecursionStatusCodes string
	Subdirs              string
	ExcludeSubdirs       string
	IncludeStatusCodes   string
	ExcludeStatusCodes   string
	ExcludeSizes         string
	ExcludeTexts         []string
	ExcludeRegex         string
	ExcludeRedirect      string
	ExcludeResponse      string
	SkipOnStatus         string
	MinimumResponseSize  int
	MaximumResponseSize  int
	MaxTime              int
	ExitOnError          bool

	// 请求设置
	HTTPMethod      string
	Data            string
	DataFile        string
	Headers         []string
	HeaderFile      string
	FollowRedirects bool
	RandomAgents    bool
	Auth            string
	AuthType        string
	CertFile        string
	KeyFile         string
	UserAgent       string
	Cookie          string

	// 连接设置
	Timeout     float64
	Delay       float64
	Proxies     []string
	ProxyFile   string
	ProxyAuth   string
	ReplayProxy string
	Tor         bool
	Scheme      string
	MaxRate     int
	MaxRetries  int
	IP          string

	// 高级设置
	Crawl bool

	// 视图设置
	FullURL          bool
	RedirectsHistory bool
	Color            bool
	Quiet            bool

	// 输出设置
	OutputFile   string
	OutputFormat string
	LogFile      string
}

// ParseArguments 解析命令行参数
func ParseArguments() *Options {
	opt := &Options{}

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-u|--url] target [-e|--extensions] extensions [options]\n", os.Args[0])
		pflag.PrintDefaults()
	}

	// 必选参数
	mandatory := pflag.NewFlagSet("Mandatory", pflag.ExitOnError)
	mandatory.StringSliceVarP(&opt.URLs, "url", "u", nil, "Target URL(s), can use multiple flags")
	mandatory.StringVarP(&opt.URLFile, "url-file", "l", "", "URL list file")
	mandatory.BoolVar(&opt.StdinURLs, "stdin", false, "Read URL(s) from STDIN")
	mandatory.StringVar(&opt.CIDR, "cidr", "", "Target CIDR")
	mandatory.StringVar(&opt.RawFile, "raw", "", "Load raw HTTP request from file (use `--scheme` flag to set the scheme)")
	mandatory.StringVar(&opt.SessionFile, "session", "", "Session file")
	mandatory.StringVar(&opt.Config, "config", "config.ini", "Full path to config file")

	// 字典设置
	dictionary := pflag.NewFlagSet("Dictionary Settings", pflag.ExitOnError)
	dictionary.StringVarP(&opt.Wordlists, "wordlists", "w", "", "Customize wordlists (separated by commas)")
	dictionary.StringVarP(&opt.Extensions, "extensions", "e", "", "Extension list separated by commas (e.g. php,asp)")
	dictionary.BoolVarP(&opt.ForceExtensions, "force-extensions", "f", false, "Add extensions to the end of every wordlist entry")
	dictionary.BoolVarP(&opt.OverwriteExtensions, "overwrite-extensions", "O", false, "Overwrite other extensions in the wordlist with your extensions")
	dictionary.StringVar(&opt.ExcludeExtensions, "exclude-extensions", "", "Exclude extension list separated by commas (e.g. asp,jsp)")
	dictionary.BoolVar(&opt.RemoveExtensions, "remove-extensions", false, "Remove extensions in all paths (e.g. admin.php -> admin)")
	dictionary.StringVar(&opt.Prefixes, "prefixes", "", "Add custom prefixes to all wordlist entries (separated by commas)")
	dictionary.StringVar(&opt.Suffixes, "suffixes", "", "Add custom suffixes to all wordlist entries, ignore directories (separated by commas)")
	dictionary.BoolVarP(&opt.Uppercase, "uppercase", "U", false, "Uppercase wordlist")
	dictionary.BoolVarP(&opt.Lowercase, "lowercase", "L", false, "Lowercase wordlist")
	dictionary.BoolVarP(&opt.Capitalization, "capital", "C", false, "Capital wordlist")

	// 通用设置
	general := pflag.NewFlagSet("General Settings", pflag.ExitOnError)
	general.IntVarP(&opt.ThreadCount, "threads", "t", 0, "Number of threads")
	general.BoolVarP(&opt.Recursive, "recursive", "r", false, "Brute-force recursively")
	general.BoolVar(&opt.DeepRecursive, "deep-recursive", false, "Perform recursive scan on every directory depth")
	general.BoolVar(&opt.ForceRecursive, "force-recursive", false, "Do recursive brute-force for every found path")
	general.IntVarP(&opt.RecursionDepth, "max-recursion-depth", "R", 0, "Maximum recursion depth")
	general.StringVar(&opt.RecursionStatusCodes, "recursion-status", "", "Valid status codes to perform recursive scan")
	general.StringVar(&opt.Subdirs, "subdirs", "", "Scan sub-directories of the given URL[s]")
	general.StringVar(&opt.ExcludeSubdirs, "exclude-subdirs", "", "Exclude the following subdirectories during recursive scan")
	general.StringVarP(&opt.IncludeStatusCodes, "include-status", "i", "", "Include status codes")
	general.StringVarP(&opt.ExcludeStatusCodes, "exclude-status", "x", "", "Exclude status codes")
	general.StringVar(&opt.ExcludeSizes, "exclude-sizes", "", "Exclude responses by sizes")
	general.StringSliceVar(&opt.ExcludeTexts, "exclude-text", nil, "Exclude responses by text")
	general.StringVar(&opt.ExcludeRegex, "exclude-regex", "", "Exclude responses by regular expression")
	general.StringVar(&opt.ExcludeRedirect, "exclude-redirect", "", "Exclude responses if this regex matches redirect URL")
	general.StringVar(&opt.ExcludeResponse, "exclude-response", "", "Exclude responses similar to response of this page")
	general.StringVar(&opt.SkipOnStatus, "skip-on-status", "", "Skip target whenever hit one of these status codes")
	general.IntVar(&opt.MinimumResponseSize, "min-response-size", 0, "Minimum response length")
	general.IntVar(&opt.MaximumResponseSize, "max-response-size", 0, "Maximum response length")
	general.IntVar(&opt.MaxTime, "max-time", 0, "Maximum runtime for the scan")
	general.BoolVar(&opt.ExitOnError, "exit-on-error", false, "Exit whenever an error occurs")

	// 请求设置
	request := pflag.NewFlagSet("Request Settings", pflag.ExitOnError)
	request.StringVarP(&opt.HTTPMethod, "http-method", "m", "", "HTTP method")
	request.StringVarP(&opt.Data, "data", "d", "", "HTTP request data")
	request.StringVar(&opt.DataFile, "data-file", "", "File contains HTTP request data")
	request.StringSliceVarP(&opt.Headers, "header", "H", nil, "HTTP request header")
	request.StringVar(&opt.HeaderFile, "header-file", "", "File contains HTTP request headers")
	request.BoolVarP(&opt.FollowRedirects, "follow-redirects", "F", false, "Follow HTTP redirects")
	request.BoolVar(&opt.RandomAgents, "random-agent", false, "Choose a random User-Agent for each request")
	request.StringVar(&opt.Auth, "auth", "", "Authentication credential")
	request.StringVar(&opt.AuthType, "auth-type", "", fmt.Sprintf("Authentication type (%s)", common.AUTHENTICATION_TYPES))
	request.StringVar(&opt.CertFile, "cert-file", "", "File contains client-side certificate")
	request.StringVar(&opt.KeyFile, "key-file", "", "File contains client-side certificate private key")
	request.StringVar(&opt.UserAgent, "user-agent", "", "User-Agent")
	request.StringVar(&opt.Cookie, "cookie", "", "Cookie")

	// 连接设置
	connection := pflag.NewFlagSet("Connection Settings", pflag.ExitOnError)
	connection.Float64Var(&opt.Timeout, "timeout", 0, "Connection timeout")
	connection.Float64Var(&opt.Delay, "delay", 0, "Delay between requests")
	connection.StringSliceVar(&opt.Proxies, "proxy", nil, "Proxy URL")
	connection.StringVar(&opt.ProxyFile, "proxy-file", "", "File contains proxy servers")
	connection.StringVar(&opt.ProxyAuth, "proxy-auth", "", "Proxy authentication credential")
	connection.StringVar(&opt.ReplayProxy, "replay-proxy", "", "Proxy to replay with found paths")
	connection.BoolVar(&opt.Tor, "tor", false, "Use Tor network as proxy")
	connection.StringVar(&opt.Scheme, "scheme", "", "Scheme for raw request")
	connection.IntVar(&opt.MaxRate, "max-rate", 0, "Max requests per second")
	connection.IntVar(&opt.MaxRetries, "retries", 0, "Number of retries for failed requests")
	connection.StringVar(&opt.IP, "ip", "", "Server IP address")

	// 高级设置
	advanced := pflag.NewFlagSet("Advanced Settings", pflag.ExitOnError)
	advanced.BoolVar(&opt.Crawl, "crawl", false, "Crawl for new paths in responses")

	// 视图设置
	view := pflag.NewFlagSet("View Settings", pflag.ExitOnError)
	view.BoolVar(&opt.FullURL, "full-url", false, "Full URLs in the output")
	view.BoolVar(&opt.RedirectsHistory, "redirects-history", false, "Show redirects history")
	view.BoolVar(&opt.Color, "color", true, "Colored output")
	view.BoolVarP(&opt.Quiet, "quiet-mode", "q", false, "Quiet mode")

	// 输出设置
	output := pflag.NewFlagSet("Output Settings", pflag.ExitOnError)
	output.StringVarP(&opt.OutputFile, "output", "o", "", "Output file")
	output.StringVar(&opt.OutputFormat, "format", "", "Report format")
	output.StringVar(&opt.LogFile, "log", "", "Log file")

	// 添加所有标志到主标志集
	pflag.CommandLine.AddFlagSet(mandatory)
	pflag.CommandLine.AddFlagSet(dictionary)
	pflag.CommandLine.AddFlagSet(general)
	pflag.CommandLine.AddFlagSet(request)
	pflag.CommandLine.AddFlagSet(connection)
	pflag.CommandLine.AddFlagSet(advanced)
	pflag.CommandLine.AddFlagSet(view)
	pflag.CommandLine.AddFlagSet(output)

	// 解析命令行参数
	pflag.Parse()

	return opt
}
