package options

import (
	"fmt"
	"strings"
)

// Version 供 --version 输出使用。
const Version = "1.0.0"

// Usage 是帮助头部的用法说明。
const Usage = "Usage: hidir [-u|--url] URL [-e|--extensions] EXTENSIONS [options]"

// commonHelpOptions 是 -h/--help 显示的常用选项集合（与 dirsearch 一致）。
var commonHelpOptions = map[string]bool{
	"--version": true, "--help": true, "--help-all": true,
	"--url": true, "--urls-file": true, "--stdin": true, "--cidr": true,
	"--raw": true, "--session": true, "--config": true,
	"--wordlists": true, "--wordlist-categories": true,
	"--extensions": true, "--force-extensions": true, "--exclude-extensions": true,
	"--prefixes": true, "--suffixes": true,
	"--threads": true, "--async": true, "--no-async": true,
	"--recursive": true, "--max-recursion-depth": true, "--recursion-status": true,
	"--include-status": true, "--exclude-status": true, "--exclude-sizes": true,
	"--exclude-text": true, "--exclude-regex": true, "--exclude-redirect": true,
	"--max-time": true, "--target-max-time": true,
	"--http-method": true, "--data": true, "--header": true,
	"--follow-redirects": true, "--random-agent": true, "--user-agent": true,
	"--cookie": true, "--auth": true, "--auth-type": true,
	"--timeout": true, "--delay": true, "--proxy": true, "--tor": true,
	"--scheme": true, "--max-rate": true, "--retries": true,
	"--crawl": true, "--full-url": true, "--no-color": true, "--quiet-mode": true,
	"--verbose": true, "--output-formats": true, "--output-file": true,
	"--save-response": true, "--save-response-jsonl": true, "--log": true,
}

// optionStrings 生成选项在帮助中的展示文本（如 "-u URL, --url=URL"）。
func optionStrings(spec *optSpec) string {
	var parts []string
	if spec.Short != "" {
		parts = append(parts, spec.Short)
	}
	for _, long := range strings.Split(spec.Long, ", ") {
		if spec.HasArg {
			metavar := spec.Metavar
			if metavar == "" {
				metavar = strings.ToUpper(strings.TrimPrefix(spec.Dest, "_"))
			}
			parts = append(parts, long+"="+metavar)
		} else {
			parts = append(parts, long)
		}
	}
	return strings.Join(parts, ", ")
}

// isCommonHelp 判断选项是否属于常用帮助集合。
func isCommonHelp(spec *optSpec) bool {
	for _, long := range strings.Split(spec.Long, ", ") {
		if commonHelpOptions[long] {
			return true
		}
	}
	return false
}

// wrapHelpText 按 optparse 风格折行帮助文本。
// firstPrefix 是首行前缀（通常为空或 24 空格缩进），
// 后续行固定缩进 24 列，整体宽度 78。
func wrapHelpText(text string, width, indent int, firstPrefix string) []string {
	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	indentStr := strings.Repeat(" ", indent)
	line := firstPrefix
	for _, word := range words {
		if line == "" || (line == firstPrefix && firstPrefix != "") {
			line += word
			continue
		}
		if len(line)+1+len(word) > width {
			lines = append(lines, line)
			line = indentStr + word
			continue
		}
		line += " " + word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// renderHelp 渲染帮助文本。onlyCommon 为 true 时仅输出常用选项。
func renderHelp(onlyCommon bool, extraHeader string) string {
	var b strings.Builder
	b.WriteString(extraHeader)
	b.WriteString(Usage + "\n\n")
	b.WriteString("Options:\n")

	// 顶层通用选项
	b.WriteString("  --version             show program's version number and exit\n")
	b.WriteString("  -h, --help            Show common options and exit\n")
	b.WriteString("  --help-all            Show all options and exit (short form: -hh)\n")

	for _, group := range groups {
		var specs []*optSpec
		for i := range optionSpecs {
			spec := &optionSpecs[i]
			if spec.Group != group {
				continue
			}
			if onlyCommon && !isCommonHelp(spec) {
				continue
			}
			specs = append(specs, spec)
		}
		if len(specs) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("\n  %s:\n", group))
		for _, spec := range specs {
			left := "    " + optionStrings(spec)
			help := spec.Help
			if help == "" {
				b.WriteString(left + "\n")
				continue
			}
			const helpCol = 24
			if len(left) < helpCol {
				// 选项列较短：帮助文本与选项同行，续行缩进 24
				b.WriteString(left + strings.Repeat(" ", helpCol-len(left)))
				lines := wrapHelpText(help, 78, helpCol, "")
				b.WriteString(lines[0] + "\n")
				for _, l := range lines[1:] {
					b.WriteString(l + "\n")
				}
			} else {
				// 选项列过长：帮助文本另起一行并整体缩进 24
				b.WriteString(left + "\n")
				lines := wrapHelpText(help, 78, helpCol, strings.Repeat(" ", helpCol))
				for _, l := range lines {
					b.WriteString(l + "\n")
				}
			}
		}
	}
	return b.String()
}

// HelpCommon 是 -h/--help 输出的常用帮助文本。
var HelpCommon = renderHelp(true, "")

// HelpAll 是 --help-all/-hh 输出的完整帮助文本。
var HelpAll = renderHelp(false, "")

// authTypes 是支持的认证类型列表。
var authTypes = []string{"basic", "digest", "bearer", "ntlm", "jwt"}

// outputFormatsAll 是全部可用的文件型输出格式。
var outputFormatsAll = []string{"simple", "plain", "json", "xml", "md", "csv", "html", "sqlite"}

// commonExtensions 是 -e "*" 对应的扩展名集合。
var commonExtensions = []string{"php", "jsp", "asp", "aspx", "do", "action", "cgi", "html", "htm", "js", "tar.gz"}
