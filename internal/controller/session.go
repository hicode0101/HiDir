package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"hidir/internal/options"
	"hidir/internal/utils"
)

// extensionRegex 用于递归判定路径是否带扩展名（与 dirsearch 一致）。
var extensionRegex = regexp.MustCompile(`\w+([.][a-zA-Z0-9]{2,5}){1,3}~?$`)

// HandlePause 处理 Ctrl+C：暂停扫描并显示交互菜单。
func (c *Controller) HandlePause() {
	// 冻结进度条刷新，否则菜单会在 300ms 内被进度条擦除重绘，
	// 用户看不到选项提示（dirsearch 的进度条由响应回调驱动，
	// 暂停后自然停止刷新，因此需要显式冻结）
	c.progressHold.Store(true)
	// 与 updateProgress 互斥：确保在途的一次进度绘制已经完成
	c.stateMutex.Lock()
	c.stateMutex.Unlock()
	defer c.progressHold.Store(false)

	if c.fuzzer == nil {
		viewForceQuit()
		os.Exit(1)
	}

	c.UI.Warning("CTRL+C detected: Pausing threads, please wait...", false)

	allPaused := c.fuzzer.Pause()
	if !allPaused {
		c.UI.Warning("Could not pause all threads (some may be blocked on I/O). "+
			"Press CTRL+C again to force quit.", false)
	}

	// 暂停菜单期间再次按下 Ctrl+C 立即强制退出（与 dirsearch 一致）
	stopForceQuit := make(chan struct{})
	defer close(stopForceQuit)
	if c.sigChan != nil {
		go func() {
			select {
			case <-c.sigChan:
				viewForceQuit()
				os.Exit(1)
			case <-stopForceQuit:
			}
		}()
	}

	for {
		message := "[q]uit / [c]ontinue"
		c.stateMutex.Lock()
		dirs := len(c.directories)
		urls := len(c.Options.URLs)
		c.stateMutex.Unlock()

		if dirs > 1 {
			message += " / [n]ext"
		}
		if urls > 1 {
			message += " / [s]kip target"
		}

		c.UI.InLine(message + ": ")
		option := strings.ToLower(c.readLine())

		switch {
		case option == "q":
			c.UI.InLine("[s]ave / [q]uit without saving: ")
			choice := strings.ToLower(c.readLine())
			if choice == "s" {
				sessionFile := c.DefaultSessionPath()
				c.UI.InLine(fmt.Sprintf("Save to file [%s]: ", sessionFile))
				input := c.readLine()
				if input != "" {
					sessionFile = input
				}
				if err := c.Export(sessionFile); err != nil {
					c.UI.Error(fmt.Sprintf("Failed to save session: %s", err))
					return
				}
				c.stateMutex.Lock()
				c.currentInterrupt = &Interrupt{Type: InterruptQuit,
					Message: fmt.Sprintf("Session saved to: %s", sessionFile)}
				c.stateMutex.Unlock()
				return
			} else if choice == "q" {
				c.stateMutex.Lock()
				c.currentInterrupt = &Interrupt{Type: InterruptQuit, Message: "Canceled by the user"}
				c.stateMutex.Unlock()
				return
			}
		case option == "c":
			c.fuzzer.Play()
			return
		case option == "n" && dirs > 1:
			c.stateMutex.Lock()
			c.currentInterrupt = &Interrupt{Type: InterruptNextDir, Message: ""}
			c.stateMutex.Unlock()
			return
		case option == "s" && urls > 1:
			c.stateMutex.Lock()
			c.currentInterrupt = &Interrupt{Type: InterruptSkipTarget,
				Message: "Target skipped by the user"}
			c.stateMutex.Unlock()
			return
		}
	}
}

