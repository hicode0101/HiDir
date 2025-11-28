package main

import (
	"fmt"
	"os"

	"github.com/yourusername/hidir/internal/core"
	"github.com/yourusername/hidir/internal/parse"
)

const VERSION = "1.0.0"

func main() {
	// 打印版本信息
	fmt.Printf("HiDir v%s\n\n", VERSION)

	// 解析命令行参数
	opts := parse.ParseArguments()

	// 初始化控制器
	controller := core.NewController(opts)

	// 设置控制器
	if err := controller.Setup(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	// 运行扫描
	controller.Run()
}
