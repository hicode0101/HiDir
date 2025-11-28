package core

import (
	"strings"

	"HiDir/internal/utils"
)

// Dictionary 管理字典
type Dictionary struct {
	files     []string
	words     []string
	index     int
	processed map[string]bool
}

// NewDictionary 创建新的Dictionary实例
func NewDictionary(files ...string) *Dictionary {
	return &Dictionary{
		files:     files,
		words:     make([]string, 0),
		index:     0,
		processed: make(map[string]bool),
	}
}

// Load 加载字典文件
func (d *Dictionary) Load() error {
	for _, file := range d.files {
		f := utils.NewFile(file)
		lines := f.GetLines()
		d.words = append(d.words, lines...)
	}

	// 去重
	d.words = utils.Uniq(d.words)

	return nil
}

// Next 获取下一个单词
func (d *Dictionary) Next() (string, bool) {
	if d.index >= len(d.words) {
		return "", false
	}

	word := d.words[d.index]
	d.index++

	return word, true
}

// Reset 重置字典索引
func (d *Dictionary) Reset() {
	d.index = 0
}

// Len 获取字典长度
func (d *Dictionary) Len() int {
	return len(d.words)
}

// IsValid 检查路径是否有效
func (d *Dictionary) IsValid(path string) bool {
	// 简单实现，实际应该更复杂
	return path != "" && !strings.HasPrefix(path, "#")
}

// GetBlacklists 获取黑名单
func GetBlacklists() map[int][]string {
	// 从文件加载黑名单
	// 这里简化实现
	return map[int][]string{
		400: {"/400.html"},
		403: {"/403.html"},
		500: {"/500.html"},
	}
}
