package core

import (
	"sync"
	"time"

	"github.com/yourusername/hidir/internal/connection"
)

// MatchCallback 匹配回调函数类型
type MatchCallback func(*connection.Response)

// NotFoundCallback 未找到回调函数类型
type NotFoundCallback func(*connection.Response)

// ErrorCallback 错误回调函数类型
type ErrorCallback func(error)

// Fuzzer 模糊测试器
type Fuzzer struct {
	threads          []*thread
	requester        *connection.Requester
	dictionary       *Dictionary
	basePath         string
	isRunning        bool
	paused           bool
	mutex            sync.Mutex
	cond             *sync.Cond
	matchCallbacks   []MatchCallback
	notFoundCallbacks []NotFoundCallback
	errorCallbacks   []ErrorCallback
}

// thread 表示一个扫描线程
type thread struct {
	id     int
	fuzzer *Fuzzer
	wg     *sync.WaitGroup
}

// NewFuzzer 创建新的Fuzzer实例
func NewFuzzer(requester *connection.Requester, dictionary *Dictionary) *Fuzzer {
	fuzzer := &Fuzzer{
		requester:        requester,
		dictionary:       dictionary,
		isRunning:        false,
		paused:           false,
		matchCallbacks:   make([]MatchCallback, 0),
		notFoundCallbacks: make([]NotFoundCallback, 0),
		errorCallbacks:   make([]ErrorCallback, 0),
	}
	fuzzer.cond = sync.NewCond(&fuzzer.mutex)

	return fuzzer
}

// AddMatchCallback 添加匹配回调
func (f *Fuzzer) AddMatchCallback(callback MatchCallback) {
	f.matchCallbacks = append(f.matchCallbacks, callback)
}

// AddNotFoundCallback 添加未找到回调
func (f *Fuzzer) AddNotFoundCallback(callback NotFoundCallback) {
	f.notFoundCallbacks = append(f.notFoundCallbacks, callback)
}

// AddErrorCallback 添加错误回调
func (f *Fuzzer) AddErrorCallback(callback ErrorCallback) {
	f.errorCallbacks = append(f.errorCallbacks, callback)
}

// SetBasePath 设置基础路径
func (f *Fuzzer) SetBasePath(path string) {
	f.basePath = path
}

// Start 开始扫描
func (f *Fuzzer) Start(threadCount int) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.isRunning {
		return
	}

	f.isRunning = true
	f.paused = false

	// 创建线程
	var wg sync.WaitGroup
	f.threads = make([]*thread, threadCount)

	for i := 0; i < threadCount; i++ {
		t := &thread{
			id:     i,
			fuzzer: f,
			wg:     &wg,
		}
		f.threads[i] = t
		wg.Add(1)
		go t.run()
	}

	// 等待所有线程完成
	go func() {
		wg.Wait()
		f.mutex.Lock()
		f.isRunning = false
		f.mutex.Unlock()
	}()
}

// Stop 停止扫描
func (f *Fuzzer) Stop() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.isRunning = false
	f.paused = false
	f.cond.Broadcast()
}

// Pause 暂停扫描
func (f *Fuzzer) Pause() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.paused = true
}

// Resume 恢复扫描
func (f *Fuzzer) Resume() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.paused = false
	f.cond.Broadcast()
}

// IsRunning 检查是否正在运行
func (f *Fuzzer) IsRunning() bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.isRunning
}

// Wait 等待扫描完成
func (f *Fuzzer) Wait(timeout ...time.Duration) bool {
	if len(timeout) > 0 {
		timer := time.NewTimer(timeout[0])
		defer timer.Stop()

		for {
			if !f.IsRunning() {
				return true
			}

			select {
			case <-timer.C:
				return false
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	for f.IsRunning() {
		time.Sleep(100 * time.Millisecond)
	}

	return true
}

// run 线程运行函数
func (t *thread) run() {
	defer t.wg.Done()

	for {
		// 检查是否需要暂停
		t.fuzzer.mutex.Lock()
		for t.fuzzer.paused {
			t.fuzzer.cond.Wait()
		}

		// 检查是否需要停止
		if !t.fuzzer.isRunning {
			t.fuzzer.mutex.Unlock()
			return
		}
		t.fuzzer.mutex.Unlock()

		// 获取下一个单词
		word, ok := t.fuzzer.dictionary.Next()
		if !ok {
			break
		}

		// 构建完整路径
		path := t.fuzzer.basePath + word

		// 发送请求
		response, err := t.fuzzer.requester.Request(path)
		if err != nil {
			// 调用错误回调
			for _, callback := range t.fuzzer.errorCallbacks {
				callback(err)
			}
			continue
		}

		// 检查响应是否有效
		if t.fuzzer.isValidResponse(response) {
			// 调用匹配回调
			for _, callback := range t.fuzzer.matchCallbacks {
				callback(response)
			}
		} else {
			// 调用未找到回调
			for _, callback := range t.fuzzer.notFoundCallbacks {
				callback(response)
			}
		}

		// 延迟
		if delay, ok := Options["delay"].(float64); ok {
			time.Sleep(time.Duration(delay) * time.Second)
		}
	}
}

// isValidResponse 检查响应是否有效
func (f *Fuzzer) isValidResponse(response *connection.Response) bool {
	// 检查状态码
	if statusCodes, ok := Options["exclude_status_codes"].(map[int]bool); ok {
		if statusCodes[response.Status] {
			return false
		}
	}

	if statusCodes, ok := Options["include_status_codes"].(map[int]bool); ok {
		if !statusCodes[response.Status] {
			return false
		}
	}

	// 检查内容长度
	if minSize, ok := Options["minimum_response_size"].(int); ok {
		if response.Length < int64(minSize) {
			return false
		}
	}

	if maxSize, ok := Options["maximum_response_size"].(int); ok && maxSize > 0 {
		if response.Length > int64(maxSize) {
			return false
		}
	}

	// 检查内容
	if excludeTexts, ok := Options["exclude_texts"].([]string); ok {
		for _, text := range excludeTexts {
			if contains(response.Content, text) {
				return false
			}
		}
	}

	return true
}

// contains 检查字符串是否包含指定文本
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || s[0:len(s)] == substr || contains(s[1:], substr))
}