// readLine 读取一行用户输入。
// Windows 下 Ctrl+C 可能中止挂起的控制台读取使其立即报错返回，
// 因此失败时短暂等待后重试；连续失败（如 stdin 已关闭/EOF）
// 则强制退出，避免菜单循环空转刷屏。
func (c *Controller) readLine() string {
	if c.QuietInput != nil {
		return strings.TrimSpace(c.QuietInput())
	}
	for attempts := 0; ; attempts++ {
		line, err := c.stdin.ReadString('\n')
		if line != "" {
			return strings.TrimSpace(line)
		}
		if err == nil {
			continue
		}
		if err == io.EOF || attempts >= 2 {
			viewForceQuit()
			os.Exit(1)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// viewForceQuit 输出强制退出警告。
func viewForceQuit() {
	fmt.Fprint(os.Stderr, "\nForce quit!")
}

// DefaultSessionPath 返回默认会话保存路径。
func (c *Controller) DefaultSessionPath() string {
	sessionsDir := c.SessionsDir()
	startTime := c.StartTime()
	date := startTime[:10]
	datetime := utils.FormatDatetimeForPath(startTime)
	return filepath.Join(sessionsDir, date, "session_"+datetime+".json")
}

// SessionsDir 返回会话存储目录。
func (c *Controller) SessionsDir() string {
	if c.Options.SessionsDir != "" {
		return c.Options.SessionsDir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".hidir", "sessions")
	}
	return "sessions"
}

// Export 保存当前扫描状态到会话文件。
func (c *Controller) Export(sessionFile string) error {
	sessionFile = utils.FormatDatetimeForPath(sessionFile)
	if dir := filepath.Dir(sessionFile); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}

	c.stateMutex.Lock()
	payload := sessionPayload{
		Version:   1,
		Args:      os.Args[1:],
		StartTime: c.StartTime(),
		TargetURL: c.targetURL,
	}
	payload.Controller.JobsProcessed = c.jobsProcessed
	payload.Controller.Errors = c.errors
	payload.Controller.ConsecutiveErrors = c.consecutiveErrors
	payload.Controller.DirectoriesLeft = append([]string{}, c.directories...)
	payload.Controller.URLsLeft = append([]string{}, c.Options.URLs...)
	c.stateMutex.Unlock()

	if len(c.OutputHistory) > 0 {
		payload.OutputHistory = c.OutputHistory
	} else if c.UI != nil {
		payload.OutputHistory = []OutputEntry{{StartTime: c.StartTime(), Output: c.UI.Buffer()}}
	}

	data, err := json.MarshalIndent(payload, "", "    ")
	if err != nil {
		return err
	}
	c.SessionFile = sessionFile
	return os.WriteFile(sessionFile, data, 0o600)
}

// Import 从会话文件恢复扫描状态。
// 返回恢复的选项（重新解析会话中保存的命令行参数）。
func (c *Controller) Import(sessionFile string) error {
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return fmt.Errorf("%s is not a valid session file or it's in an old format", sessionFile)
	}
	var payload sessionPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("%s is not a valid session file or it's in an old format", sessionFile)
	}

	c.SessionFile = sessionFile
	c.OldSession = true

	// 重新解析保存的命令行参数（会话选项覆盖当前选项）
	fs := options.OSFilesystem()
	parsed, err := options.ParseOptions(payload.Args, fs, c.ResourceFS)
	if err == nil {
		parsed.SessionFile = sessionFile
		c.Options = parsed
	}

	// 恢复计数器与剩余队列
	c.jobsProcessed = payload.Controller.JobsProcessed
	c.errors = payload.Controller.Errors
	c.consecutiveErrors = payload.Controller.ConsecutiveErrors
	c.directories = payload.Controller.DirectoriesLeft
	c.Options.URLs = payload.Controller.URLsLeft
	c.OutputHistory = payload.OutputHistory

	// 询问覆盖方式
	c.UI.InLine(fmt.Sprintf("Resume session from %s. Overwrite on save? [o]verwrite/[n]ew: ", sessionFile))
	choice := strings.ToLower(c.readLine())
	if choice == "n" {
		c.SessionNew = true
	}
	return nil
}

// SessionInfo 描述一个可恢复的会话（用于列表展示）。
type SessionInfo struct {
	// Path 是会话文件路径。
	Path string
	// URL 是会话目标。
	URL string
	// TargetsLeft 是剩余目标数。
	TargetsLeft int
	// DirectoriesLeft 是剩余目录任务数。
	DirectoriesLeft int
	// JobsProcessed 是已完成任务数。
	JobsProcessed int
	// Errors 是错误计数。
	Errors int
	// Modified 是最后修改时间。
	Modified time.Time
}

// ListSessions 扫描会话目录并返回可恢复会话列表。
func ListSessions(baseDir string) ([]SessionInfo, error) {
	var sessions []SessionInfo
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过不可访问的目录
		}
		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		var payload sessionPayload
		if json.Unmarshal(data, &payload) != nil {
			return nil
		}
		statInfo, statErr := os.Stat(path)
		modified := time.Time{}
		if statErr == nil {
			modified = statInfo.ModTime()
		}
		sessions = append(sessions, SessionInfo{
			Path:            path,
			URL:             payload.TargetURL,
			TargetsLeft:     len(payload.Controller.URLsLeft),
			DirectoriesLeft: len(payload.Controller.DirectoriesLeft),
			JobsProcessed:   payload.Controller.JobsProcessed,
			Errors:          payload.Controller.Errors,
			Modified:        modified,
		})
		return nil
	})
	// 按修改时间排序
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.Before(sessions[j].Modified)
	})
	return sessions, err
}

// PrintSessionList 输出会话列表（格式与 dirsearch 一致）。
// 返回会话数量。
func PrintSessionList(baseDir string, print func(string)) int {
	sessions, err := ListSessions(baseDir)
	if err != nil || len(sessions) == 0 {
		print(fmt.Sprintf("No resumable sessions found in %s", baseDir))
		return 0
	}
	print(fmt.Sprintf("Resumable sessions in %s:", baseDir))
	for index, session := range sessions {
		modified := session.Modified.Format("2006-01-02 15:04:05")
		targetURL := session.URL
		if targetURL == "" {
			targetURL = "(unknown target)"
		}
		print(fmt.Sprintf("%d. %s | %s | targets left: %d | dirs left: %d | jobs done: %d | errors: %d | modified: %s",
			index+1, session.Path, targetURL, session.TargetsLeft,
			session.DirectoriesLeft, session.JobsProcessed, session.Errors, modified))
	}
	return len(sessions)
}

// ResolveSessionID 按编号解析会话文件路径（1 起始）。
func ResolveSessionID(baseDir, id string) (string, error) {
	index := 0
	for _, ch := range id {
		if ch < '0' || ch > '9' {
			return "", fmt.Errorf("Invalid session id: %s", id)
		}
		index = index*10 + int(ch-'0')
	}
	sessions, err := ListSessions(baseDir)
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "", fmt.Errorf("No resumable sessions found in %s", baseDir)
	}
	if index < 1 || index > len(sessions) {
		return "", fmt.Errorf("Session id out of range: %d (1-%d)", index, len(sessions))
	}
	return sessions[index-1].Path, nil
}
