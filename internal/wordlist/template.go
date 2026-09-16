// Package wordlist 实现词表的模板展开、生成与运行时迭代，
// 对应 dirsearch 的 lib/core/dictionary.py、wordlist_backend.py、wordlist_template.py。
package wordlist

import (
	"regexp"
	"strings"
	"time"

	"hidir/internal/settings"
)

// TokenRegex 匹配 %TOKEN% 占位符（大小写不敏感）。
var TokenRegex = regexp.MustCompile(`(?i)%([A-Z0-9_:/-]+)%`)

// ExtensionToken 是扩展名占位符的规范化名称。
var ExtensionToken = strings.ToUpper(strings.Trim(settings.ExtensionTag, "%"))

// defaultPlaceholders 是内置占位符取值集合，与 dirsearch 的 DEFAULT_PLACEHOLDERS 一致。
var defaultPlaceholders = map[string][]string{
	"SUBJECT": {"user", "users", "account", "accounts", "profile", "article", "articles",
		"post", "posts", "product", "products", "order", "orders", "invoice", "invoices"},
	"CRUD_OP": {"create", "read", "update", "delete", "list", "get", "add", "edit", "remove", "search"},
	"AUTH_OP": {"login", "logout", "signin", "signout", "signup", "register", "reset",
		"forgot", "password", "oauth", "sso"},
	"ADMIN_OP": {"admin", "dashboard", "panel", "manage", "settings", "users", "roles", "permissions"},
	"ENV":      {"dev", "development", "test", "stage", "staging", "prod", "production", "local"},
	"SEP":      {"-", "_", ".", "/"},
}

// datePlaceholders 是与日期相关的占位符。
var datePlaceholders = map[string]bool{
	"YYYY": true, "YY": true, "MM": true, "DD": true, "DATE": true, "DATE_COMPACT": true,
}

// IsTemplateToken 判断占位符是否受支持（EXT、内置、日期或 CATEGORY:*）。
func IsTemplateToken(token string) bool {
	normalized := strings.ToUpper(strings.Trim(token, "%"))
	if normalized == ExtensionToken {
		return true
	}
	if _, ok := defaultPlaceholders[normalized]; ok {
		return true
	}
	if datePlaceholders[normalized] {
		return true
	}
	return strings.HasPrefix(normalized, "CATEGORY:")
}

// CategoryLoader 按分类名加载词表条目（由调用方注入，避免循环依赖）。
type CategoryLoader func(name string) []string

// placeholderValues 汇总全部占位符的取值。
func placeholderValues(extensions []string, loader CategoryLoader, now time.Time) map[string][]string {
	values := make(map[string][]string, 32)
	for k, v := range defaultPlaceholders {
		values[k] = v
	}
	values[ExtensionToken] = extensions

	// 数据库引擎占位符
	dbEngines := []string{"mysql", "postgres", "postgresql", "sqlite", "mariadb", "mongodb", "redis"}
	values["DB"] = dbEngines
	values["DB_ENGINE"] = dbEngines

	// 归档与备份扩展名
	archive := []string{"zip", "tar", "tar.gz", "tgz", "gz", "7z", "rar", "bak"}
	values["ARCHIVE"] = archive
	values["ARCHIVE_EXT"] = archive
	backup := append(append([]string{}, archive...), "bkp", "bkup", "old", "swn", "swp")
	values["BACKUP"] = backup
	values["BACKUP_EXT"] = backup

	values["API_VERSION"] = []string{"v1", "v2", "v3", "v4", "latest", "beta"}

	values["YYYY"] = []string{now.Format("2006")}
	values["YY"] = []string{now.Format("06")}
	values["MM"] = []string{now.Format("01")}
	values["DD"] = []string{now.Format("02")}
	values["DATE"] = []string{now.Format("2006-01-02")}
	values["DATE_COMPACT"] = []string{now.Format("20060102")}

	_ = loader
	return values
}

// resolveToken 解析单个占位符的取值；未知返回 nil。
func resolveToken(token string, values map[string][]string, loader CategoryLoader) []string {
	if strings.HasPrefix(token, "CATEGORY:") {
		if loader == nil {
			return []string{}
		}
		return loader(strings.ToLower(strings.TrimPrefix(token, "CATEGORY:")))
	}
	if v, ok := values[token]; ok {
		return v
	}
	return nil
}

