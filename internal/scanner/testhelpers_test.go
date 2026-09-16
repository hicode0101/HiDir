package scanner

import (
	"hidir/internal/options"
)

// testOptions 构造一组带默认值的测试选项。
func testOptions() *options.Options {
	opts := options.NewDefaults()
	opts.ThreadCount = 2
	timeout := 5.0
	delay := 0.0
	retries := 0
	rate := 0
	depth := 0
	threshold := 0
	async := true
	opts.Timeout = &timeout
	opts.Delay = &delay
	opts.MaxRetries = &retries
	opts.MaxRate = &rate
	opts.RecursionDepth = &depth
	opts.FilterThreshold = &threshold
	opts.AsyncMode = &async
	return opts
}
