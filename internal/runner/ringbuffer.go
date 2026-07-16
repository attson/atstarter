package runner

import "sync"

// ringBuffer 是固定容量的日志行环形缓冲,满时丢弃最旧行。并发安全。
type ringBuffer struct {
	mu    sync.Mutex
	buf   []string
	size  int
	start int
	count int
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{buf: make([]string, size), size: size}
}

func (r *ringBuffer) add(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	idx := (r.start + r.count) % r.size
	if r.count < r.size {
		r.buf[idx] = line
		r.count++
	} else {
		r.buf[r.start] = line
		r.start = (r.start + 1) % r.size
	}
}

// snapshot 返回当前缓冲内容(按时间顺序)的拷贝。
func (r *ringBuffer) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, r.count)
	for i := 0; i < r.count; i++ {
		out[i] = r.buf[(r.start+i)%r.size]
	}
	return out
}
