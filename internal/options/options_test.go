package options

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hidir/internal/utils"
)

// testFS 构造一个内存文件系统用于解析测试。
func testFS(files map[string]string) *Filesystem {
	return &Filesystem{
		Exists: func(path string) bool {
			_, ok := files[path]
			return ok
		},
		IsFile: func(path string) bool {
			_, ok := files[path]
			return ok
		},
		ReadBytes: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return []byte(data), nil
			}
			return nil, &fsError{}
		},
		ReadLines: func(path string) ([]string, error) {
			if data, ok := files[path]; ok {
				var lines []string
				for _, line := range strings.Split(data, "\n") {
					lines = append(lines, strings.TrimSuffix(line, "\r"))
				}
				if len(lines) > 0 && lines[len(lines)-1] == "" {
					lines = lines[:len(lines)-1]
				}
				return lines, nil
			}
			return nil, &fsError{}
		},
		Abs: func(path string) string { return "/abs/" + path },
		ReadStdin: func() (string, error) {
			if data, ok := files["-stdin-"]; ok {
				return data, nil
			}
			return "", nil
		},
		ExpandHome: func(path string) string { return path },
	}
}

type fsError struct{}

func (e *fsError) Error() string { return "not found" }

// parse 辅助：解析参数并忽略内嵌配置影响（使用空配置文件）。
func mustParse(t *testing.T, args []string) *Options {
	t.Helper()
	opts, err := ParseOptions(args, testFS(nil), utils.NewResourceFS())
	if err != nil {
		t.Fatalf("ParseOptions(%v) error: %v", args, err)
	}
	return opts
}

// ---- 帮助与版本 ----

func TestHelpAndVersion(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		_, err := ParseOptions([]string{flag}, testFS(nil), utils.NewResourceFS())
		exitErr, ok := err.(*ExitError)
		if !ok || exitErr.Code != 0 {
			t.Errorf("%s should produce help exit 0, got %v", flag, err)
		}
		if !containsStr(exitErr.Message, "Usage: hidir") {
			t.Errorf("help should contain usage")
		}
	}
	for _, flag := range []string{"--help-all", "-hh"} {
		_, err := ParseOptions([]string{flag}, testFS(nil), utils.NewResourceFS())
		exitErr := err.(*ExitError)
		if exitErr.Code != 0 {
			t.Errorf("%s exit code = %d", flag, exitErr.Code)
		}
		if !containsStr(exitErr.Message, "Advanced Filtering") {
			t.Errorf("--help-all should include advanced filtering")
		}
	}
	_, err := ParseOptions([]string{"--version"}, testFS(nil), utils.NewResourceFS())
	exitErr := err.(*ExitError)
	if !containsStr(exitErr.Message, "HiDir v") {
		t.Errorf("version output = %q", exitErr.Message)
	}
}

// ---- 目标输入 ----

func TestTargetMissing(t *testing.T) {
	_, err := ParseOptions(nil, testFS(nil), utils.NewResourceFS())
	exitErr, ok := err.(*ExitError)
	if !ok || exitErr.Message != "URL target is missing, try using -u <url>" {
		t.Errorf("missing target error = %v", err)
	}
}

func TestTargetMultipleURLs(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-u", "http://b.com/", "--url", "http://c.com/"})
	if len(opts.URLs) != 3 {
		t.Errorf("URLs = %v", opts.URLs)
	}
}

func TestTargetURLsFile(t *testing.T) {
	fs := testFS(map[string]string{
		"urls.txt": "http://a.com/\n# comment\nhttp://b.com/\n\n",
	})
	opts, err := ParseOptions([]string{"-l", "urls.txt"}, fs, utils.NewResourceFS())
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.URLs) != 2 || opts.URLs[0] != "http://a.com/" {
		t.Errorf("URLs = %v (comments/blanks should be stripped)", opts.URLs)
	}
}

func TestTargetURLsFileMissing(t *testing.T) {
	_, err := ParseOptions([]string{"-l", "missing.txt"}, testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "does not exist") {
		t.Errorf("missing urls file error = %v", err)
	}
}

func TestTargetStdin(t *testing.T) {
	fs := testFS(map[string]string{"-stdin-": "http://a.com/\nhttp://b.com/\n"})
	opts, err := ParseOptions([]string{"--stdin"}, fs, utils.NewResourceFS())
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.URLs) != 2 {
		t.Errorf("stdin URLs = %v", opts.URLs)
	}
}

