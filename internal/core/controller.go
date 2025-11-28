package core

import (
	"fmt"
	"os"
	"strings"
	"time"

	"HiDir/internal/common"
	"HiDir/internal/connection"
	"HiDir/internal/parse"
	"HiDir/internal/utils"
)

// Controller 控制器
type Controller struct {
	requester         *connection.Requester
	dictionary        *Dictionary
	fuzzer            *Fuzzer
	opts              *parse.Options
	results           []*connection.Response
	targets           []string
	startTime         time.Time
	directories       []string
	passedURLs        map[string]bool
	errors            int
	consecutiveErrors int
	dictFiles         []string // 保存使用的字典文件
}

// NewController 创建新的Controller实例
func NewController(opts *parse.Options) *Controller {
	return &Controller{
		opts:              opts,
		results:           make([]*connection.Response, 0),
		targets:           make([]string, 0),
		dictFiles:         make([]string, 0),
		passedURLs:        make(map[string]bool),
		errors:            0,
		consecutiveErrors: 0,
	}
}

// Setup 初始化控制器
func (c *Controller) Setup() error {
	// 初始化黑名单
	Blacklists = GetBlacklists()

	// 初始化请求器
	c.requester = connection.NewRequester()

	// 初始化字典
	var dictFiles []string
	if c.opts.Wordlists != "" {
		dictFiles = strings.Split(c.opts.Wordlists, ",")
	} else {
		// 默认使用dict目录下的所有txt文件
		dictFiles = utils.GetFilesByExtension("./dict", "txt")
		if len(dictFiles) == 0 {
			return fmt.Errorf("no dictionary files found in dict directory")
		}
	}

	// 保存字典文件路径
	c.dictFiles = dictFiles

	c.dictionary = NewDictionary(dictFiles...)
	if err := c.dictionary.Load(); err != nil {
		return err
	}

	// 初始化fuzzer
	c.fuzzer = NewFuzzer(c.requester, c.dictionary)
	c.fuzzer.SetOptions(c.opts) // 设置选项

	// 设置回调
	c.setupCallbacks()

	// 处理URLs
	c.processURLs()

	// 设置请求头
	c.setupHeaders()

	// 设置认证
	if c.opts.Auth != "" {
		c.requester.SetAuth(c.opts.AuthType, c.opts.Auth)
	}

	// 设置代理认证
	if c.opts.ProxyAuth != "" {
		c.requester.SetProxyAuth(c.opts.ProxyAuth)
	}

	return nil
}

// setupCallbacks 设置回调函数
func (c *Controller) setupCallbacks() {
	// 匹配回调
	c.fuzzer.AddMatchCallback(func(response *connection.Response) {
		c.matchCallback(response)
	})

	// 未找到回调
	c.fuzzer.AddNotFoundCallback(func(response *connection.Response) {
		c.notFoundCallback(response)
	})

	// 错误回调
	c.fuzzer.AddErrorCallback(func(err error) {
		c.errorCallback(err)
	})
}

// processURLs 处理URLs
func (c *Controller) processURLs() {
	if c.opts.URLFile != "" {
		f := utils.NewFile(c.opts.URLFile)
		c.targets = f.GetLines()
	} else if c.opts.CIDR != "" {
		c.targets = utils.IPRange(c.opts.CIDR)
	} else if c.opts.StdinURLs {
		// 从标准输入读取
		bytes, _ := os.ReadFile("/dev/stdin")
		c.targets = strings.Split(string(bytes), "\n")
	} else if c.opts.RawFile != "" {
		// 处理原始请求文件
		// 这里简化实现
	} else {
		c.targets = c.opts.URLs
	}

	// 去重
	c.targets = utils.Uniq(c.targets)
}

