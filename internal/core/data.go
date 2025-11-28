package core

// 全局数据存储
var (
	// 黑名单
	Blacklists = make(map[int][]string)
	// 选项
	Options = make(map[string]interface{})
)
