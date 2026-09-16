// Package options 定义 HiDir 的全部命令行参数、配置文件合并与校验逻辑，
// 与 dirsearch 的 lib/parse/cmdline.py、lib/core/options.py 保持兼容。
package options

import (
	"fmt"

	"hidir/internal/filters"
)

// IntSet 是状态码集合。
type IntSet map[int]struct{}

// NewIntSet 从整数列表构建集合。
func NewIntSet(values ...int) IntSet {
	s := make(IntSet, len(values))
	for _, v := range values {
		s[v] = struct{}{}
	}
	return s
}

// Contains 判断值是否在集合中。
func (s IntSet) Contains(v int) bool {
	_, ok := s[v]
	return ok
}

// Add 加入一个值。
func (s IntSet) Add(v int) {
	s[v] = struct{}{}
}

// Len 返回集合大小。
func (s IntSet) Len() int {
	return len(s)
}

// Values 返回集合中的全部值（顺序不定，仅用于测试与展示）。
func (s IntSet) Values() []int {
	out := make([]int, 0, len(s))
	for v := range s {
		out = append(out, v)
	}
	return out
}

// Options 汇总一次扫描所需的全部配置。
// 字段名与 dirsearch 的 options 字典键保持一致，便于对照。
// 带 Raw 后缀的字段保存命令行/配置文件中的原始字符串，
// 在 ParseOptions 校验阶段被解析为对应的结构化字段。
type Options struct {
	// ---- Mandatory ----
	URLs         []string
	URLsFile     string
	StdinURLs    bool
	CIDR         string
	RawFile      string
	NmapReport   string
	SessionFile  string
	SessionID    string
	ListSessions bool
	SessionsDir  string
	Config       string

	// ---- Dictionary ----
	Wordlists            []string // 解析后的词表文件列表
	WordlistsRaw         string   // 原始 -w 参数
	WordlistCategories   string
	WordlistBackend      string
	WordlistStatus       bool
	WordlistMaxSize      int
	Extensions           []string
	ExtensionsRaw        string
	ForceExtensions      bool
	OverwriteExtensions  bool
	ExcludeExtensions    []string
	ExcludeExtensionsRaw string
	Prefixes             []string
	PrefixesRaw          string
	Suffixes             []string
	SuffixesRaw          string
	Uppercase            bool
	Lowercase            bool
	Capitalization       bool

	// ---- General ----
	ThreadCount        int
	AsyncMode          *bool // nil 表示未显式指定
	Recursive          bool
	DeepRecursive      bool
	ForceRecursive     bool
	RecursionDepth     *int
	RecursionStatusRaw string
	RecursionStatus    IntSet
	FilterThreshold    *int
	Subdirs            []string
	SubdirsRaw         string
	ExcludeSubdirs     []string
	ExcludeSubdirsRaw  string

	IncludeStatusCodesRaw string
	IncludeStatus         IntSet
	ExcludeStatusCodesRaw string
	ExcludeStatus         IntSet
	ExcludeSizesRaw       string
	ExcludeSizes          map[int64]struct{}
	ExcludeTextsRaw       []string
	ExcludeTexts          []string
	ExcludeRegex          string
	ExcludeRedirect       string
	ExcludeResponse       string
	SkipOnStatusRaw       string
	SkipOnStatus          IntSet

	MinResponseSizeRaw string
	MinResponseSize    int64
	MaxResponseSizeRaw string
	MaxResponseSize    int64

	MaxTime       *int
	TargetMaxTime *int
	ExitOnError   bool

	// ---- Advanced Filtering ----
	AutoCalibration   bool
	MatcherMode       string
	FilterMode        string
	MatchStatusRaw    string
	MatchStatus       IntSet
	FilterStatusRaw   string
	FilterStatus      IntSet
	MatchSizesRaw     string
	MatchSizes        []filters.NumericRange
	FilterSizesRaw    string
	FilterSizes       []filters.NumericRange
	MatchWordsRaw     string
	MatchWords        []filters.NumericRange
	FilterWordsRaw    string
	FilterWords       []filters.NumericRange
	MatchLinesRaw     string
	MatchLines        []filters.NumericRange
	FilterLinesRaw    string
	FilterLines       []filters.NumericRange
	MatchRegex        string
	FilterRegex       string
	MatchHeadersRaw   []string
	MatchHeaders      []string
	FilterHeadersRaw  []string
	FilterHeaders     []string
	MatchHeaderRegex  string
	FilterHeaderRegex string
	MatchTimeRaw      string
	MatchTime         []filters.TimeFilter
	FilterTimeRaw     string
	FilterTime        []filters.TimeFilter

	// ---- Request ----
	HTTPMethod      string
	RequestBackend  string
	Data            []byte
	DataRaw         string
	DataFile        string
	Headers         map[string]string
	HeadersRaw      []string
	HeadersFile     string
	FollowRedirects bool
	RandomAgents    bool
	Auth            string
	AuthType        string
	CertFile        string
	KeyFile         string
	UserAgent       string
	Cookie          string

	// ---- Connection ----
	Timeout          *float64
	Delay            *float64
	Proxies          []string
	ProxiesRaw       []string
	ProxiesFile      string
	ProxyAuth        string
	ReplayProxy      string
	Tor              bool
	Scheme           string
	MaxRate          *int
	MaxRetries       *int
	NetworkInterface string
	IP               string

	// ---- Advanced ----
	Crawl      bool
	FindBackup bool

	// ---- View ----
	FullURL          bool
	RedirectsHistory bool
	Color            *bool
	Quiet            bool
	DisableCLI       bool
	Verbose          bool

	// ---- Output ----
	OutputFile        string
	OutputTable       string
	OutputFormatsRaw  string
	OutputFormats     []string
	MysqlURL          string
	PostgresURL       string
	SaveResponse      string
	SaveResponseJSONL string
	LogFile           string
	LogFileSize       int
}

// NewDefaults 返回与 dirsearch 默认值一致的 Options。
// 注意：MatcherMode/FilterMode/HTTPMethod/RequestBackend/ThreadCount/
// WordlistMaxSize 留空（对应 dirsearch 中的 None），
// 由 mergeConfig 依据配置文件填写默认值（与 dirsearch 的合并语义一致）。
func NewDefaults() *Options {
	return &Options{
		WordlistBackend: "auto",
		WordlistMaxSize: 0,
		ThreadCount:     0,
		RecursionStatus: IntSet{},
		IncludeStatus:   IntSet{},
		ExcludeStatus:   IntSet{},
		ExcludeSizes:    map[int64]struct{}{},
		SkipOnStatus:    IntSet{},
		MatchStatus:     IntSet{},
		FilterStatus:    IntSet{},
		Headers:         map[string]string{},
	}
}

// ExitError 表示解析/校验失败需要退出程序。
type ExitError struct {
	// Message 是要打印给用户的信息。
	Message string
	// Code 是进程退出码。
	Code int
	// ToStderr 为 true 时输出到 stderr（模拟 optparse 行为）。
	ToStderr bool
}

// Error 实现 error 接口。
func (e *ExitError) Error() string {
	return e.Message
}

// errf 构造一个退出码为 1 的 stdout 错误（模拟 print + sys.exit(1)）。
func errf(format string, args ...any) *ExitError {
	return &ExitError{Message: fmt.Sprintf(format, args...), Code: 1}
}

// cliErrf 构造一个 stderr 错误（模拟 parser.error）。
func cliErrf(format string, args ...any) *ExitError {
	return &ExitError{Message: "error: " + fmt.Sprintf(format, args...), Code: 2, ToStderr: true}
}
