package controller

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"hidir/internal/httpx"
	"hidir/internal/report"
	"hidir/internal/scanner"
	"hidir/internal/settings"
	"hidir/internal/utils"
	"hidir/internal/wordlist"
)

// Run 执行完整扫描流程（多目标循环）。
func (c *Controller) Run() error {
	if err := c.Setup(); err != nil {
		return err
	}

	// Ctrl+C 处理
	sigChan := make(chan os.Signal, 1)
	c.sigChan = sigChan
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for range sigChan {
			c.HandlePause()
		}
	}()

	// 进度条刷新信号
	progressDone := make(chan struct{})
	go c.progressLoop(progressDone)

	// 恢复会话时打印历史输出
	if len(c.OutputHistory) > 0 {
		c.UI.NewLine(strings.TrimRight(c.formatOutputHistory(), "\n"))
	}

	for len(c.Options.URLs) > 0 {
		target := c.Options.URLs[0]
		err := c.scanTarget(target)

		switch e := err.(type) {
		case *Interrupt:
			c.directories = nil
			c.dictionary.Reset()
			if e.Type == InterruptQuit {
				c.finishAll()
				c.UI.Error(e.Message)
				return nil
			}
			if e.Message != "" {
				c.UI.Error(e.Message)
			}
		case error:
			c.directories = nil
			c.dictionary.Reset()
			c.UI.Error(e.Error())
		}

		// 移除已处理目标
		c.Options.URLs = c.Options.URLs[1:]
	}

	c.finishAll()
	return nil
}

// finishAll 输出完成信息并收尾报告与会话。
func (c *Controller) finishAll() {
	c.UI.Warning("\nTask Completed", true)
	if c.reporter != nil {
		c.reporter.Finish()
	}
	if c.SessionFile != "" && !c.SessionNew {
		_ = os.Remove(c.SessionFile)
	}
}

// scanTarget 对单个目标执行全部扫描任务。
func (c *Controller) scanTarget(target string) error {
	if err := c.SetTarget(target); err != nil {
		return err
	}

	// 初始目录：基础路径 + 子目录
	if len(c.directories) == 0 {
		for _, subdir := range c.Options.Subdirs {
			c.addDirectory(c.basePath + subdir)
		}
	}

	if !c.OldSession {
		c.UI.Target(c.targetURL)
	}
	if c.reporter != nil {
		if err := c.reporter.Prepare(c.targetURL); err != nil {
			return err
		}
	}

	c.crawlTarget()
	c.targetStartTime = time.Now()
	return c.scanDirectories()
}

// scanDirectories 依次扫描目录队列中的全部任务。
func (c *Controller) scanDirectories() error {
	for len(c.directories) > 0 {
		current := c.directories[0]

		if !c.OldSession {
			currentTime := time.Now().Format("15:04:05")
			c.UI.Warning(fmt.Sprintf("\n[%s] Scanning: %s", currentTime, current), true)
		}

		if err := c.runJob(current); err != nil {
			if interrupt, ok := err.(*Interrupt); ok {
				if interrupt.Type == InterruptNextDir {
					// 仅跳出当前目录任务
				} else {
					// Quit / SkipTarget
					c.dictionary.Reset()
					c.directories = c.directories[1:]
					c.jobsProcessed++
					c.OldSession = false
					return interrupt
				}
			} else {
				return err
			}
		}

		c.dictionary.Reset()
		c.directories = c.directories[1:]
		c.jobsProcessed++
		c.OldSession = false
	}
	return nil
}

// runJob 是 scanDirectories 的实际执行体。
func (c *Controller) runJob(directory string) error {
	c.dictionary.Reset()
	c.fuzzer = scanner.NewFuzzer(c.requester, c.dictionary, c.Options, c.blacklists,
		scanner.Callbacks{
			Match:    c.MatchCallback,
			NotFound: c.NotFoundCallback,
			Error:    c.ErrorCallback,
		}, c.delay())
	c.fuzzer.SetBasePath(directory)

	// 词表总数（含动态追加前的规模）
	total := c.dictionary.Len()
	_ = total

	done := make(chan struct{})
	go func() {
		defer close(done)
		c.fuzzer.Start()
	}()

	// 恢复扫描（若处于暂停状态）
	c.fuzzer.Play()

	for {
		select {
		case <-done:
			// 校准失败（无法连接目标）时上报错误
			if c.fuzzer.SetupErr != nil {
				return c.fuzzer.SetupErr
			}
			// 任务结束后仍需检查回调中产生的中断（如 skip-on-status）
			c.stateMutex.Lock()
			pending := c.currentInterrupt
			c.stateMutex.Unlock()
			if pending != nil {
				return pending
			}
			return nil
		case <-time.After(200 * time.Millisecond):
		}

		// 检查暂停菜单产生的中断
		c.stateMutex.Lock()
		pending := c.currentInterrupt
		c.stateMutex.Unlock()
		if pending != nil {
			c.fuzzer.Quit()
			<-done
			return pending
		}

		// 时间限制检查
		if interrupt := c.checkTimeLimits(); interrupt != nil {
			c.fuzzer.Quit()
			<-done
			return interrupt
		}
	}
}

