// Package view 实现终端输出（状态报告、进度条、错误提示与颜色），
// 对应 dirsearch 的 lib/view 目录。
package view

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode"

	"hidir/internal/httpx"
)

// ANSI 颜色代码。
const (
	ansiReset   = "\x1b[0m"
	ansiBright  = "\x1b[1m"
	ansiDim     = "\x1b[2m"
	ansiRed     = "\x1b[31m"
	ansiGreen   = "\x1b[32m"
	ansiYellow  = "\x1b[33m"
	ansiBlue    = "\x1b[34m"
	ansiMagenta = "\x1b[35m"
	ansiCyan    = "\x1b[36m"
	ansiWhite   = "\x1b[37m"

	ansiBackRed = "\x1b[41m"
)

// colorEnabled 控制是否输出 ANSI 颜色。
var colorEnabled = true

// DisableColor 关闭颜色输出（--no-color）。
func DisableColor() { colorEnabled = false }

// SetColor 为文本添加颜色（fore/back/style 与 dirsearch 的 set_color 一致）。
func SetColor(msg, fore, back, style string) string {
	if !colorEnabled {
		return msg
	}
	var b strings.Builder
	switch style {
	case "bright":
		b.WriteString(ansiBright)
	case "dim":
		b.WriteString(ansiDim)
	}
	switch fore {
	case "red":
		b.WriteString(ansiRed)
	case "green":
		b.WriteString(ansiGreen)
	case "yellow":
		b.WriteString(ansiYellow)
	case "blue":
		b.WriteString(ansiBlue)
	case "magenta":
		b.WriteString(ansiMagenta)
	case "cyan":
		b.WriteString(ansiCyan)
	case "white":
		b.WriteString(ansiWhite)
	}
	switch back {
	case "red":
		b.WriteString(ansiBackRed)
	}
	b.WriteString(msg)
	b.WriteString(ansiReset)
	return b.String()
}

// CleanColor 去除文本中的 ANSI 转义序列。
func CleanColor(msg string) string {
	var b strings.Builder
	for i := 0; i < len(msg); i++ {
		if msg[i] == 0x1b && i+1 < len(msg) && msg[i+1] == '[' {
			// 跳过到字母结束
			j := i + 2
			for j < len(msg) && !isANSITerminator(msg[j]) {
				j++
			}
			if j < len(msg) {
				i = j
				continue
			}
		}
		b.WriteByte(msg[i])
	}
	return b.String()
}

