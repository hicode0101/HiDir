package core

import (
	"HiDir/internal/connection"
)

// Scanner 检测响应是否为有效路径
type Scanner struct {
	requester *connection.Requester
	path      string
	context   string
	tested    map[string]map[string]*Scanner
}

// NewScanner 创建新的Scanner实例
func NewScanner(requester *connection.Requester, path string, tested map[string]map[string]*Scanner, context string) *Scanner {
	return &Scanner{
		requester: requester,
		path:      path,
		context:   context,
		tested:    tested,
	}
}

// Check 检查响应是否有效
func (s *Scanner) Check(path string, response *connection.Response) bool {
	// 简单实现，实际应该更复杂
	// 检查状态码、内容长度等
	return true
}