// delay 返回请求间延迟秒数。
func (c *Controller) delay() float64 {
	if c.Options.Delay != nil {
		return *c.Options.Delay
	}
	return 0
}

// checkTimeLimits 检查 --max-time 与 --target-max-time。
func (c *Controller) checkTimeLimits() *Interrupt {
	now := time.Now()
	if c.Options.MaxTime != nil && *c.Options.MaxTime > 0 {
		if now.Sub(c.startTime) > time.Duration(*c.Options.MaxTime)*time.Second {
			return &Interrupt{Type: InterruptQuit,
				Message: "Runtime exceeded the maximum set by the user"}
		}
	}
	if c.Options.TargetMaxTime != nil && *c.Options.TargetMaxTime > 0 {
		if now.Sub(c.targetStartTime) > time.Duration(*c.Options.TargetMaxTime)*time.Second {
			return &Interrupt{Type: InterruptSkipTarget,
				Message: "Runtime for target exceeded the maximum set by the user"}
		}
	}
	return nil
}

// crawlTarget 抓取目标根路径以发现初始路径列表。
func (c *Controller) crawlTarget() {
	if !c.Options.Crawl {
		return
	}
	response, err := c.requester.Do(c.basePath, "")
	if err != nil {
		c.ErrorCallback(err)
		return
	}
	c.addCrawledPaths(response)
}

// progressLoop 定期刷新进度条。
func (c *Controller) progressLoop(done chan struct{}) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			c.updateProgress()
		}
	}
}

// updateProgress 刷新进度条显示。
func (c *Controller) updateProgress() {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	// 交互菜单（暂停）期间冻结刷新，避免进度条擦除菜单提示
	if c.progressHold.Load() {
		return
	}
	if c.fuzzer == nil || c.dictionary == nil {
		return
	}
	jobsCount := len(c.Options.Subdirs)*(len(c.Options.URLs)-1) +
		len(c.directories) + c.jobsProcessed
	c.UI.LastPath(c.dictionary.Index(), c.dictionary.Len(),
		c.jobsProcessed+1, jobsCount, c.requester.Rate(), c.errors)
}

// MatchCallback 处理命中响应（状态报告、递归、重放、爬取与备份探测）。
func (c *Controller) MatchCallback(response *httpx.Response) {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()

	// skip-on-status：跳过目标
	if c.Options.SkipOnStatus.Contains(response.Status) {
		c.setInterrupt(&Interrupt{Type: InterruptSkipTarget,
			Message: fmt.Sprintf("Skipped the target due to %d status code", response.Status)})
		return
	}

	fullURL := c.Options.FullURL || c.Options.Quiet
	c.UI.StatusReport(response, fullURL, c.Options.Verbose)

	// 保存结果到报告
	if c.reporter != nil {
		c.reporter.Save(reportResult(response))
	}

	// 递归
	if c.Options.RecursionStatus.Contains(response.Status) &&
		(c.Options.Recursive || c.Options.DeepRecursive || c.Options.ForceRecursive) {
		var added []string
		if response.Redirect != "" {
			newPath := utils.SameOriginPath(response.URL, response.Redirect)
			if newPath != "" {
				added = c.recurForRedirect(
					utils.CleanPath(response.Path, false, false),
					utils.CleanPath(newPath, false, false))
			}
		} else if len(response.History) > 0 {
			oldPath := utils.CleanPath(utils.ParsePath(response.History[0]), false, false)
			added = c.recurForRedirect(oldPath, utils.CleanPath(response.Path, false, false))
		} else {
			added = c.recur(utils.CleanPath(response.Path, false, false))
		}
		if len(added) > 0 {
			c.UI.NewDirectories(added)
		}
	}

	// 重放代理
	if c.Options.ReplayProxy != "" {
		go func() {
			_, _ = c.requester.Do(response.FullPath, c.Options.ReplayProxy)
		}()
	}

	// 爬取响应中的新路径
	if c.Options.Crawl {
		c.addCrawledPaths(response)
	}

	// 备份文件探测
	if c.Options.FindBackup {
		path := utils.LstripOnce(response.Path, c.basePath)
		for _, backupPath := range wordlist.GenerateBackupPaths(path) {
			c.dictionary.AddExtra(backupPath, c.Options.ExcludeExtensions)
		}
	}

	c.consecutiveErrors = 0
}

// NotFoundCallback 处理被过滤的响应（进度与错误计数复位）。
func (c *Controller) NotFoundCallback(response *httpx.Response) {
	c.stateMutex.Lock()
	c.consecutiveErrors = 0
	c.stateMutex.Unlock()
}