// isANSITerminator 判断 ANSI 序列结束字节。
func isANSITerminator(c byte) bool {
	return (c >= '@' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// maxDisplayTextLength 是状态行 URL 的最大显示长度。
const maxDisplayTextLength = 240

// SafeDisplayText 移除控制字符并截断过长文本。
func SafeDisplayText(value string) string {
	var b strings.Builder
	for _, c := range value {
		if !unicode.IsControl(c) {
			b.WriteRune(c)
		}
	}
	text := b.String()
	if len(text) > maxDisplayTextLength {
		return text[:maxDisplayTextLength-3] + "..."
	}
	return text
}

// CLI 是终端输出接口，包含普通/安静/静默三种实现。
type CLI struct {
	mutex        sync.Mutex
	lastInLine   bool
	lastLineLen  int // 上一条 InLine 内容的可显示宽度（用于回退擦除）
	outputBuffer []string
	disableAll   bool
	quiet        bool
}

// NewCLI 按选项创建对应级别的终端输出。
func NewCLI(color, quiet, disableCLI bool) *CLI {
	if !color {
		DisableColor()
	}
	return &CLI{disableAll: disableCLI, quiet: quiet}
}

// Buffer 返回已保存的输出（用于会话保存）。
func (c *CLI) Buffer() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return strings.Join(c.outputBuffer, "\n")
}

// erase 清除当前输出行，使进度条可以原地刷新（与 dirsearch 的 erase 一致）。
func (c *CLI) erase() {
	if ansiEnabled {
		// 清除整行并将光标移到行首
		fmt.Print("\x1b[1K\x1b[0G")
		return
	}
	// 回退方案：回车后用空格覆盖上一条内容再回到行首
	if c.lastLineLen > 0 {
		fmt.Print("\r" + strings.Repeat(" ", c.lastLineLen) + "\r")
	} else {
		fmt.Print("\r")
	}
	c.lastLineLen = 0
}

// NewLine 输出一行并按需保存。
func (c *CLI) NewLine(message string) {
	c.NewLineSave(message, true)
}

// NewLineSave 输出一行，doSave 控制是否保存到输出缓冲。
func (c *CLI) NewLineSave(message string, doSave bool) {
	if c.disableAll {
		return
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// 若上一条输出是不换行的进度条/提示，先将其擦除
	if c.lastInLine {
		c.erase()
		c.lastInLine = false
	}
	fmt.Println(message)
	if doSave {
		c.outputBuffer = append(c.outputBuffer, message)
	}
}

// InLine 原地刷新一行内容：先擦除上一条内容再输出（用于进度条与交互菜单）。
func (c *CLI) InLine(message string) {
	if c.disableAll {
		return
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.erase()
	fmt.Print(message)
	c.lastLineLen = len(CleanColor(message))
	c.lastInLine = true
}

// Error 输出错误信息。
func (c *CLI) Error(reason string) {
	if c.disableAll {
		return
	}
	c.NewLine("\n" + SetColor(reason, "white", "red", "bright"))
}

// Warning 输出警告信息。
func (c *CLI) Warning(message string, doSave bool) {
	if c.disableAll || c.quiet {
		return
	}
	c.NewLineSave(SetColor(message, "yellow", "", "bright"), doSave)
}

// Header 输出横幅。
func (c *CLI) Header(message string) {
	if c.disableAll || c.quiet {
		return
	}
	c.NewLine(SetColor(message, "magenta", "", "bright"))
}

// PrintHeader 输出键值配置信息。
func (c *CLI) PrintHeader(headers map[string]string) {
	if c.disableAll || c.quiet {
		return
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sortStrings(keys)
	var lines []string
	for _, key := range keys {
		pair := SetColor(key+": ", "yellow", "", "bright") + SetColor(headers[key], "cyan", "", "bright")
		if len(lines) == 0 {
			lines = append(lines, pair)
		} else {
			lines[len(lines)-1] += SetColor(" | ", "magenta", "", "bright") + pair
		}
	}
	c.NewLine(strings.Join(lines, "\n"))
}

// Target 输出目标信息。
func (c *CLI) Target(target string) {
	if c.disableAll || c.quiet {
		return
	}
	c.NewLine("")
	c.PrintHeader(map[string]string{"Target": target})
}

// Config 输出扫描配置概览。
func (c *CLI) Config(extensions, prefixes, suffixes []string, method string, threads int, wordlistSize int) {
	if c.disableAll || c.quiet {
		return
	}
	config := map[string]string{
		"Extensions":    strings.Join(extensions, ", "),
		"HTTP method":   method,
		"Threads":       fmt.Sprintf("%d", threads),
		"Wordlist size": fmt.Sprintf("%d", wordlistSize),
	}
	if len(prefixes) > 0 {
		config["Prefixes"] = strings.Join(prefixes, ", ")
	}
	if len(suffixes) > 0 {
		config["Suffixes"] = strings.Join(suffixes, ", ")
	}
	c.PrintHeader(config)
}

// LogFile 输出日志文件路径。
func (c *CLI) LogFile(file string) {
	if c.disableAll || c.quiet {
		return
	}
	c.NewLine(fmt.Sprintf("\nLog File: %s", file))
}

// StatusReport 输出一条命中结果。
func (c *CLI) StatusReport(response *httpx.Response, fullURL, verbose bool) {
	if c.disableAll {
		return
	}
	target := "/" + response.FullPath
	if fullURL {
		target = response.URL
	}
	target = SafeDisplayText(target)
	timePart := response.Datetime
	if idx := strings.Index(response.Datetime, " "); idx >= 0 {
		timePart = response.Datetime[idx+1:]
	}
	message := fmt.Sprintf("[%s] %d - %6s - %s", timePart, response.Status, response.Size(), target)

	if verbose {
		elapsedMS := 0
		if response.Elapsed > 0 {
			elapsedMS = int(response.Elapsed * 1000)
		}
		message += fmt.Sprintf("  (%dms, %s)", elapsedMS, response.Type())
	}

	switch {
	case response.Status == 200 || response.Status == 201 || response.Status == 204:
		message = SetColor(message, "green", "", "")
	case response.Status == 401:
		message = SetColor(message, "yellow", "", "")
	case response.Status == 403:
		message = SetColor(message, "blue", "", "")
	case response.Status >= 500 && response.Status < 600:
		message = SetColor(message, "red", "", "")
	case response.Status >= 300 && response.Status < 400:
		message = SetColor(message, "cyan", "", "")
	default:
		message = SetColor(message, "magenta", "", "")
	}

	if response.Redirect != "" {
		message += "  ->  " + SafeDisplayText(response.Redirect)
	}
	for _, redirect := range response.History {
		message += fmt.Sprintf("\n-->  %s", SafeDisplayText(redirect))
	}
	c.NewLine(message)
}

// LastPath 输出进度条。
func (c *CLI) LastPath(index, length, currentJob, allJobs int, rate, errors int) {
	if c.disableAll || c.quiet || length == 0 {
		return
	}
	percentage := index * 100 / length
	bars := percentage / 5
	task := strings.Repeat(SetColor("#", "cyan", "", "bright"), bars)
	task += strings.Repeat(" ", 20-bars)
	progress := fmt.Sprintf("%d/%d", index, length)

	jobs := fmt.Sprintf("%s:%d/%d", SetColor("job", "green", "", "bright"), currentJob, allJobs)
	errorText := fmt.Sprintf("%s:%d", SetColor("errors", "red", "", "bright"), errors)

	progressBar := fmt.Sprintf("[%s] %2d%% ", task, percentage)
	progressBar += fmt.Sprintf("%12s ", progress)
	progressBar += fmt.Sprintf("%9s/s       ", fmt.Sprintf("%d", rate))
	progressBar += fmt.Sprintf("%-21s %s", jobs, errorText)

	if len(CleanColor(progressBar)) >= terminalWidth() {
		return
	}
	c.InLine(progressBar)
}

// NewDirectories 输出递归新增目录。
func (c *CLI) NewDirectories(directories []string) {
	if c.disableAll || c.quiet {
		return
	}
	message := SetColor(
		fmt.Sprintf("Added to the queue: %s", SafeDisplayText(strings.Join(directories, ", "))),
		"yellow", "", "dim")
	c.NewLine(message)
}

// terminalWidth 返回终端宽度：优先 COLUMNS 环境变量，
// 其次查询真实终端，均不可用时回退 80（与 dirsearch 的 get_terminal_size 一致）。
func terminalWidth() int {
	if columns := os.Getenv("COLUMNS"); columns != "" {
		if n := parseInt(columns); n > 0 {
			return n
		}
	}
	if w, ok := queryTerminalWidth(); ok && w > 0 {
		return w
	}
	return 80
}

// parseInt 解析非负整数，失败返回 0。
func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// sortStrings 简单排序。
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// ForceQuitWarning 输出强制退出警告。
func ForceQuitWarning() {
	fmt.Fprint(os.Stderr, "\nForce quit!")
}
