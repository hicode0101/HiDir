package scanner

import (
	"regexp"
	"strings"
	"time"

	"hidir/internal/httpx"
	"hidir/internal/settings"
	"hidir/internal/utils"
)

// TestedRegistry 保存已创建的通配符扫描器（按类别分组），
// 用于跨扫描器复用校准结果。
type TestedRegistry struct {
	// Categories 包含 default/prefixes/suffixes 三类。
	Categories map[string]map[string]*WildcardScanner
}

// NewTestedRegistry 创建空注册表。
func NewTestedRegistry() *TestedRegistry {
	return &TestedRegistry{Categories: map[string]map[string]*WildcardScanner{
		"default":  {},
		"prefixes": {},
		"suffixes": {},
	}}
}

// FindDuplicate 查找与响应等价的既有扫描器（复用其校准结果）。
func (t *TestedRegistry) FindDuplicate(response *httpx.Response) *WildcardScanner {
	for _, category := range t.Categories {
		for _, tester := range category {
			if response.EqualResponse(tester.Response) {
				return tester
			}
		}
	}
	return nil
}

// WildcardScanner 通过随机路径采样学习目标的通配符（软 404）特征。
// 对应 dirsearch 的 BaseScanner/Scanner。
type WildcardScanner struct {
	// Path 是含替换点标记的路径模板。
	Path string
	// ContextName 是扫描器上下文描述（用于日志）。
	ContextName string
	// Requester 是 HTTP 客户端。
	Requester *httpx.Requester
	// Tested 是共享的扫描器注册表。
	Tested *TestedRegistry

	// Response 是第一份通配符响应。
	Response *httpx.Response
	// ContentParser 是动态内容解析器。
	ContentParser *utils.DynamicContentParser
	// WildcardRedirectRegex 是通配符重定向匹配正则。
	WildcardRedirectRegex string
	// SampleCount 是已采样的请求数。
	SampleCount int
	// Reason 是最近一次判定的原因。
	Reason string

	// Delay 是每次采样请求之间的延迟秒数。
	Delay float64
	// AutoCalibration 是否强制额外校准。
	AutoCalibration bool

	stealthGen *StealthWordGenerator
}

// NewWildcardScanner 创建并完成校准的扫描器；校准请求失败时返回错误。
func NewWildcardScanner(requester *httpx.Requester, path, contextName string,
	tested *TestedRegistry, delay float64, autoCalibration bool) (*WildcardScanner, error) {
	s := &WildcardScanner{
		Path:            path,
		ContextName:     contextName,
		Requester:       requester,
		Tested:          tested,
		Delay:           delay,
		AutoCalibration: autoCalibration,
		stealthGen:      NewStealthWordGenerator(),
	}
	if err := s.Setup(); err != nil {
		return nil, err
	}
	return s, nil
}

// Setup 执行两次（或多次）随机路径采样以学习通配符特征。
func (s *WildcardScanner) Setup() error {
	firstPath := strings.ReplaceAll(s.Path, settings.WildcardTestPointMarker, s.stealthGen.Generate(nil))
	firstResponse, err := s.Requester.Do(firstPath, "")
	if err != nil {
		return err
	}
	s.Response = firstResponse
	s.SampleCount = 1
	time.Sleep(time.Duration(s.Delay * float64(time.Second)))

	// 若与既有扫描器的响应等价，直接复用其校准结果
	if duplicate := s.Tested.FindDuplicate(firstResponse); duplicate != nil {
		s.ContentParser = duplicate.ContentParser
		s.WildcardRedirectRegex = duplicate.WildcardRedirectRegex
		return nil
	}

	secondPath := strings.ReplaceAll(s.Path, settings.WildcardTestPointMarker,
		s.stealthGen.Generate(map[string]bool{firstPath: true}))
	secondResponse, err := s.Requester.Do(secondPath, "")
	if err != nil {
		return err
	}
	s.SampleCount++
	time.Sleep(time.Duration(s.Delay * float64(time.Second)))

	if firstResponse.Redirect != "" && secondResponse.Redirect != "" {
		s.WildcardRedirectRegex = GenerateRedirectRegex(
			utils.CleanPath(firstResponse.Redirect, false, false), firstPath,
			utils.CleanPath(secondResponse.Redirect, false, false), secondPath)
	}

	s.ContentParser = utils.NewDynamicContentParser(firstResponse.Content, secondResponse.Content)
	s.AutoCalibrate(map[string]bool{firstPath: true, secondPath: true})
	return nil
}