// setupHeaders 设置请求头
func (c *Controller) setupHeaders() {
	// 合并默认头和自定义头
	headers := make(map[string]string)

	// 添加默认头
	for k, v := range common.DEFAULT_HEADERS {
		headers[k] = v
	}

	// 添加自定义头
	if c.opts.HeaderFile != "" {
		f := utils.NewFile(c.opts.HeaderFile)
		headerContent := f.Read()
		customHeaders := parse.ParseHeaders(headerContent)
		for k, v := range customHeaders {
			headers[k] = v
		}
	}

	// 添加命令行指定的头
	for _, header := range c.opts.Headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers[key] = value
		}
	}

	// 设置User-Agent
	if c.opts.UserAgent != "" {
		headers["User-Agent"] = c.opts.UserAgent
	}

	// 设置Cookie
	if c.opts.Cookie != "" {
		headers["Cookie"] = c.opts.Cookie
	}

	c.requester.SetHeaders(headers)
}

// Run 运行扫描
func (c *Controller) Run() {
	// 输出主要参数信息
	fmt.Println("\n=== Scan Configuration ===")
	fmt.Println("Target URLs:")
	for _, target := range c.targets {
		if target != "" {
			fmt.Printf("  - %s\n", target)
		}
	}
	
	// 输出HTTP请求方法
	httpMethod := "GET" // 默认值
	if c.opts.HTTPMethod != "" {
		httpMethod = c.opts.HTTPMethod
	}
	fmt.Printf("HTTP Method: %s\n", httpMethod)
	
	// 输出使用的字典文件
	fmt.Println("Dictionary Files:")
	for _, dictFile := range c.dictFiles {
		fmt.Printf("  - %s\n", dictFile)
	}
	fmt.Println("========================")

	c.startTime = time.Now()

	for _, target := range c.targets {
		c.scanTarget(target)
	}

	fmt.Println("\nTask Completed")
}

// scanTarget 扫描单个目标
func (c *Controller) scanTarget(target string) {
	// 设置目标URL
	c.requester.SetURL(target)

	// 初始化目录
	c.directories = make([]string, 0)

	// 添加子目录
	if c.opts.Subdirs != "" {
		subdirs := strings.Split(c.opts.Subdirs, ",")
		for _, subdir := range subdirs {
			c.addDirectory(subdir)
		}
	} else {
		// 默认添加根目录
		c.addDirectory("")
	}

	// 开始扫描
	for _, dir := range c.directories {
		c.fuzzer.SetBasePath(dir)
		c.fuzzer.Start(c.opts.ThreadCount)
		c.fuzzer.Wait()
		c.dictionary.Reset()
	}
}

// addDirectory 添加目录到扫描队列
func (c *Controller) addDirectory(path string) {
	// 检查是否在排除列表中
	if c.opts.ExcludeSubdirs != "" {
		excludeSubdirs := strings.Split(c.opts.ExcludeSubdirs, ",")
		for _, exclude := range excludeSubdirs {
			if strings.Contains(path, exclude) {
				return
			}
		}
	}

	// 检查递归深度
	if c.opts.RecursionDepth > 0 {
		// 简化实现，实际应该更复杂
	}

	// 检查是否已处理
	if _, ok := c.passedURLs[path]; ok {
		return
	}

	c.directories = append(c.directories, path)
	c.passedURLs[path] = true
}

// matchCallback 匹配回调
func (c *Controller) matchCallback(response *connection.Response) {
	// 输出结果
	fmt.Printf("[%d] %s\n", response.Status, response.FullPath)

	// 添加到结果
	c.results = append(c.results, response)

	// 处理递归
	if c.opts.Recursive || c.opts.DeepRecursive || c.opts.ForceRecursive {
		// 简化实现，实际应该更复杂
		c.addDirectory(response.Path)
	}

	// 重置连续错误计数
	c.consecutiveErrors = 0
}

// notFoundCallback 未找到回调
func (c *Controller) notFoundCallback(response *connection.Response) {
	// 简化实现，实际应该更新进度条
	c.consecutiveErrors = 0
}

// errorCallback 错误回调
func (c *Controller) errorCallback(err error) {
	c.errors++
	c.consecutiveErrors++

	// 检查连续错误数
	if c.consecutiveErrors > common.MAX_CONSECUTIVE_REQUEST_ERRORS {
		fmt.Println("Too many consecutive errors, skipping target")
		c.fuzzer.Stop()
	}
}