// ErrorCallback 处理请求错误。
func (c *Controller) ErrorCallback(err error) {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()

	if c.Options.ExitOnError {
		c.setInterrupt(&Interrupt{Type: InterruptQuit, Message: "Canceled due to an error"})
		return
	}
	c.errors++
	c.consecutiveErrors++
	if c.consecutiveErrors > settings.MaxConsecutiveRequestErrors {
		c.setInterrupt(&Interrupt{Type: InterruptSkipTarget, Message: "Too many request errors"})
	}
}

// setInterrupt 记录待处理的中断（调用方需持有 stateMutex）。
func (c *Controller) setInterrupt(interrupt *Interrupt) {
	if c.currentInterrupt == nil {
		c.currentInterrupt = interrupt
	}
	if c.fuzzer != nil {
		c.fuzzer.Quit()
	}
}

// reportResult 将响应转换为报告条目。
func reportResult(response *httpx.Response) report.Result {
	return report.Result{
		Datetime: response.Datetime,
		URL:      response.URL,
		Status:   response.Status,
		Length:   response.Length(),
		Type:     response.Type(),
		Redirect: response.Redirect,
		Elapsed:  response.Elapsed,
	}
}

// addDirectory 将目录加入递归队列（检查深度/排除/重复）。
func (c *Controller) addDirectory(path string) {
	// 排除子目录
	if c.isExcludedSubdir(path) {
		return
	}
	targetURL := c.targetURL + path
	depth := strings.Count(path, "/") - strings.Count(c.basePath, "/")
	if c.Options.RecursionDepth != nil && *c.Options.RecursionDepth > 0 && depth > *c.Options.RecursionDepth {
		return
	}
	if c.passedUrls[targetURL] {
		return
	}
	c.directories = append(c.directories, path)
	c.passedUrls[targetURL] = true
}

// isExcludedSubdir 判断路径是否命中排除子目录。
func (c *Controller) isExcludedSubdir(path string) bool {
	resourcePath := path
	if idx := strings.Index(path, "?"); idx >= 0 {
		resourcePath = path[:idx]
	}
	resourcePath = strings.TrimPrefix(resourcePath, "/")
	for _, subdir := range c.Options.ExcludeSubdirs {
		if strings.HasPrefix(resourcePath, subdir) || strings.Contains(resourcePath, "/"+subdir) {
			return true
		}
	}
	return false
}

// recur 依据递归选项将路径加入队列，返回新增目录。
func (c *Controller) recur(path string) []string {
	count := len(c.directories)
	path = utils.CleanPath(path, false, false)

	if c.Options.ForceRecursive && !strings.HasSuffix(path, "/") {
		path += "/"
	}

	if c.Options.DeepRecursive {
		// 每一层目录深度都入队
		i := 0
		depth := strings.Count(path, "/")
		for n := 0; n < depth; n++ {
			idx := strings.Index(path[i:], "/")
			if idx < 0 {
				break
			}
			i += idx + 1
			c.addDirectory(path[:i])
		}
	} else if c.Options.Recursive && strings.HasSuffix(path, "/") &&
		!hasExtensionRecognition(path[:len(path)-1]) {
		c.addDirectory(path)
	}
	return c.directories[count:]
}

// recurForRedirect 处理跳转目标的递归（仅同目录加深一层）。
func (c *Controller) recurForRedirect(path, redirectPath string) []string {
	if redirectPath == path+"/" {
		return c.recur(redirectPath)
	}
	return nil
}

// addCrawledPaths 从响应内容爬取同源路径并加入词表。
func (c *Controller) addCrawledPaths(response *httpx.Response) {
	paths := crawlResponse(response)
	for _, path := range paths {
		path = utils.LstripOnce(path, c.basePath)
		if !c.isExcludedSubdir(path) {
			c.dictionary.AddExtra(path, c.Options.ExcludeExtensions)
		}
	}
}

// crawlResponse 封装爬虫调用。
func crawlResponse(response *httpx.Response) []string {
	return utils.Crawl(response.URL, response.FullPath, response.Type(), response.Content)
}

// formatOutputHistory 渲染历史输出。
func (c *Controller) formatOutputHistory() string {
	var parts []string
	for _, entry := range c.OutputHistory {
		if entry.Output == "" {
			continue
		}
		if entry.StartTime != "" {
			parts = append(parts, "--- Previous run started: "+entry.StartTime+" ---")
		} else {
			parts = append(parts, "--- Previous run ---")
		}
		parts = append(parts, strings.TrimRight(entry.Output, "\n"))
	}
	return strings.Join(parts, "\n")
}

// hasExtensionRecognition 判断路径是否以已知扩展名结尾。
func hasExtensionRecognition(path string) bool {
	return extensionRegex.MatchString(path)
}
