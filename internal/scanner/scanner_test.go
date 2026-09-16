package scanner

import (
	"strings"
	"testing"
)

// ---- 隐匿词生成 ----

func TestStealthWordGenerator(t *testing.T) {
	gen := NewStealthWordGenerator()
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		word := gen.Generate(nil)
		if len(word) < 15 || len(word) > 30 {
			t.Fatalf("word length out of range: %q", word)
		}
		if seen[word] {
			t.Fatalf("duplicate word: %q", word)
		}
		seen[word] = true
		for _, c := range word {
			if !(c >= 'a' && c <= 'z' || c == '-' || c == '_') {
				t.Fatalf("invalid character in %q", word)
			}
		}
		if strings.Contains(word, "--") || strings.Contains(word, "__") {
			t.Fatalf("double separator in %q", word)
		}
	}
}

func TestStealthWordGeneratorOmit(t *testing.T) {
	gen := NewStealthWordGenerator()
	first := gen.Generate(nil)
	second := gen.Generate(map[string]bool{first: true})
	if second == first {
		t.Fatalf("omitted word returned again: %q", first)
	}
}

// ---- 通配符正则生成 ----

func TestGenerateRedirectRegexFn(t *testing.T) {
	re := GenerateRedirectRegex("/foo/bar", "bar", "/foo/baz", "baz")
	// 两个通配符重定向 /foo/<随机词> 生成的正则应匹配 /foo/前缀 并区分不同路径
	if !strings.HasPrefix(re, "^") || !strings.HasSuffix(re, "$") {
		t.Errorf("regex anchors missing: %q", re)
	}
}

// ---- TestedRegistry ----

func TestTestedRegistryFindDuplicate(t *testing.T) {
	registry := NewTestedRegistry()
	if registry.FindDuplicate(nil) != nil {
		// nil 情况只验证不 panic
	}
}

// ---- Fuzzer 过滤逻辑（不发起请求的部分） ----

func newTestFuzzer(t *testing.T) *Fuzzer {
	t.Helper()
	return NewFuzzer(nil, nil, testOptions(), nil, Callbacks{}, 0)
}

// combineChecks 语义
func TestCombineChecks(t *testing.T) {
	if !combineChecks(nil, "or", true) {
		t.Error("empty checks with or should return default true")
	}
	if combineChecks(nil, "and", false) {
		t.Error("empty checks with and should return default false")
	}
	if !combineChecks([]bool{true, true}, "and", true) {
		t.Error("and with all true")
	}
	if combineChecks([]bool{true, false}, "and", true) {
		t.Error("and with false")
	}
	if !combineChecks([]bool{false, true}, "or", true) {
		t.Error("or with any true")
	}
	if combineChecks([]bool{false, false}, "or", true) {
		t.Error("or with all false")
	}
}
