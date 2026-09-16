package utils

import (
	"regexp"
	"strings"
)

// 动态内容识别正则组，移植自 dirsearch 的 lib/utils/diff.py。
// 这些正则用于把软 404 页面中的易变内容（UUID、时间戳、长随机串等）归一化。
var dynamicTokenRegexes = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`),
	regexp.MustCompile(`(?i)\b[0-9a-f]{16,}\b`),
	regexp.MustCompile(`\b[A-Za-z0-9+/]{24,}={0,2}\b`),
	regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}(?:[T ][0-9:.+-]+Z?)?\b`),
	regexp.MustCompile(`\b\d{2}:\d{2}:\d{2}(?:\.\d+)?\b`),
	regexp.MustCompile(`\b\d{6,}\b`),
}

// htmlAttributeValueRegex 匹配 nonce/csrf/token 等动态 HTML 属性值。
var htmlAttributeValueRegex = regexp.MustCompile(
	`(?i)\b(?:nonce|csrf|token|request[_-]?id|trace[_-]?id|session[_-]?id)=["'][^"']+["']`)

// NormalizeDynamicContent 将常见易变内容替换为稳定标记并压缩空白。
func NormalizeDynamicContent(content string) string {
	normalized := htmlAttributeValueRegex.ReplaceAllString(content, "__DYNAMIC_ATTR__")
	for _, re := range dynamicTokenRegexes {
		normalized = re.ReplaceAllString(normalized, "__DYNAMIC__")
	}
	return strings.Join(strings.Fields(normalized), " ")
}

// ContentSimilarity 计算两个内容归一化后的相似度。
func ContentSimilarity(content1, content2 string) float64 {
	return NormalizedContentSimilarity(
		NormalizeDynamicContent(content1),
		NormalizeDynamicContent(content2),
	)
}

// NormalizedContentSimilarity 计算已归一化内容的词级相似度
// （2*公共词数 / 总词数，为 Python difflib.SequenceMatcher.ratio 的词级近似）。
func NormalizedContentSimilarity(content1, content2 string) float64 {
	words1 := strings.Fields(content1)
	words2 := strings.Fields(content2)
	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	total := float64(len(words1) + len(words2))
	if total == 0 {
		return 0
	}
	counts := make(map[string]int, len(words1))
	for _, w := range words1 {
		counts[w]++
	}
	matched := 0
	for _, w := range words2 {
		if counts[w] > 0 {
			counts[w]--
			matched++
		}
	}
	return 2 * float64(matched) / total
}

// DynamicContentParser 通过两份通配符响应学习"静态模式"，
// 用于判断某个响应是否与通配符响应同源（即软 404 / 通配符响应）。
type DynamicContentParser struct {
	staticPatterns     []string
	contents           []string
	normalizedContents []string
	baseContent        string
	isStatic           bool
}

// NewDynamicContentParser 使用两份通配符样本初始化解析器。
func NewDynamicContentParser(content1, content2 string) *DynamicContentParser {
	p := &DynamicContentParser{
		contents:           []string{content1, content2},
		normalizedContents: []string{NormalizeDynamicContent(content1), NormalizeDynamicContent(content2)},
		baseContent:        content1,
	}
	p.recalculate()
	return p
}

// StaticPatterns 返回学习到的静态模式（公共词）。
func (p *DynamicContentParser) StaticPatterns() []string {
	if p.staticPatterns == nil {
		return nil
	}
	return p.staticPatterns
}

// StaticPatternCount 返回静态模式的数量。
func (p *DynamicContentParser) StaticPatternCount() int {
	return len(p.staticPatterns)
}

// IsAmbiguous 判断通配符响应是否"模糊"（需要更多校准样本）。
func (p *DynamicContentParser) IsAmbiguous() bool {
	if p.isStatic {
		return false
	}
	if len(p.staticPatterns) < 8 {
		return true
	}
	last := len(p.contents) - 1
	return p.SimilarityTo(p.contents[last], p.normalizedContents[last]) < 0.55
}

// AddSample 追加校准样本并重新计算静态模式。
func (p *DynamicContentParser) AddSample(content string) {
	p.contents = append(p.contents, content)
	p.normalizedContents = append(p.normalizedContents, NormalizeDynamicContent(content))
	p.recalculate()
}

