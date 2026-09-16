package scanner

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"hidir/internal/filters"
	"hidir/internal/httpx"
	"hidir/internal/options"
	"hidir/internal/settings"
	"hidir/internal/wordlist"
)

// Callbacks 汇总扫描回调。
type Callbacks struct {
	// Match 在响应命中（通过全部过滤器）时调用。
	Match func(*httpx.Response)
	// NotFound 在响应被过滤时调用（用于进度更新）。
	NotFound func(*httpx.Response)
	// Error 在请求出错时调用。
	Error func(error)
}

// Fuzzer 是单目录扫描的调度器：并发领取词条、请求、过滤与回调。
type Fuzzer struct {
	// Requester 是 HTTP 客户端。
	Requester *httpx.Requester
	// Dictionary 是词表。
	Dictionary *wordlist.Dictionary
	// Options 是扫描选项。
	Options *options.Options
	// Blacklists 是状态码黑名单词表。
	Blacklists wordlist.Blacklists
	// BasePath 是当前扫描目录（不含开头斜杠）。
	BasePath string
	// Callbacks 是响应回调。
	Callbacks Callbacks
	// Delay 是每次请求之间的延迟秒数。
	Delay float64

	// Tested 是通配符扫描器注册表。
	Tested *TestedRegistry

	filterFingerprints  map[string]int
	similarFingerprints map[string]int
	autoCalibrated      map[string]bool
	fingerprintMutex    sync.Mutex
	scannersMutex       sync.Mutex
	quitChan            chan struct{}
	workerWG            sync.WaitGroup
	SetupErr            error

	pauseMutex    sync.Mutex
	paused        bool
	resumeChan    chan struct{}
	pausedWG      *sync.WaitGroup
	activeWorkers int
}

// NewFuzzer 创建扫描调度器。
func NewFuzzer(requester *httpx.Requester, dictionary *wordlist.Dictionary,
	opts *options.Options, blacklists wordlist.Blacklists, callbacks Callbacks, delay float64) *Fuzzer {
	return &Fuzzer{
		Requester:           requester,
		Dictionary:          dictionary,
		Options:             opts,
		Blacklists:          blacklists,
		Callbacks:           callbacks,
		Delay:               delay,
		Tested:              NewTestedRegistry(),
		filterFingerprints:  map[string]int{},
		similarFingerprints: map[string]int{},
		autoCalibrated:      map[string]bool{},
		quitChan:            make(chan struct{}),
	}
}

// SetBasePath 设置扫描目录。
func (f *Fuzzer) SetBasePath(path string) { f.BasePath = path }

// IsExcluded 校验响应是否应被过滤。语义与 dirsearch 的 is_excluded 一致。
func (f *Fuzzer) IsExcluded(resp *httpx.Response) bool {
	opts := f.Options

	if opts.ExcludeStatus.Contains(resp.Status) {
		return true
	}
	if opts.IncludeStatus.Len() > 0 && !opts.IncludeStatus.Contains(resp.Status) {
		return true
	}

	// 状态码黑名单词表
	if suffixes, ok := f.Blacklists[resp.Status]; ok {
		for _, suffix := range suffixes {
			if strings.HasSuffix(resp.Path, strings.TrimPrefix(suffix, "/")) {
				return true
			}
		}
	}

	if _, excluded := opts.ExcludeSizes[resp.Length()]; excluded {
		return true
	}
	if resp.Length() < opts.MinResponseSize {
		return true
	}
	if resp.Length() > opts.MaxResponseSize && opts.MaxResponseSize > 0 {
		return true
	}

	content := resp.Content
	for _, text := range opts.ExcludeTexts {
		if text != "" && strings.Contains(content, text) {
			return true
		}
	}

	if opts.ExcludeRegex != "" {
		if re, err := regexp.Compile(opts.ExcludeRegex); err == nil {
			if re.MatchString(content) {
				return true
			}
		}
	}

	if opts.ExcludeRedirect != "" {
		if strings.Contains(resp.Redirect, opts.ExcludeRedirect) {
			return true
		}
		if re, err := regexp.Compile(opts.ExcludeRedirect); err == nil && re.MatchString(resp.Redirect) {
			return true
		}
	}

	if !f.MatchesAdvancedMatchers(resp) {
		return true
	}
	if f.MatchesAdvancedFilters(resp) {
		return true
	}
	if f.IsAutoCalibrated(resp) {
		return true
	}

	if *getFilterThreshold(opts) > 0 {
		f.fingerprintMutex.Lock()
		count := f.filterFingerprints[resp.FilterFingerprint()]
		f.fingerprintMutex.Unlock()
		if count >= *getFilterThreshold(opts) {
			return true
		}
	}

	return false
}

