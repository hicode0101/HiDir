package parse

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
)

func TestParseArguments(t *testing.T) {
	// 保存原始命令行参数
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// 测试用例1：基本参数
	t.Run("BasicParameters", func(t *testing.T) {
		// 重置pflag.CommandLine
		pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
		os.Args = []string{"./hidir", "-u", "https://example.com", "-e", "php,html"}
		opts := ParseArguments()

		if len(opts.URLs) != 1 {
			t.Errorf("Expected 1 URL, got %d", len(opts.URLs))
		}
		if opts.URLs[0] != "https://example.com" {
			t.Errorf("Expected URL https://example.com, got %s", opts.URLs[0])
		}
		if opts.Extensions != "php,html" {
			t.Errorf("Expected extensions php,html, got %s", opts.Extensions)
		}
	})

	// 测试用例2：多URL
	t.Run("MultipleURLs", func(t *testing.T) {
		// 重置pflag.CommandLine
		pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
		os.Args = []string{"./hidir", "-u", "https://example.com", "-u", "https://test.com"}
		opts := ParseArguments()

		if len(opts.URLs) != 2 {
			t.Errorf("Expected 2 URLs, got %d", len(opts.URLs))
		}
		if opts.URLs[0] != "https://example.com" {
			t.Errorf("Expected first URL https://example.com, got %s", opts.URLs[0])
		}
		if opts.URLs[1] != "https://test.com" {
			t.Errorf("Expected second URL https://test.com, got %s", opts.URLs[1])
		}
	})

	// 测试用例3：URL文件
	t.Run("URLFile", func(t *testing.T) {
		// 重置pflag.CommandLine
		pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
		os.Args = []string{"./hidir", "-l", "urls.txt"}
		opts := ParseArguments()

		if opts.URLFile != "urls.txt" {
			t.Errorf("Expected URL file urls.txt, got %s", opts.URLFile)
		}
	})

	// 测试用例4：字典设置
	t.Run("DictionarySettings", func(t *testing.T) {
		// 重置pflag.CommandLine
		pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
		os.Args = []string{"./hidir", "-u", "https://example.com", "-w", "dict1.txt,dict2.txt", "-f", "-U"}
		opts := ParseArguments()

		if opts.Wordlists != "dict1.txt,dict2.txt" {
			t.Errorf("Expected wordlists dict1.txt,dict2.txt, got %s", opts.Wordlists)
		}
		if !opts.ForceExtensions {
			t.Error("Expected force-extensions to be true")
		}
		if !opts.Uppercase {
			t.Error("Expected uppercase to be true")
		}
	})

	// 测试用例5：通用设置
	t.Run("GeneralSettings", func(t *testing.T) {
		// 重置pflag.CommandLine
		pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
		os.Args = []string{"./hidir", "-u", "https://example.com", "-t", "100", "-r", "-R", "3"}
		opts := ParseArguments()

		if opts.ThreadCount != 100 {
			t.Errorf("Expected thread count 100, got %d", opts.ThreadCount)
		}
		if !opts.Recursive {
			t.Error("Expected recursive to be true")
		}
		if opts.RecursionDepth != 3 {
			t.Errorf("Expected recursion depth 3, got %d", opts.RecursionDepth)
		}
	})

	// 测试用例6：请求设置
	t.Run("RequestSettings", func(t *testing.T) {
		// 重置pflag.CommandLine
		pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
		os.Args = []string{"./hidir", "-u", "https://example.com", "-m", "POST", "-d", "key=value", "-H", "X-Test: value"}
		opts := ParseArguments()

		if opts.HTTPMethod != "POST" {
			t.Errorf("Expected HTTP method POST, got %s", opts.HTTPMethod)
		}
		if opts.Data != "key=value" {
			t.Errorf("Expected data key=value, got %s", opts.Data)
		}
		if len(opts.Headers) != 1 {
			t.Errorf("Expected 1 header, got %d", len(opts.Headers))
		}
		if opts.Headers[0] != "X-Test: value" {
			t.Errorf("Expected header X-Test: value, got %s", opts.Headers[0])
		}
	})

	// 测试用例7：连接设置
	t.Run("ConnectionSettings", func(t *testing.T) {
		// 重置pflag.CommandLine
		pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
		os.Args = []string{"./hidir", "-u", "https://example.com", "--timeout", "10", "--delay", "0.5", "--proxy", "http://proxy.example.com:8080"}
		opts := ParseArguments()

		if opts.Timeout != 10 {
			t.Errorf("Expected timeout 10, got %f", opts.Timeout)
		}
		if opts.Delay != 0.5 {
			t.Errorf("Expected delay 0.5, got %f", opts.Delay)
		}
		if len(opts.Proxies) != 1 {
			t.Errorf("Expected 1 proxy, got %d", len(opts.Proxies))
		}
		if opts.Proxies[0] != "http://proxy.example.com:8080" {
			t.Errorf("Expected proxy http://proxy.example.com:8080, got %s", opts.Proxies[0])
		}
	})

	// 测试用例8：高级设置
	t.Run("AdvancedSettings", func(t *testing.T) {
		// 重置pflag.CommandLine
		pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
		os.Args = []string{"./hidir", "-u", "https://example.com", "--crawl"}
		opts := ParseArguments()

		if !opts.Crawl {
			t.Error("Expected crawl to be true")
		}
	})

	// 测试用例9：视图设置
	t.Run("ViewSettings", func(t *testing.T) {
		// 重置pflag.CommandLine
		pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
		os.Args = []string{"./hidir", "-u", "https://example.com", "--full-url", "-q"}
		opts := ParseArguments()

		if !opts.FullURL {
			t.Error("Expected full-url to be true")
		}
		if !opts.Quiet {
			t.Error("Expected quiet to be true")
		}
	})

	// 测试用例10：输出设置
	t.Run("OutputSettings", func(t *testing.T) {
		// 重置pflag.CommandLine
		pflag.CommandLine = pflag.NewFlagSet("", pflag.ExitOnError)
		os.Args = []string{"./hidir", "-u", "https://example.com", "-o", "results.txt", "--format", "json", "--log", "hidir.log"}
		opts := ParseArguments()

		if opts.OutputFile != "results.txt" {
			t.Errorf("Expected output file results.txt, got %s", opts.OutputFile)
		}
		if opts.OutputFormat != "json" {
			t.Errorf("Expected output format json, got %s", opts.OutputFormat)
		}
		if opts.LogFile != "hidir.log" {
			t.Errorf("Expected log file hidir.log, got %s", opts.LogFile)
		}
	})
}