// AutoCalibrate 在响应模糊时追加额外采样。
func (s *WildcardScanner) AutoCalibrate(omitted map[string]bool) {
	if !s.AutoCalibration && (s.ContentParser == nil || !s.ContentParser.IsAmbiguous()) {
		return
	}
	for i := 0; i < settings.AutoCalibrationExtraSamples; i++ {
		samplePath := strings.ReplaceAll(s.Path, settings.WildcardTestPointMarker,
			s.stealthGen.Generate(omitted))
		omitted[samplePath] = true
		sampleResponse, err := s.Requester.Do(samplePath, "")
		if err != nil {
			continue
		}
		s.SampleCount++
		if sampleResponse.Content != "" && s.ContentParser != nil {
			s.ContentParser.AddSample(sampleResponse.Content)
		}
		time.Sleep(time.Duration(s.Delay * float64(time.Second)))
	}
}

// Classify 判定响应类型："unique"（独立响应）或 "wildcard"（通配符响应）。
func (s *WildcardScanner) Classify(path string, response *httpx.Response) string {
	if s.Response.Status != response.Status {
		s.Reason = "status differs from wildcard profile"
		return "unique"
	}

	// 检查重定向是否符合通配符重定向模式
	if s.WildcardRedirectRegex != "" && response.Redirect != "" {
		redirect := replaceWithMarker(
			utils.CleanPath(response.Redirect, false, false),
			utils.CleanPath(path, false, false))
		re, err := regexp.Compile(s.WildcardRedirectRegex)
		if err == nil {
			if !re.MatchString(redirect) {
				s.Reason = "redirect differs from wildcard profile"
				return "unique"
			}
		}
	}

	if s.IsWildcard(response) {
		s.Reason = "matches wildcard profile"
		return "wildcard"
	}

	if s.IsProbableWildcard(path, response) {
		s.Reason = "matches ambiguous wildcard profile"
		return "wildcard"
	}

	s.Reason = "response is unique enough"
	return "unique"
}

// Check 是 Classify 的便捷封装：true 表示通过（非通配符）。
func (s *WildcardScanner) Check(path string, response *httpx.Response) bool {
	return s.Classify(path, response) != "wildcard"
}

// IsWildcard 判断响应是否与通配符响应相似。
func (s *WildcardScanner) IsWildcard(response *httpx.Response) bool {
	// 两个二进制响应直接比较响应体
	if s.Response.Content == "" && response.Content == "" {
		return s.Response.HasSameBody(response)
	}
	if s.ContentParser == nil {
		return false
	}
	return s.ContentParser.CompareTo(response.Content)
}

// IsProbableWildcard 是动态软 404 模板的保守回退判定。
func (s *WildcardScanner) IsProbableWildcard(path string, response *httpx.Response) bool {
	if s.Response.Content == "" || response.Content == "" {
		return false
	}
	if s.Response.Type() != response.Type() {
		return false
	}
	if s.Response.Redirect != "" && response.Redirect == "" {
		return false
	}
	if s.ContentParser != nil && s.ContentParser.StaticPatternCount() >= 20 {
		return false
	}
	if len(s.Response.Body) > settings.AmbiguousSimilarityMaxContentLength ||
		len(response.Body) > settings.AmbiguousSimilarityMaxContentLength {
		return false
	}

	similarity := s.ContentParser.SimilarityTo(response.Content, "")
	if similarity < settings.AmbiguousSimilarityThreshold {
		return false
	}

	baseLength := s.Response.Length()
	if baseLength < 1 {
		baseLength = 1
	}
	lengthDelta := float64(abs64(s.Response.Length()-response.Length())) / float64(baseLength)
	if lengthDelta > 0.35 {
		return false
	}

	normalizedContent := response.NormalizedContent()
	normalizedPath := strings.Trim(utils.CleanPath(path, false, false), "/")
	if normalizedPath != "" && strings.Contains(normalizedContent, normalizedPath) {
		return true
	}
	return s.ContentParser.IsAmbiguous()
}

// GenerateRedirectRegex 由两个通配符重定向生成匹配正则。
func GenerateRedirectRegex(firstLoc, firstPath, secondLoc, secondPath string) string {
	if firstPath != "" {
		firstLoc = strings.ReplaceAll(firstLoc, "/"+firstPath, settings.ReflectedPathMarker)
	}
	if secondPath != "" {
		secondLoc = strings.ReplaceAll(secondLoc, "/"+secondPath, settings.ReflectedPathMarker)
	}
	return utils.GenerateMatchingRegex(firstLoc, secondLoc)
}

// replaceWithMarker 将响应中的路径替换为标记。
func replaceWithMarker(content, path string) string {
	if path == "" {
		return content
	}
	return strings.ReplaceAll(content, "/"+path, settings.ReflectedPathMarker)
}

// abs64 返回 int64 绝对值。
func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
