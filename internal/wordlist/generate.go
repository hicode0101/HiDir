package wordlist

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"hidir/internal/utils"
)

// LimitError 表示生成的词表超过 --wordlist-max-size 上限。
type LimitError struct {
	// MaxSize 是允许的最大条目数。
	MaxSize int
}

// Error 实现 error 接口，消息与 dirsearch 的 WordlistLimitError 一致。
func (e *LimitError) Error() string {
	return fmt.Sprintf("Generated wordlist exceeded --wordlist-max-size (%d)", e.MaxSize)
}

// GenerateParams 汇总词表生成所需的选项。
type GenerateParams struct {
	// Extensions 是选定的扩展名列表（对应 -e）。
	Extensions []string
	// ForceExtensions 对应 -f。
	ForceExtensions bool
	// OverwriteExtensions 对应 --overwrite-extensions。
	OverwriteExtensions bool
	// ExcludeExtensions 是排除的扩展名列表。
	ExcludeExtensions []string
	// Prefixes 是自定义前缀。
	Prefixes []string
	// Suffixes 是自定义后缀。
	Suffixes []string
	// Lowercase/Uppercase/Capitalization 对应 -L/-U/-C。
	Lowercase      bool
	Uppercase      bool
	Capitalization bool
	// MaxSize 是最大生成条目数（0 表示不限制）。
	MaxSize int
	// IsBlacklist 表示生成的是黑名单词表（跳过扩展与前后缀改写）。
	IsBlacklist bool
}

// extensionRecognitionRegex 用于识别路径是否已带扩展名（与 dirsearch 一致）。
var extensionRecognitionRegex = regexp.MustCompile(`\w+([.][a-zA-Z0-9]{2,5}){1,3}~?$`)

// excludeOverwriteExtensions 是 overwrite-extensions 时不会被覆盖的扩展名
// （媒体扩展名 + 常见静态资源扩展名）。
var excludeOverwriteExtensions = []string{
	"axd", "cache", "coffee", "conf", "config", "css", "dll", "lock", "log",
	"key", "pub", "properties", "ini", "jar", "js", "json", "toml", "txt",
	"xml", "yaml", "yml",
	"webm", "mkv", "avi", "ts", "mov", "qt", "amv", "mp4", "m4p", "m4v",
	"mp3", "swf", "mpg", "mpeg", "jpg", "jpeg", "pjpeg", "png", "woff",
	"svg", "webp", "bmp", "pdf", "wav", "vtt",
}

// IsValidPath 判断词条是否可用：跳过空行、注释行与被排除扩展名的路径。
// 对应 dirsearch 的 is_valid_path。
func IsValidPath(path string, excludeExtensions []string) bool {
	if path == "" || strings.HasPrefix(path, "#") {
		return false
	}
	cleaned := utils.CleanPath(path, false, false)
	for _, ext := range excludeExtensions {
		if strings.HasSuffix(cleaned, "."+ext) {
			return false
		}
	}
	return true
}

// OrderedSet 保持插入顺序的去重集合。
type OrderedSet struct {
	items []string
	index map[string]struct{}
}

// NewOrderedSet 创建空集合。
func NewOrderedSet() *OrderedSet {
	return &OrderedSet{index: map[string]struct{}{}}
}

// Add 添加元素并检查上限（maxSize<=0 表示不限制）。
func (s *OrderedSet) Add(item string, maxSize int) error {
	if _, ok := s.index[item]; ok {
		return nil
	}
	s.items = append(s.items, item)
	s.index[item] = struct{}{}
	if maxSize > 0 && len(s.items) > maxSize {
		return &LimitError{MaxSize: maxSize}
	}
	return nil
}

// Items 返回全部元素（保持顺序）。
func (s *OrderedSet) Items() []string {
	return s.items
}

// Len 返回元素个数。
func (s *OrderedSet) Len() int {
	return len(s.items)
}

// LineReader 按文件路径读取词表行。
type LineReader func(path string) ([]string, error)

