package utils

import (
	"bufio"
	"os"
	"strings"
)

// RedactedValue 是敏感参数的替换占位。
const RedactedValue = "<redacted>"

// sensitiveOptions 是值需要脱敏的选项集合。
var sensitiveOptions = map[string]bool{
	"-d": true, "--data": true,
	"-H": true, "--header": true,
	"--auth": true, "--cookie": true,
	"-p": true, "--proxy": true,
	"--proxy-auth": true, "--replay-proxy": true,
	"--mysql-url": true, "--postgres-url": true,
}

// targetOptions 是目标选项（值含 @ 时脱敏）。
var targetOptions = map[string]bool{"-u": true, "--url": true}

// shortFlagOptions 是无参数的短选项。
var shortFlagOptions = map[string]bool{
	"-a": true, "-C": true, "-f": true, "-F": true, "-h": true,
	"-L": true, "-q": true, "-r": true, "-U": true, "-v": true,
}

// RedactCommand 对命令行参数进行脱敏，返回可安全展示/持久化的字符串。
// 行为与 dirsearch 的 redact_command 一致。
func RedactCommand(arguments []string) string {
	var redacted []string
	index := 0

	for index < len(arguments) {
		argument := arguments[index]
		if argument == "--" {
			redacted = append(redacted, arguments[index:]...)
			break
		}

		// --option=value 形式
		if option, value, found := strings.Cut(argument, "="); found && strings.HasPrefix(option, "--") {
			redacted = append(redacted, option+"="+redactValue(option, value))
			index++
			continue
		}

		if sensitiveOptions[argument] || targetOptions[argument] {
			redacted = append(redacted, argument)
			if index+1 < len(arguments) {
				redacted = append(redacted, redactValue(argument, arguments[index+1]))
				index += 2
			} else {
				index++
			}
			continue
		}

		// 短选项与值连写（如 -pHost:8080、-uURL）
		if option, valueIdx, ok := findShortValueOption(argument); ok {
			attached := argument[valueIdx:]
			if attached != "" {
				redacted = append(redacted, argument[:valueIdx]+redactValue(option, attached))
			} else {
				redacted = append(redacted, argument)
				if index+1 < len(arguments) {
					redacted = append(redacted, redactValue(option, arguments[index+1]))
					index++
				}
			}
			index++
			continue
		}

		redacted = append(redacted, argument)
		index++
	}
	return strings.Join(redacted, " ")
}

// redactValue 依据选项类型决定是否脱敏取值。
func redactValue(option, value string) string {
	if sensitiveOptions[option] {
		return RedactedValue
	}
	if targetOptions[option] && strings.Contains(value, "@") {
		return RedactedValue
	}
	return value
}

// findShortValueOption 检查短选项串是否携带值（如 -uX 或 -p 后接值）。
func findShortValueOption(argument string) (string, int, bool) {
	if !strings.HasPrefix(argument, "-") || strings.HasPrefix(argument, "--") || len(argument) < 2 {
		return "", 0, false
	}
	for i := 1; i < len(argument); i++ {
		option := "-" + string(argument[i])
		if option == "-u" || option == "-d" || option == "-H" || option == "-p" {
			return option, i + 1, true
		}
		if !shortFlagOptions[option] {
			return "", 0, false
		}
	}
	return "", 0, false
}

// StdinReader 提供可测试的标准输入逐行读取。
type StdinReader struct {
	// scanner 是底层行扫描器（懒创建）。
	scanner *bufioScanner
}

// bufioScanner 是 bufio.Scanner 的别名（避免直接依赖扩散）。
type bufioScanner = bufio.Scanner

// NewStdinReader 创建标准输入读取器。
func NewStdinReader() *StdinReader {
	return &StdinReader{scanner: bufio.NewScanner(os.Stdin)}
}

// ReadLine 读取一行并去除行尾换行；输入结束时返回空串。
func (r *StdinReader) ReadLine() string {
	if r.scanner.Scan() {
		return strings.TrimSpace(r.scanner.Text())
	}
	return ""
}
