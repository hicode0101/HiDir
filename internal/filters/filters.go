// Package filters 实现响应过滤值的解析与匹配，
// 对应 dirsearch 的 lib/core/filters.py。
package filters

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// NumericRange 表示闭区间数值范围 [Min, Max]。
type NumericRange struct {
	Min int
	Max int
}

// TimeFilter 表示时间比较器（>、<、=）与毫秒阈值。
type TimeFilter struct {
	Op    string
	Value float64
}

// SizeUnit 表示大小单位与换算系数。
var SizeUnits = map[string]int64{
	"B":  1,
	"KB": 1024,
	"MB": 1024 * 1024,
	"GB": 1024 * 1024 * 1024,
	"TB": 1024 * 1024 * 1024 * 1024,
	"PB": 1024 * 1024 * 1024 * 1024 * 1024,
	"EB": 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
	// ZB/YB 超出 int64 表示范围，饱和为最大值
	"ZB": math.MaxInt64,
	"YB": math.MaxInt64,
}

// sizeRegex 匹配 "数字+可选单位" 形式的大小值。
var sizeRegex = regexp.MustCompile(`(?i)^(\d+)\s*([KMGTPEZY]?B)?$`)

// ParseNumericRanges 解析逗号分隔的数值/范围列表（如 "200,300-399"）。
func ParseNumericRanges(value string) ([]NumericRange, error) {
	if value == "" {
		return nil, nil
	}
	var ranges []NumericRange
	for _, token := range strings.Split(value, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if strings.Contains(token, "-") {
			parts := strings.SplitN(token, "-", 2)
			minimum, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			maximum, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("invalid numeric range: %s", token)
			}
			if minimum > maximum {
				return nil, fmt.Errorf("invalid numeric range: %s", token)
			}
			ranges = append(ranges, NumericRange{Min: minimum, Max: maximum})
			continue
		}
		number, err := strconv.Atoi(token)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric value: %s", token)
		}
		ranges = append(ranges, NumericRange{Min: number, Max: number})
	}
	return ranges, nil
}

// ParseTimeFilters 解析时间过滤表达式（如 ">100,<200"）。
func ParseTimeFilters(value string) ([]TimeFilter, error) {
	if value == "" {
		return nil, nil
	}
	var out []TimeFilter
	for _, token := range strings.Split(value, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		op := "="
		if token[0] == '>' || token[0] == '<' {
			op = string(token[0])
			token = token[1:]
		}
		num, err := strconv.ParseFloat(strings.TrimSpace(token), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid time filter: %s%s", op, token)
		}
		out = append(out, TimeFilter{Op: op, Value: num})
	}
	return out, nil
}

// ParseSize 解析大小值（如 "1024"、"1KB"、"4kb"），空值返回 0。
func ParseSize(value string) (int64, error) {
	token := strings.TrimSpace(value)
	if token == "" {
		return 0, nil
	}
	m := sizeRegex.FindStringSubmatch(token)
	if m == nil {
		return 0, fmt.Errorf("invalid response size: %s", value)
	}
	number, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid response size: %s", value)
	}
	unit := strings.ToUpper(m[2])
	if unit == "" {
		unit = "B"
	}
	multiplier, ok := SizeUnits[unit]
	if !ok {
		return 0, fmt.Errorf("invalid response size: %s", value)
	}
	return number * multiplier, nil
}

// ParseSizeList 解析逗号分隔的大小列表为集合（如 "0,4KB"）。
func ParseSizeList(value string) (map[int64]struct{}, error) {
	if value == "" {
		return nil, nil
	}
	out := make(map[int64]struct{})
	for _, token := range strings.Split(value, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		size, err := ParseSize(token)
		if err != nil {
			return nil, err
		}
		out[size] = struct{}{}
	}
	return out, nil
}

// ValidateRegex 校验正则表达式可编译，否则返回带选项名的错误。
func ValidateRegex(pattern, label string) error {
	if pattern == "" {
		return nil
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("invalid %s regular expression: %s", label, err)
	}
	return nil
}

// MatchesNumericRanges 判断 value 是否落在任一范围内。
func MatchesNumericRanges(value int64, ranges []NumericRange) bool {
	for _, r := range ranges {
		if value >= int64(r.Min) && value <= int64(r.Max) {
			return true
		}
	}
	return false
}

// MatchesTimeFilters 判断耗时（毫秒）是否满足任一时间过滤条件。
func MatchesTimeFilters(elapsedSeconds float64, filters []TimeFilter) bool {
	milliseconds := elapsedSeconds * 1000
	for _, f := range filters {
		switch f.Op {
		case ">":
			if milliseconds > f.Value {
				return true
			}
		case "<":
			if milliseconds < f.Value {
				return true
			}
		case "=":
			if milliseconds == f.Value {
				return true
			}
		}
	}
	return false
}