// Generate 从词表文件生成最终词条列表。
// 行为与 dirsearch 的 PythonWordlistBackend.generate 完全一致：
//  1. 去掉行首 "/" 后做模板展开；
//  2. 过滤无效词条（空行/注释/被排除扩展名）；
//  3. 黑名单词表不做扩展名与前后的改写；
//  4. -f 为无扩展名路径追加扩展名；--overwrite-extensions 覆盖未知扩展名；
//  5. 添加前缀/后缀（存在改写时替换整个词表，与 dirsearch 行为一致）；
//  6. 按需做大小写转换。
func Generate(files []string, params GenerateParams, lineReader LineReader, loader CategoryLoader) ([]string, error) {
	return generateAt(files, params, lineReader, loader, time.Now())
}

// generateAt 支持注入时间的内部实现（便于测试日期占位符）。
func generateAt(files []string, params GenerateParams, lineReader LineReader, loader CategoryLoader, now time.Time) ([]string, error) {
	wordlist := NewOrderedSet()

	for _, dictFile := range files {
		lines, err := lineReader(dictFile)
		if err != nil {
			return nil, err
		}
		for _, line := range lines {
			// 去掉行首 "/"，便于与前缀配合
			line = utils.LstripOnce(line, "/")

			for _, expanded := range ExpandTemplateLine(line, params.Extensions, loader, now) {
				line := expanded
				if !IsValidPath(line, params.ExcludeExtensions) {
					continue
				}

				if err := wordlist.Add(line, params.MaxSize); err != nil {
					return nil, err
				}

				// 黑名单不做扩展名改写，避免漏报
				if params.IsBlacklist {
					continue
				}

				// -f：为不含点且不是目录的路径追加全部扩展名
				if params.ForceExtensions && !strings.Contains(line, ".") && !strings.HasSuffix(line, "/") {
					if err := wordlist.Add(line+"/", params.MaxSize); err != nil {
						return nil, err
					}
					for _, ext := range params.Extensions {
						if err := wordlist.Add(line+"."+ext, params.MaxSize); err != nil {
							return nil, err
						}
					}
					continue
				}

				// --overwrite-extensions：用选定扩展名覆盖未知扩展名
				if params.OverwriteExtensions &&
					!hasSuffixAny(line, params.Extensions) &&
					!hasSuffixAny(line, excludeOverwriteExtensions) &&
					!strings.Contains(line, "?") &&
					!strings.Contains(line, "#") &&
					extensionRecognitionRegex.MatchString(line) {
					base := strings.SplitN(line, ".", 2)[0]
					for _, ext := range params.Extensions {
						if err := wordlist.Add(base+"."+ext, params.MaxSize); err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}

	if !params.IsBlacklist {
		// 添加前缀与后缀（存在改写时替换整个词表）
		altered := NewOrderedSet()
		for _, path := range wordlist.Items() {
			for _, pref := range params.Prefixes {
				if !strings.HasPrefix(path, "/") && !hasPrefixAny(path, params.Prefixes) {
					if err := altered.Add(pref+path, params.MaxSize); err != nil {
						return nil, err
					}
				}
			}
			for _, suff := range params.Suffixes {
				if !strings.HasSuffix(path, "/") && !hasSuffixAny(path, params.Suffixes) &&
					!strings.Contains(path, "?") && !strings.Contains(path, "#") {
					if err := altered.Add(path+suff, params.MaxSize); err != nil {
						return nil, err
					}
				}
			}
		}
		if altered.Len() > 0 {
			wordlist = altered
		}
	}

	switch {
	case params.Lowercase:
		return mapItems(wordlist.Items(), strings.ToLower), nil
	case params.Uppercase:
		return mapItems(wordlist.Items(), strings.ToUpper), nil
	case params.Capitalization:
		return mapItems(wordlist.Items(), capitalize), nil
	default:
		return wordlist.Items(), nil
	}
}

// hasSuffixAny 判断字符串是否以任一候选后缀结尾（不带点，与 dirsearch 一致）。
func hasSuffixAny(s string, candidates []string) bool {
	for _, c := range candidates {
		if c != "" && strings.HasSuffix(s, c) {
			return true
		}
	}
	return false
}

// hasPrefixAny 判断字符串是否以任一候选前缀开头。
func hasPrefixAny(s string, candidates []string) bool {
	for _, c := range candidates {
		if c != "" && strings.HasPrefix(s, c) {
			return true
		}
	}
	return false
}

// mapItems 对全部元素应用转换函数。
func mapItems(items []string, fn func(string) string) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = fn(item)
	}
	return out
}

// capitalize 首字母大写（等价于 Python str.capitalize）。
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