func TestTargetCIDR(t *testing.T) {
	opts, err := ParseOptions([]string{"--cidr", "127.0.0.0/30"}, testFS(nil), utils.NewResourceFS())
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.URLs) != 4 {
		t.Errorf("CIDR URLs = %v", opts.URLs)
	}
}

func TestTargetDuplicateRemoval(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-u", " http://a.com/ "})
	if len(opts.URLs) != 1 {
		t.Errorf("duplicates should be removed: %v", opts.URLs)
	}
}

// ---- 字典参数 ----

func TestWordlistDefault(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-e", "php"})
	if len(opts.Wordlists) != 1 || opts.Wordlists[0] != "dicc.txt" {
		t.Errorf("default wordlist = %v", opts.Wordlists)
	}
}

func TestWordlistMultiple(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-w", "dicc.txt,dicc.txt", "-e", "php"})
	if len(opts.Wordlists) != 1 {
		t.Errorf("duplicated wordlist should be deduped: %v", opts.Wordlists)
	}
}

func TestWordlistCategories(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--wordlist-categories", "common,web"})
	found := map[string]bool{}
	for _, path := range opts.Wordlists {
		found[path] = true
	}
	if !found["categories/common.txt"] || !found["categories/web.txt"] {
		t.Errorf("category wordlists = %v", opts.Wordlists)
	}
}

func TestWordlistCategoriesAll(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--wordlist-categories", "all"})
	if len(opts.Wordlists) < 20 {
		t.Errorf("'all' should include every category: %d", len(opts.Wordlists))
	}
}

func TestWordlistCategoriesUnknown(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "--wordlist-categories", "nope"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "Unknown wordlist categories: nope") {
		t.Errorf("unknown category error = %v", err)
	}
}