// ExpandTemplateLine 展开一行词表中的全部占位符，返回所有组合。
// 语义与 dirsearch 的 expand_template_line 一致：
//   - 无占位符 → 原样返回；
//   - 存在无法解析的占位符 → 原样返回；
//   - 任一占位符取值为空 → 不产出；
//   - 否则输出笛卡尔积组合。
func ExpandTemplateLine(line string, extensions []string, loader CategoryLoader, now time.Time) []string {
	values := placeholderValues(extensions, loader, now)

	// 按出现顺序收集去重后的有效占位符
	var tokens []string
	seen := map[string]bool{}
	for _, m := range TokenRegex.FindAllStringSubmatch(line, -1) {
		normalized := strings.ToUpper(m[1])
		if seen[normalized] {
			continue
		}
		if resolveToken(normalized, values, loader) != nil {
			tokens = append(tokens, normalized)
			seen[normalized] = true
		}
	}

	if len(tokens) == 0 {
		return []string{line}
	}

	// 展开笛卡尔积
	combinations := [][]string{{}}
	for _, token := range tokens {
		expansion := resolveToken(token, values, loader)
		if len(expansion) == 0 {
			return nil
		}
		var next [][]string
		for _, prefix := range combinations {
			for _, value := range expansion {
				row := make([]string, len(prefix), len(prefix)+1)
				copy(row, prefix)
				next = append(next, append(row, value))
			}
		}
		combinations = next
	}

	out := make([]string, 0, len(combinations))
	for _, combo := range combinations {
		rendered := line
		for i, token := range tokens {
			rendered = replaceToken(rendered, token, combo[i])
		}
		out = append(out, rendered)
	}
	return out
}

// replaceToken 大小写不敏感地替换 %TOKEN% 占位符。
func replaceToken(line, token, value string) string {
	re, err := compileTokenRegex(token)
	if err != nil {
		return line
	}
	return re.ReplaceAllLiteralString(line, value)
}

// tokenRegexCache 缓存每个占位符的替换正则。
var tokenRegexCache = map[string]*regexp.Regexp{}

func compileTokenRegex(token string) (*regexp.Regexp, error) {
	if re, ok := tokenRegexCache[token]; ok {
		return re, nil
	}
	re, err := regexp.Compile(`(?i)%` + regexp.QuoteMeta(token) + `%`)
	if err != nil {
		return nil, err
	}
	tokenRegexCache[token] = re
	return re, nil
}

// GenerateBackupPaths 为已发现的文件路径生成备份候选路径。
// 与 dirsearch 的 generate_backup_paths 一致。
func GenerateBackupPaths(path string) []string {
	idx := strings.LastIndex(path, "/")
	directory, filename := "", path
	if idx >= 0 {
		directory = path[:idx+1]
		filename = path[idx+1:]
	}
	if filename == "" || !isAlnum(filename[len(filename)-1]) {
		return nil
	}
	extIdx := strings.LastIndex(filename, ".")
	if extIdx < 0 || extIdx == len(filename)-1 {
		return nil
	}

	archiveExtensions := []string{"zip", "tar", "tar.gz", "tgz", "gz", "7z", "rar", "bak"}
	backupExtensions := append(append([]string{}, archiveExtensions...), "bkp", "bkup", "old", "swn", "swp")
	mediaExtensions := []string{
		"webm", "mkv", "avi", "ts", "mov", "qt", "amv", "mp4", "m4p", "m4v",
		"mp3", "swf", "mpg", "mpeg", "jpg", "jpeg", "pjpeg", "png", "woff",
		"svg", "webp", "bmp", "pdf", "wav", "vtt",
	}

	if hasAnyExtension(path, mediaExtensions) || hasAnyExtension(path, backupExtensions) {
		return nil
	}

	var candidates []string
	candidates = append(candidates, path+"~")
	for _, ext := range backupExtensions {
		candidates = append(candidates, path+"."+ext)
	}
	if extIdx > 0 {
		basenamePath := directory + filename[:extIdx]
		for _, ext := range archiveExtensions {
			candidates = append(candidates, basenamePath+"."+ext)
		}
	}

	seen := map[string]bool{}
	var out []string
	for _, candidate := range candidates {
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	return out
}

// isAlnum 判断字节是否为字母或数字。
func isAlnum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// hasAnyExtension 判断路径是否以任一扩展名结尾（带点、大小写不敏感）。
func hasAnyExtension(path string, extensions []string) bool {
	lower := strings.ToLower(path)
	for _, ext := range extensions {
		if strings.HasSuffix(lower, "."+strings.ToLower(ext)) {
			return true
		}
	}
	return false
}