// getFilterThreshold 返回过滤阈值指针。
func getFilterThreshold(opts *options.Options) *int {
	if opts.FilterThreshold == nil {
		zero := 0
		return &zero
	}
	return opts.FilterThreshold
}

// headerMatchesText 判断响应头文本是否包含任一模式（大小写不敏感）。
func headerMatchesText(resp *httpx.Response, patterns []string) bool {
	headers := strings.ToLower(resp.HeadersText())
	for _, pattern := range patterns {
		if strings.Contains(headers, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// headerMatchesRegex 判断响应头文本是否匹配正则（大小写不敏感）。
func headerMatchesRegex(resp *httpx.Response, pattern string) bool {
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return false
	}
	return re.MatchString(resp.HeadersText())
}

// compilePattern 安全编译正则，失败返回 nil。
func compilePattern(pattern string) *regexp.Regexp {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	return re
}

// MatchesAdvancedMatchers 评估高级匹配器（and/or 模式）。
func (f *Fuzzer) MatchesAdvancedMatchers(resp *httpx.Response) bool {
	opts := f.Options
	var checks []bool

	if opts.MatchStatus.Len() > 0 {
		checks = append(checks, opts.MatchStatus.Contains(resp.Status))
	}
	if len(opts.MatchSizes) > 0 {
		checks = append(checks, filters.MatchesNumericRanges(resp.Length(), opts.MatchSizes))
	}
	if len(opts.MatchWords) > 0 {
		checks = append(checks, filters.MatchesNumericRanges(int64(resp.Words()), opts.MatchWords))
	}
	if len(opts.MatchLines) > 0 {
		checks = append(checks, filters.MatchesNumericRanges(int64(resp.Lines()), opts.MatchLines))
	}
	if opts.MatchRegex != "" {
		re := compilePattern(opts.MatchRegex)
		checks = append(checks, re != nil && re.MatchString(resp.Text()))
	}
	if len(opts.MatchHeaders) > 0 {
		checks = append(checks, headerMatchesText(resp, opts.MatchHeaders))
	}
	if opts.MatchHeaderRegex != "" {
		checks = append(checks, headerMatchesRegex(resp, opts.MatchHeaderRegex))
	}
	if len(opts.MatchTime) > 0 {
		checks = append(checks, filters.MatchesTimeFilters(resp.Elapsed, opts.MatchTime))
	}

	return combineChecks(checks, opts.MatcherMode, true)
}

// MatchesAdvancedFilters 评估高级过滤器（and/or 模式）。
func (f *Fuzzer) MatchesAdvancedFilters(resp *httpx.Response) bool {
	opts := f.Options
	var checks []bool

	if opts.FilterStatus.Len() > 0 {
		checks = append(checks, opts.FilterStatus.Contains(resp.Status))
	}
	if len(opts.FilterSizes) > 0 {
		checks = append(checks, filters.MatchesNumericRanges(resp.Length(), opts.FilterSizes))
	}
	if len(opts.FilterWords) > 0 {
		checks = append(checks, filters.MatchesNumericRanges(int64(resp.Words()), opts.FilterWords))
	}
	if len(opts.FilterLines) > 0 {
		checks = append(checks, filters.MatchesNumericRanges(int64(resp.Lines()), opts.FilterLines))
	}
	if opts.FilterRegex != "" {
		re := compilePattern(opts.FilterRegex)
		checks = append(checks, re != nil && re.MatchString(resp.Text()))
	}
	if len(opts.FilterHeaders) > 0 {
		checks = append(checks, headerMatchesText(resp, opts.FilterHeaders))
	}
	if opts.FilterHeaderRegex != "" {
		checks = append(checks, headerMatchesRegex(resp, opts.FilterHeaderRegex))
	}
	if len(opts.FilterTime) > 0 {
		checks = append(checks, filters.MatchesTimeFilters(resp.Elapsed, opts.FilterTime))
	}

	return combineChecks(checks, opts.FilterMode, false)
}

// combineChecks 以 and/or 模式组合布尔检查。
func combineChecks(checks []bool, mode string, emptyDefault bool) bool {
	if len(checks) == 0 {
		return emptyDefault
	}
	if mode == "and" {
		for _, c := range checks {
			if !c {
				return false
			}
		}
		return true
	}
	for _, c := range checks {
		if c {
			return true
		}
	}
	return false
}

// HasAdvancedMatchers 判断是否配置了高级匹配器。
func (f *Fuzzer) HasAdvancedMatchers() bool {
	opts := f.Options
	return opts.MatchStatus.Len() > 0 || len(opts.MatchSizes) > 0 ||
		len(opts.MatchWords) > 0 || len(opts.MatchLines) > 0 ||
		opts.MatchRegex != "" || len(opts.MatchHeaders) > 0 ||
		opts.MatchHeaderRegex != "" || len(opts.MatchTime) > 0
}

// ShouldRecordAutoCalibration 判断响应是否参与自动校准统计。
func (f *Fuzzer) ShouldRecordAutoCalibration(resp *httpx.Response) bool {
	if f.HasAdvancedMatchers() {
		return false
	}
	if resp.Length() < settings.AutoCalibrationMinContentLength {
		return false
	}
	if f.Options.AutoCalibration {
		return true
	}
	if resp.Status >= 400 && resp.Status <= 599 {
		return true
	}
	path := strings.Trim(utilsCleanPath(resp.FullPath), "/")
	if path != "" && strings.Contains(resp.Text(), path) {
		return true
	}
	return resp.Redirect != ""
}

// IsAutoCalibrated 判断响应指纹是否已被自动校准过滤。
func (f *Fuzzer) IsAutoCalibrated(resp *httpx.Response) bool {
	fingerprint := resp.ResponseFingerprint()

	f.fingerprintMutex.Lock()
	defer f.fingerprintMutex.Unlock()

	if f.autoCalibrated[fingerprint] {
		return true
	}
	if !f.ShouldRecordAutoCalibration(resp) {
		return false
	}

	f.similarFingerprints[fingerprint]++
	threshold := settings.AutoCalibrationDuplicateThreshold
	if f.Options.AutoCalibration {
		threshold = settings.AutoCalibrationForcedThreshold
	}
	if f.similarFingerprints[fingerprint] < threshold {
		return false
	}
	f.autoCalibrated[fingerprint] = true
	return true
}

// getScannersFor 返回适用于路径的扫描器列表。
func (f *Fuzzer) getScannersFor(path string) []*WildcardScanner {
	path = utilsCleanPath(path)
	var out []*WildcardScanner

	f.scannersMutex.Lock()
	defer f.scannersMutex.Unlock()

	for prefix, tester := range f.Tested.Categories["prefixes"] {
		if strings.HasPrefix(path, prefix) {
			out = append(out, tester)
		}
	}
	for suffix, tester := range f.Tested.Categories["suffixes"] {
		if strings.HasSuffix(path, suffix) {
			out = append(out, tester)
		}
	}
	for _, tester := range f.Tested.Categories["default"] {
		out = append(out, tester)
	}
	return out
}

// ProcessResponse 处理单条响应：过滤、通配符检测与回调分发。
func (f *Fuzzer) ProcessResponse(path string, response *httpx.Response) {
	// 通配符检测
	for _, tester := range f.getScannersFor(path) {
		if !tester.Check(path, response) {
			if f.Callbacks.NotFound != nil {
				f.Callbacks.NotFound(response)
			}
			return
		}
	}

	if f.IsExcluded(response) {
		if f.Callbacks.NotFound != nil {
			f.Callbacks.NotFound(response)
		}
		return
	}

	if *getFilterThreshold(f.Options) > 0 {
		f.fingerprintMutex.Lock()
		f.filterFingerprints[response.FilterFingerprint()]++
		f.fingerprintMutex.Unlock()
	}

	if f.Callbacks.Match != nil {
		f.Callbacks.Match(response)
	}
}

// SetupScanners 创建通配符扫描器（default/prefixes/suffixes/扩展名）。
// 返回错误时表示校准请求失败（无法连接目标）。
func (f *Fuzzer) SetupScanners() error {
	opts := f.Options
	basePath := f.BasePath

	defaultScanner, err := NewWildcardScanner(f.Requester,
		basePath+settings.WildcardTestPointMarker, "all cases", f.Tested,
		f.Delay, opts.AutoCalibration)
	if err != nil {
		return err
	}
	f.Tested.Categories["default"]["random"] = defaultScanner

	if opts.ExcludeResponse != "" {
		custom, err := NewWildcardScanner(f.Requester, opts.ExcludeResponse,
			"all cases", f.Tested, f.Delay, opts.AutoCalibration)
		if err != nil {
			return err
		}
		f.Tested.Categories["default"]["custom"] = custom
	}

	// 前缀扫描器（用户前缀 + 默认前缀）
	prefixSet := map[string]bool{}
	for _, prefix := range opts.Prefixes {
		prefixSet[prefix] = true
	}
	for _, prefix := range settings.DefaultTestPrefixes {
		prefixSet[prefix] = true
	}
	for prefix := range prefixSet {
		scanner, err := NewWildcardScanner(f.Requester,
			basePath+prefix+settings.WildcardTestPointMarker,
			"/"+basePath+prefix+"***", f.Tested, f.Delay, opts.AutoCalibration)
		if err != nil {
			return err
		}
		f.Tested.Categories["prefixes"][prefix] = scanner
	}

	// 后缀扫描器（用户后缀 + 默认后缀）
	suffixSet := map[string]bool{}
	for _, suffix := range opts.Suffixes {
		suffixSet[suffix] = true
	}
	for _, suffix := range settings.DefaultTestSuffixes {
		suffixSet[suffix] = true
	}
	for suffix := range suffixSet {
		scanner, err := NewWildcardScanner(f.Requester,
			basePath+settings.WildcardTestPointMarker+suffix,
			"/"+basePath+"***"+suffix, f.Tested, f.Delay, opts.AutoCalibration)
		if err != nil {
			return err
		}
		f.Tested.Categories["suffixes"][suffix] = scanner
	}

	// 扩展名后缀扫描器
	for _, extension := range opts.Extensions {
		key := "." + extension
		if _, exists := f.Tested.Categories["suffixes"][key]; exists {
			continue
		}
		scanner, err := NewWildcardScanner(f.Requester,
			basePath+settings.WildcardTestPointMarker+"."+extension,
			"/"+basePath+"***."+extension, f.Tested, f.Delay, opts.AutoCalibration)
		if err != nil {
			return err
		}
		f.Tested.Categories["suffixes"][key] = scanner
	}
	return nil
}

// Start 启动扫描（阻塞直到词表耗尽或收到退出信号）。
func (f *Fuzzer) Start() {
	if err := f.SetupScanners(); err != nil {
		f.SetupErr = err
		return
	}

	threads := f.Options.ThreadCount
	f.activeWorkers = threads
	for i := 0; i < threads; i++ {
		f.workerWG.Add(1)
		go f.workerProc()
	}
	f.workerWG.Wait()
}

// workerProc 是单个扫描工作协程。
func (f *Fuzzer) workerProc() {
	defer f.workerWG.Done()

	for {
		select {
		case <-f.quitChan:
			return
		default:
		}

		// 暂停检查
		f.pauseMutex.Lock()
		if f.paused {
			resume := f.resumeChan
			f.pausedWG.Done()
			f.pauseMutex.Unlock()
			if resume != nil {
				<-resume
			}
			continue
		}
		f.pauseMutex.Unlock()

		path, ok := f.Dictionary.ClaimNext()
		if !ok {
			return
		}
		f.scan(f.BasePath + path)
		f.Dictionary.ReleaseClaims([]string{path})

		// 请求间延迟
		if f.Delay > 0 {
			time.Sleep(time.Duration(f.Delay * float64(time.Second)))
		}
	}
}

// scan 请求路径并处理响应。
func (f *Fuzzer) scan(path string) {
	response, err := f.Requester.Do(path, "")
	if err != nil {
		if f.Callbacks.Error != nil {
			f.Callbacks.Error(err)
		}
		return
	}
	f.ProcessResponse(path, response)
}

// Pause 暂停全部工作协程并等待其确认。
// 返回是否全部协程都已暂停（超时 2 秒，避免 IO 阻塞时死锁）。
func (f *Fuzzer) Pause() bool {
	f.pauseMutex.Lock()
	if f.paused {
		f.pauseMutex.Unlock()
		return true
	}
	f.paused = true
	f.resumeChan = make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(f.activeWorkers)
	f.pausedWG = wg
	f.pauseMutex.Unlock()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

// Play 恢复全部工作协程。
func (f *Fuzzer) Play() {
	f.pauseMutex.Lock()
	defer f.pauseMutex.Unlock()
	if f.paused {
		close(f.resumeChan)
		f.paused = false
		f.pausedWG = nil
	}
}

// Quit 请求停止全部工作协程。
func (f *Fuzzer) Quit() {
	select {
	case <-f.quitChan:
	default:
		close(f.quitChan)
		f.Play()
	}
}

// utilsCleanPath 是路径清洗的本地封装。
func utilsCleanPath(path string) string {
	return strings.TrimRight(strings.Split(strings.Split(path, "#")[0], "?")[0], "")
}