func TestWordlistCategoryWildcard(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--wordlist-categories", "php/*"})
	if len(opts.Wordlists) != 9 {
		t.Errorf("php/* wordlists = %v", opts.Wordlists)
	}
}

func TestWordlistMissing(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "-w", "no-such-file.txt"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "does not exist") {
		t.Errorf("missing wordlist error = %v", err)
	}
}

func TestWordlistMaxSize(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--wordlist-max-size", "1000"})
	if opts.WordlistMaxSize != 1000 {
		t.Errorf("wordlist max size = %d", opts.WordlistMaxSize)
	}
	// 0 是假值：回退到默认 500000（与 dirsearch 一致）
	opts = mustParse(t, []string{"-u", "http://a.com/", "--wordlist-max-size", "0"})
	if opts.WordlistMaxSize != 500000 {
		t.Errorf("max size 0 should fall back to 500000, got %d", opts.WordlistMaxSize)
	}
}

func TestWordlistStatus(t *testing.T) {
	opts, err := ParseOptions([]string{"--wordlist-status"}, testFS(nil), utils.NewResourceFS())
	if err != nil {
		t.Fatal(err)
	}
	if !opts.WordlistStatus {
		t.Error("wordlist status flag")
	}
}

// ---- 扩展名参数 ----

func TestExtensions(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-e", "php,asp"})
	if len(opts.Extensions) != 2 || opts.Extensions[0] != "php" {
		t.Errorf("extensions = %v", opts.Extensions)
	}
}

func TestExtensionsStar(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-e", "*"})
	if len(opts.Extensions) != 11 || opts.Extensions[0] != "php" {
		t.Errorf("* extensions = %v", opts.Extensions)
	}
}

func TestExtensionsDotsStripped(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-e", ".php,.asp"})
	if opts.Extensions[0] != "php" || opts.Extensions[1] != "asp" {
		t.Errorf("dots should be stripped: %v", opts.Extensions)
	}
}

func TestExcludeExtensions(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-e", "php", "--exclude-extensions", "old,bak"})
	if len(opts.ExcludeExtensions) != 2 {
		t.Errorf("exclude extensions = %v", opts.ExcludeExtensions)
	}
}

func TestExcludeExtensionsConflict(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "-e", "php", "--exclude-extensions", "php"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "can not contain any extension") {
		t.Errorf("conflict error = %v", err)
	}
}

func TestPrefixSuffix(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-e", "php", "--prefixes", ".,admin", "--suffixes", "~,.bak"})
	if len(opts.Prefixes) != 2 || opts.Prefixes[0] != "." {
		t.Errorf("prefixes = %v", opts.Prefixes)
	}
	if len(opts.Suffixes) != 2 || opts.Suffixes[1] != ".bak" {
		t.Errorf("suffixes = %v", opts.Suffixes)
	}
}

func TestCaseTransforms(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-L"})
	if !opts.Lowercase || opts.Uppercase || opts.Capitalization {
		t.Errorf("lowercase flags = %v/%v/%v", opts.Lowercase, opts.Uppercase, opts.Capitalization)
	}
	opts = mustParse(t, []string{"-u", "http://a.com/", "-U", "-C"})
	if !opts.Uppercase || !opts.Capitalization {
		t.Error("-U -C flags")
	}
}

func TestForceOverwriteExtensions(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-f", "--overwrite-extensions"})
	if !opts.ForceExtensions || !opts.OverwriteExtensions {
		t.Error("force/overwrite flags")
	}
}

// ---- 通用参数 ----

func TestThreads(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-t", "50"})
	if opts.ThreadCount != 50 {
		t.Errorf("threads = %d", opts.ThreadCount)
	}
	// -t 0 是假值：会被配置默认值 25 替换（与 dirsearch 的 or 语义一致）
	opts = mustParse(t, []string{"-u", "http://a.com/", "-t", "0"})
	if opts.ThreadCount != 25 {
		t.Errorf("threads=0 should fall back to default 25, got %d", opts.ThreadCount)
	}
}

func TestAsyncMode(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-a"})
	if opts.AsyncMode == nil || !*opts.AsyncMode {
		t.Error("-a should set async true")
	}
	opts = mustParse(t, []string{"-u", "http://a.com/", "--no-async"})
	if opts.AsyncMode == nil || *opts.AsyncMode {
		t.Error("--no-async should set async false")
	}
}

func TestRecursionFlags(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-r", "--deep-recursive",
		"--force-recursive", "-R", "3", "--recursion-status", "200,301"})
	if !opts.Recursive || !opts.DeepRecursive || !opts.ForceRecursive {
		t.Errorf("recursion flags")
	}
	if opts.RecursionDepth == nil || *opts.RecursionDepth != 3 {
		t.Errorf("recursion depth = %v", opts.RecursionDepth)
	}
	if !opts.RecursionStatus.Contains(200) || !opts.RecursionStatus.Contains(301) {
		t.Errorf("recursion status = %v", opts.RecursionStatus.Values())
	}
}

func TestSubdirs(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--subdirs", "api/,admin"})
	if len(opts.Subdirs) != 2 || opts.Subdirs[0] != "api/" || opts.Subdirs[1] != "admin/" {
		t.Errorf("subdirs = %v", opts.Subdirs)
	}
	opts = mustParse(t, []string{"-u", "http://a.com/", "--exclude-subdirs", "/secret"})
	if len(opts.ExcludeSubdirs) != 1 || opts.ExcludeSubdirs[0] != "secret/" {
		t.Errorf("exclude subdirs = %v", opts.ExcludeSubdirs)
	}
}

// ---- 状态码过滤 ----

func TestStatusCodes(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-i", "200,300-399", "-x", "404",
		"--skip-on-status", "429", "--recursion-status", "200-399,401,403"})
	if !opts.IncludeStatus.Contains(200) || !opts.IncludeStatus.Contains(350) {
		t.Errorf("include status = %v", opts.IncludeStatus.Values())
	}
	if !opts.ExcludeStatus.Contains(404) {
		t.Error("exclude 404")
	}
	if !opts.SkipOnStatus.Contains(429) {
		t.Error("skip 429")
	}
	if !opts.RecursionStatus.Contains(403) {
		t.Error("recursion 403")
	}
}

func TestInvalidStatusCode(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "-i", "abc"}, testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "Invalid status code or status code range: abc") {
		t.Errorf("invalid status error = %v", err)
	}
}

// ---- 响应过滤 ----

func TestResponseFilters(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--exclude-sizes", "0,4KB",
		"--exclude-text", "not found", "--exclude-text", "missing",
		"--exclude-regex", "^404$",
		"--exclude-redirect", "/index.html",
		"--min-response-size", "1KB",
		"--max-response-size", "2MB"})
	if len(opts.ExcludeSizes) != 2 {
		t.Errorf("exclude sizes = %v", opts.ExcludeSizes)
	}
	if _, ok := opts.ExcludeSizes[4096]; !ok {
		t.Error("4KB size missing")
	}
	if len(opts.ExcludeTexts) != 2 {
		t.Errorf("exclude texts = %v", opts.ExcludeTexts)
	}
	if opts.ExcludeRegex != "^404$" || opts.ExcludeRedirect != "/index.html" {
		t.Errorf("regex/redirect = %q/%q", opts.ExcludeRegex, opts.ExcludeRedirect)
	}
	if opts.MinResponseSize != 1024 || opts.MaxResponseSize != 2*1024*1024 {
		t.Errorf("min/max size = %d/%d", opts.MinResponseSize, opts.MaxResponseSize)
	}
}

func TestInvalidExcludeRegex(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "--exclude-regex", "["},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "regular expression") {
		t.Errorf("invalid regex error = %v", err)
	}
}

func TestTimeLimits(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--max-time", "60", "--target-max-time", "30"})
	if *opts.MaxTime != 60 || *opts.TargetMaxTime != 30 {
		t.Errorf("time limits = %v/%v", *opts.MaxTime, *opts.TargetMaxTime)
	}
}

// ---- 高级过滤 ----

func TestAdvancedFiltering(t *testing.T) {
	opts := mustParse(t, []string{
		"-u", "http://a.com/",
		"--matcher-mode", "and", "--filter-mode", "or",
		"--match-status", "200,401", "--filter-status", "500-599",
		"--match-size", "100-2000", "--filter-size", "0",
		"--match-words", "10-100", "--filter-words", "0",
		"--match-lines", "2-50", "--filter-lines", "0",
		"--match-regex", "admin", "--filter-regex", "not found",
		"--match-header", "Server: nginx", "--filter-header", "X-Debug",
		"--match-header-regex", "^Server", "--filter-header-regex", "Debug$",
		"--match-time", ">100", "--filter-time", "<50",
		"--auto-calibration", "--filter-threshold", "10",
	})
	if opts.MatcherMode != "and" || opts.FilterMode != "or" {
		t.Errorf("modes = %q/%q", opts.MatcherMode, opts.FilterMode)
	}
	if !opts.MatchStatus.Contains(401) || !opts.FilterStatus.Contains(599) {
		t.Error("match/filter status")
	}
	if len(opts.MatchSizes) != 1 || opts.MatchSizes[0].Min != 100 || opts.MatchSizes[0].Max != 2000 {
		t.Errorf("match sizes = %v", opts.MatchSizes)
	}
	if len(opts.MatchWords) != 1 || len(opts.MatchLines) != 1 {
		t.Error("words/lines ranges")
	}
	if opts.MatchRegex != "admin" || opts.FilterRegex != "not found" {
		t.Error("regexes")
	}
	if len(opts.MatchHeaders) != 1 || len(opts.FilterHeaders) != 1 {
		t.Error("headers")
	}
	if opts.MatchHeaderRegex != "^Server" || opts.FilterHeaderRegex != "Debug$" {
		t.Error("header regexes")
	}
	if len(opts.MatchTime) != 1 || opts.MatchTime[0].Op != ">" {
		t.Errorf("match time = %v", opts.MatchTime)
	}
	if len(opts.FilterTime) != 1 || opts.FilterTime[0].Op != "<" {
		t.Errorf("filter time = %v", opts.FilterTime)
	}
	if !opts.AutoCalibration || *opts.FilterThreshold != 10 {
		t.Error("auto calibration / threshold")
	}
}

func TestAdvancedModeAliases(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--mmode", "or", "--fmode", "and",
		"--mc", "200", "--fc", "404", "--ms", "100", "--fs", "0",
		"--mw", "5", "--fw", "0", "--ml", "1", "--fl", "0",
		"--mr", "x", "--fr", "y", "--mt", ">1", "--ft", "<2"})
	if opts.MatcherMode != "or" || opts.FilterMode != "and" {
		t.Errorf("alias modes")
	}
	if !opts.MatchStatus.Contains(200) || !opts.FilterStatus.Contains(404) {
		t.Error("alias statuses")
	}
}

func TestInvalidAdvancedMode(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "--matcher-mode", "xor"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "--matcher-mode must be either 'and' or 'or'") {
		t.Errorf("invalid mode error = %v", err)
	}
}

func TestAdvancedModeConfigDefault(t *testing.T) {
	// 未显式指定时默认为 or
	opts := mustParse(t, []string{"-u", "http://a.com/"})
	if opts.MatcherMode != "or" || opts.FilterMode != "or" {
		t.Errorf("default modes = %q/%q", opts.MatcherMode, opts.FilterMode)
	}
}

// ---- 请求参数 ----

func TestRequestOptions(t *testing.T) {
	fs := testFS(map[string]string{
		"data.bin":    "raw-body-data",
		"headers.txt": "X-A: 1\nX-B: 2\n",
	})
	opts := mustParseFS(t, []string{"-u", "http://a.com/", "-m", "post",
		"-d", "a=1&b=2", "-H", "X-Custom: yes", "--headers-file", "headers.txt",
		"-F", "--random-agent"}, fs)
	if opts.HTTPMethod != "POST" {
		t.Errorf("method = %q", opts.HTTPMethod)
	}
	if string(opts.Data) != "a=1&b=2" {
		t.Errorf("data = %q", opts.Data)
	}
	// 文件头与 CLI 头合并
	if opts.Headers["x-custom"] != "yes" || opts.Headers["x-a"] != "1" {
		t.Errorf("headers = %v", opts.Headers)
	}
	if !opts.FollowRedirects || !opts.RandomAgents {
		t.Error("follow/random flags")
	}
}

func TestDataFile(t *testing.T) {
	fs := testFS(map[string]string{"body.bin": "binary-payload"})
	opts, err := ParseOptions([]string{"-u", "http://a.com/", "--data-file", "body.bin"},
		fs, utils.NewResourceFS())
	if err != nil {
		t.Fatal(err)
	}
	if string(opts.Data) != "binary-payload" {
		t.Errorf("data from file = %q", opts.Data)
	}
}

func TestUserAgentCookie(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--user-agent", "MyAgent", "--cookie", "SID=123"})
	if opts.Headers["user-agent"] != "MyAgent" {
		t.Errorf("user-agent header = %q", opts.Headers["user-agent"])
	}
	if opts.Headers["cookie"] != "SID=123" {
		t.Errorf("cookie header = %q", opts.Headers["cookie"])
	}
}

func TestInvalidHeader(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "-H", "NoColonHere"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "Invalid headers") {
		t.Errorf("invalid header error = %v", err)
	}
}

// ---- 认证 ----

func TestAuthValidation(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "--auth", "u:p"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "Please select the authentication type") {
		t.Errorf("missing auth type error = %v", err)
	}

	_, err = ParseOptions([]string{"-u", "http://a.com/", "--auth-type", "basic"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "No authentication credential found") {
		t.Errorf("missing credential error = %v", err)
	}

	_, err = ParseOptions([]string{"-u", "http://a.com/", "--auth", "u:p", "--auth-type", "oauth1"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "'oauth1' is not in available authentication types") {
		t.Errorf("invalid type error = %v", err)
	}

	opts := mustParse(t, []string{"-u", "http://a.com/", "--auth", "u:p", "--auth-type", "digest"})
	if opts.Auth != "u:p" || opts.AuthType != "digest" {
		t.Errorf("auth = %q/%q", opts.Auth, opts.AuthType)
	}
}

// ---- 连接参数 ----

func TestConnectionOptions(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--timeout", "5.5", "--delay", "0.5",
		"-p", "socks5://127.0.0.1:9050", "-p", "localhost:8080",
		"--proxy-auth", "u:p", "--replay-proxy", "localhost:8000",
		"--max-rate", "100", "--retries", "3", "--ip", "1.2.3.4"})
	if *opts.Timeout != 5.5 || *opts.Delay != 0.5 {
		t.Errorf("timeout/delay = %v/%v", *opts.Timeout, *opts.Delay)
	}
	if len(opts.Proxies) != 2 || opts.Proxies[0] != "socks5://127.0.0.1:9050" {
		t.Errorf("proxies = %v", opts.Proxies)
	}
	if opts.ProxyAuth != "u:p" || opts.ReplayProxy != "localhost:8000" {
		t.Error("proxy auth / replay proxy")
	}
	if *opts.MaxRate != 100 || *opts.MaxRetries != 3 {
		t.Error("rate/retries")
	}
	if opts.IP != "1.2.3.4" {
		t.Error("ip")
	}
}

func TestTorProxy(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--tor"})
	if len(opts.Proxies) != 2 || opts.Proxies[0] != "socks5://127.0.0.1:9050" {
		t.Errorf("tor proxies = %v", opts.Proxies)
	}
}

func TestNumericValidation(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "--timeout", "0"}, testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "--timeout must be finite and greater than zero") {
		t.Errorf("timeout error = %v", err)
	}
	_, err = ParseOptions([]string{"-u", "http://a.com/", "--delay", "-1"}, testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "--delay must be finite and zero or greater") {
		t.Errorf("delay error = %v", err)
	}
	_, err = ParseOptions([]string{"-u", "http://a.com/", "--retries", "-2"}, testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "--retries must be zero or greater") {
		t.Errorf("retries error = %v", err)
	}
	_, err = ParseOptions([]string{"-u", "http://a.com/", "--max-rate", "-1"}, testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "--max-rate must be zero or greater") {
		t.Errorf("max-rate error = %v", err)
	}
	_, err = ParseOptions([]string{"-u", "http://a.com/", "-R", "-1"}, testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "--max-recursion-depth must be zero or greater") {
		t.Errorf("recursion error = %v", err)
	}
}

func TestInvalidBackend(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "--wordlist-backend", "rust"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "--wordlist-backend must be one of") {
		t.Errorf("wordlist backend error = %v", err)
	}
	_, err = ParseOptions([]string{"-u", "http://a.com/", "--request-backend", "go"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "--request-backend must be one of") {
		t.Errorf("request backend error = %v", err)
	}
}

func TestSchemeValidation(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--scheme", "https"})
	if opts.Scheme != "https" {
		t.Errorf("scheme = %q", opts.Scheme)
	}
}

// ---- 输出参数 ----

func TestOutputFormats(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "-O", "simple,json", "-o", "out.{format}"})
	if len(opts.OutputFormats) != 2 || opts.OutputFormats[0] != "simple" {
		t.Errorf("formats = %v", opts.OutputFormats)
	}
	if opts.OutputFile != "/abs/out.{format}" {
		t.Errorf("output file = %q", opts.OutputFile)
	}
}

func TestOutputFileWithoutFormat(t *testing.T) {
	// 与 dirsearch 一致：未指定 -O 时回退到配置默认值 plain
	opts, err := ParseOptions([]string{"-u", "http://a.com/", "-o", "out.txt"},
		testFS(nil), utils.NewResourceFS())
	if err != nil {
		t.Fatalf("output file without -O should not error: %v", err)
	}
	if len(opts.OutputFormats) != 1 || opts.OutputFormats[0] != "plain" {
		t.Errorf("default output formats = %v", opts.OutputFormats)
	}
}

func TestInvalidOutputFormat(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "-O", "docx", "-o", "out.docx"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "Invalid output format(s): docx") {
		t.Errorf("invalid format error = %v", err)
	}
}

func TestOutputFormatConflict(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "-O", "json,xml", "-o", "same.txt"},
		testFS(nil), utils.NewResourceFS())
	if err == nil || !containsStr(err.Error(), "sharing the same output file") {
		t.Errorf("conflict error = %v", err)
	}
	// {format} 变量解决冲突
	opts := mustParse(t, []string{"-u", "http://a.com/", "-O", "json,xml", "-o", "out.{format}"})
	if len(opts.OutputFormats) != 2 {
		t.Errorf("formats with variable = %v", opts.OutputFormats)
	}
}

func TestDatabaseOutputURLs(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/",
		"--mysql-url", "mysql://root@localhost/db", "--postgres-url", "postgres://root@localhost/db"})
	formats := map[string]bool{}
	for _, f := range opts.OutputFormats {
		formats[f] = true
	}
	if !formats["mysql"] || !formats["postgresql"] {
		t.Errorf("db formats = %v", opts.OutputFormats)
	}
}

// ---- 视图参数 ----

func TestViewOptions(t *testing.T) {
	opts := mustParse(t, []string{"-u", "http://a.com/", "--full-url", "--redirects-history",
		"--no-color", "-q", "-v"})
	if !opts.FullURL || !opts.RedirectsHistory || !opts.Quiet || !opts.Verbose {
		t.Errorf("view flags")
	}
	if opts.Color == nil || *opts.Color {
		t.Error("--no-color should set color false")
	}
}

// ---- 位置参数 ----

func TestUnexpectedPositional(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "extra"}, testFS(nil), utils.NewResourceFS())
	exitErr, ok := err.(*ExitError)
	if !ok || !containsStr(exitErr.Message, "unexpected positional argument") {
		t.Errorf("positional error = %v", err)
	}
}

func TestUnknownOption(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "http://a.com/", "--no-such-option"}, testFS(nil), utils.NewResourceFS())
	exitErr, ok := err.(*ExitError)
	if !ok || !containsStr(exitErr.Message, "no such option: --no-such-option") {
		t.Errorf("unknown option error = %v", err)
	}
	if !exitErr.ToStderr {
		t.Error("parser errors should target stderr")
	}
}

// ---- config.ini 合并 ----

func TestConfigMerge(t *testing.T) {
	configPath := writeTempConfig(t, `
[general]
threads = 33
recursive = True
[dictionary]
default-extensions = asp
`)
	opts, err := ParseOptions([]string{"-u", "http://a.com/", "--config", configPath},
		testFS(nil), utils.NewResourceFS())
	if err != nil {
		t.Fatal(err)
	}
	if opts.ThreadCount != 33 {
		t.Errorf("threads from config = %d", opts.ThreadCount)
	}
	if !opts.Recursive {
		t.Error("recursive from config")
	}
	if len(opts.Extensions) != 1 || opts.Extensions[0] != "asp" {
		t.Errorf("extensions from config = %v", opts.Extensions)
	}
}

func TestConfigCLIPrecedence(t *testing.T) {
	fs := testFS(map[string]string{
		"config.ini": "[general]\nthreads = 33\n[dictionary]\ndefault-extensions = asp\n",
	})
	opts, err := ParseOptions([]string{"-u", "http://a.com/", "--config", "config.ini", "-t", "7", "-e", "php"},
		fs, utils.NewResourceFS())
	if err != nil {
		t.Fatal(err)
	}
	if opts.ThreadCount != 7 {
		t.Errorf("CLI threads should win: %d", opts.ThreadCount)
	}
	if opts.Extensions[0] != "php" {
		t.Errorf("CLI extensions should win: %v", opts.Extensions)
	}
}

// ---- 会话模式 ----

func TestSessionSkipsValidation(t *testing.T) {
	// 会话模式下不需要目标 URL
	opts, err := ParseOptions([]string{"-s", "session.json"}, testFS(nil), utils.NewResourceFS())
	if err != nil {
		t.Fatalf("session parse error: %v", err)
	}
	if opts.SessionFile != "session.json" {
		t.Errorf("session file = %q", opts.SessionFile)
	}
}

func TestListSessionsFlag(t *testing.T) {
	opts := mustParse(t, []string{"--list-sessions"})
	if !opts.ListSessions {
		t.Error("list sessions flag")
	}
}

func TestSessionIdConflict(t *testing.T) {
	// --session 与 --session-id 的互斥在 main 中处理，
	// ParseOptions 在两者同时存在时直接返回（不校验目标）
	_, err := ParseOptions([]string{"-s", "a.json", "--session-id", "1"}, testFS(nil), utils.NewResourceFS())
	if err != nil {
		t.Errorf("session parse should succeed: %v", err)
	}
}

// ---- mustParseFS 辅助 ----

func mustParseFS(t *testing.T, args []string, fs *Filesystem) *Options {
	t.Helper()
	opts, err := ParseOptions(args, fs, utils.NewResourceFS())
	if err != nil {
		t.Fatalf("ParseOptions(%v) error: %v", args, err)
	}
	return opts
}

func containsStr(s, sub string) bool {
	return strings.Contains(s, sub)
}

// writeTempConfig 写入临时配置文件并返回路径。
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.ini")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
