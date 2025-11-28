package parse

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfigParser 解析配置文件
type ConfigParser struct {
	config map[string]map[string]string
}

// NewConfigParser 创建新的ConfigParser实例
func NewConfigParser() *ConfigParser {
	return &ConfigParser{
		config: make(map[string]map[string]string),
	}
}

// Read 读取配置文件
func (cp *ConfigParser) Read(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var currentSection string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// 处理section
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimPrefix(strings.TrimSuffix(line, "]"), "[")
			cp.config[currentSection] = make(map[string]string)
			continue
		}

		// 处理键值对
		if currentSection != "" {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				cp.config[currentSection][key] = value
			}
		}
	}

	return scanner.Err()
}

// SafeGetString 安全获取字符串值
func (cp *ConfigParser) SafeGetString(section, key, defaultValue string) string {
	if sectionMap, ok := cp.config[section]; ok {
		if value, ok := sectionMap[key]; ok {
			return value
		}
	}
	return defaultValue
}

// SafeGetInt 安全获取整数值
func (cp *ConfigParser) SafeGetInt(section, key string, defaultValue int) int {
	valueStr := cp.SafeGetString(section, key, "")
	if valueStr == "" {
		return defaultValue
	}

	value := defaultValue
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return defaultValue
	}

	return value
}

// SafeGetFloat 安全获取浮点数值
func (cp *ConfigParser) SafeGetFloat(section, key string, defaultValue float64) float64 {
	valueStr := cp.SafeGetString(section, key, "")
	if valueStr == "" {
		return defaultValue
	}

	value := defaultValue
	_, err := fmt.Sscanf(valueStr, "%f", &value)
	if err != nil {
		return defaultValue
	}

	return value
}

// SafeGetBool 安全获取布尔值
func (cp *ConfigParser) SafeGetBool(section, key string, defaultValue bool) bool {
	valueStr := cp.SafeGetString(section, key, "")
	if valueStr == "" {
		return defaultValue
	}

	valueStr = strings.ToLower(valueStr)
	switch valueStr {
	case "true", "yes", "1":
		return true
	case "false", "no", "0":
		return false
	default:
		return defaultValue
	}
}

// SafeGetStringSlice 安全获取字符串切片
func (cp *ConfigParser) SafeGetStringSlice(section, key string, defaultValue []string) []string {
	valueStr := cp.SafeGetString(section, key, "")
	if valueStr == "" {
		return defaultValue
	}

	parts := strings.Split(valueStr, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	return parts
}
