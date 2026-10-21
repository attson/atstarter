package main

import (
	"sync"
	"time"

	"atstarter/internal/runner"
)

// LogBatch 是发送给前端的有界日志批次。
type LogBatch struct {
	Lines []runner.LogLine `json:"lines"`
}

type logBatcher struct {
	mu     sync.Mutex
	closed bool
	input  chan runner.LogLine
	stop   chan struct{}
	done   chan struct{}
}

func newLogBatcher(interval time.Duration, maxBatchSize int, emit func(LogBatch)) *logBatcher {
	if maxBatchSize < 1 {
		maxBatchSize = 1
	}
	b := &logBatcher{
		input: make(chan runner.LogLine, maxBatchSize),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	go b.run(interval, maxBatchSize, emit)
	return b
}

func (b *logBatcher) Add(line runner.LogLine) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.input <- line
}

func (b *logBatcher) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		<-b.done
		return
	}
	b.closed = true
	close(b.stop)
	b.mu.Unlock()
	<-b.done
}

func (b *logBatcher) run(interval time.Duration, maxBatchSize int, emit func(LogBatch)) {
	defer close(b.done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	lines := make([]runner.LogLine, 0, maxBatchSize)
	flush := func() {
		if len(lines) == 0 {
			return
		}
		batch := LogBatch{Lines: append([]runner.LogLine(nil), lines...)}
		lines = lines[:0]
		emit(batch)
	}
	add := func(line runner.LogLine) {
		lines = append(lines, line)
		if len(lines) >= maxBatchSize {
			flush()
		}
	}

	for {
		select {
		case line := <-b.input:
			add(line)
		case <-ticker.C:
			flush()
		case <-b.stop:
			for {
				select {
				case line := <-b.input:
					add(line)
				default:
					flush()
					return
				}
			}
		}
	}
}
