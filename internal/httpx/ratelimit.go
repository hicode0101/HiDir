package httpx

import (
	"sync"
	"time"
)

// rateWindowSeconds 是限速统计窗口。
const rateWindowSeconds = time.Second

// RateLimiter 实现滑动窗口限速，与 dirsearch 的 RequestRateLimiter 一致。
type RateLimiter struct {
	mutex        sync.Mutex
	requestTimes []time.Time
}

// Wait 在达到 maxRate 限速时阻塞等待（maxRate<=0 表示不限速）。
func (r *RateLimiter) Wait(maxRate int) {
	for {
		delay := r.reserve(maxRate)
		if delay <= 0 {
			return
		}
		if delay > 100*time.Millisecond {
			delay = 100 * time.Millisecond
		}
		time.Sleep(delay)
	}
}

// reserve 尝试预约一次请求配额；需要等待时返回等待时长，否则返回 0。
func (r *RateLimiter) reserve(maxRate int) time.Duration {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	r.discardExpired(now)
	if maxRate <= 0 || len(r.requestTimes) < maxRate {
		r.requestTimes = append(r.requestTimes, now)
		return 0
	}
	// 计算最早一次请求移出窗口还需要多久
	wait := r.requestTimes[0].Add(rateWindowSeconds).Sub(now)
	if wait < 0 {
		wait = 0
	}
	return wait
}

// Rate 返回当前窗口内的请求数（当前速率）。
func (r *RateLimiter) Rate() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.discardExpired(time.Now())
	return len(r.requestTimes)
}

// discardExpired 移除窗口外的请求记录（调用方需持有锁）。
func (r *RateLimiter) discardExpired(now time.Time) {
	cutoff := now.Add(-rateWindowSeconds)
	idx := 0
	for idx < len(r.requestTimes) && !r.requestTimes[idx].After(cutoff) {
		idx++
	}
	if idx > 0 {
		r.requestTimes = append([]time.Time{}, r.requestTimes[idx:]...)
	}
}