// CompareTo 判断响应内容是否匹配通配符特征。
// 语义与 dirsearch 的 DynamicContentParser.compare_to 一致：
//  1. 静态响应直接比较原文；
//  2. 否则检查响应包含全部静态模式（>=20 个模式时允许漏掉 1 个）；
//  3. 静态模式不可靠时退回相似度比较（> 0.75）。
func (p *DynamicContentParser) CompareTo(content string) bool {
	normalized := NormalizeDynamicContent(content)

	if p.isStatic {
		return content == p.baseContent || normalized == p.normalizedContents[0]
	}

	words := strings.Fields(normalized)
	// 对应 Python 的顺序查找（splitted.index(pattern, i+1)）：静态模式须按顺序出现
	i := -1
	misses := 0
	for _, pattern := range p.staticPatterns {
		idx := indexOfFrom(words, pattern, i+1)
		if idx == -1 {
			// 允许漏掉一个模式，但要求模式足够多（见 dirsearch issue #1279）
			if misses > 0 || len(p.staticPatterns) < 20 {
				return false
			}
			misses++
			continue
		}
		i = idx
	}

	if len(strings.Fields(content)) > len(strings.Fields(p.baseContent)) && len(p.staticPatterns) < 20 {
		return p.SimilarityTo(content, normalized) > 0.75
	}
	return true
}

// indexOfFrom 从 from 下标开始查找 pattern 在 words 中的位置，未找到返回 -1。
func indexOfFrom(words []string, pattern string, from int) int {
	for i := from; i < len(words); i++ {
		if words[i] == pattern {
			return i
		}
	}
	return -1
}

// SimilarityTo 计算响应与第一份通配符样本的相似度；
// normalizedContent 可传入已归一化的内容以避免重复计算（传空则内部归一化）。
func (p *DynamicContentParser) SimilarityTo(content, normalizedContent string) float64 {
	normalized := normalizedContent
	if normalized == "" {
		normalized = NormalizeDynamicContent(content)
	}
	return NormalizedContentSimilarity(p.normalizedContents[0], normalized)
}

// recalculate 重新计算静态响应标记与静态模式集合。
func (p *DynamicContentParser) recalculate() {
	p.isStatic = true
	for _, content := range p.contents {
		if content != p.baseContent {
			p.isStatic = false
			break
		}
	}
	if p.isStatic {
		p.staticPatterns = nil
		return
	}

	// 近似 difflib.Differ：取两份样本的公共词（按第一份的顺序与次数）
	first := strings.Fields(p.normalizedContents[0])
	second := strings.Fields(p.normalizedContents[1])
	counts := make(map[string]int, len(second))
	for _, w := range second {
		counts[w]++
	}
	var patterns []string
	for _, w := range first {
		if counts[w] > 0 {
			counts[w]--
			patterns = append(patterns, w)
		}
	}
	// 后续校准样本必须包含全部模式
	for _, normalized := range p.normalizedContents[2:] {
		words := strings.Fields(normalized)
		set := make(map[string]int, len(words))
		for _, w := range words {
			set[w]++
		}
		var kept []string
		for _, pattern := range patterns {
			if set[pattern] > 0 {
				set[pattern]--
				kept = append(kept, pattern)
			}
		}
		patterns = kept
	}
	p.staticPatterns = patterns
}

// GenerateMatchingRegex 由两个通配符重定向生成能同时匹配它们的正则，
// 语义与 dirsearch 的 generate_matching_regex 一致：^公共前缀.*公共后缀$。
func GenerateMatchingRegex(string1, string2 string) string {
	start := "^"
	end := "$"

	r1 := []rune(string1)
	r2 := []rune(string2)
	broke := false
	for i := 0; i < len(r1) && i < len(r2); i++ {
		if r1[i] != r2[i] {
			start += ".*"
			broke = true
			break
		}
		start += regexp.QuoteMeta(string(r1[i]))
	}
	if broke {
		for i := 0; i < len(r1) && i < len(r2); i++ {
			c1 := r1[len(r1)-1-i]
			c2 := r2[len(r2)-1-i]
			if c1 != c2 {
				break
			}
			end = regexp.QuoteMeta(string(c1)) + end
		}
	}
	return start + end
}
