package main

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"atstarter/internal/runner"
)

func TestLogBatcherFlushesInOrderAtSizeLimit(t *testing.T) {
	batches := make(chan LogBatch, 1)
	b := newLogBatcher(time.Hour, 3, func(batch LogBatch) { batches <- batch })
	t.Cleanup(b.Close)

	b.Add(runner.LogLine{ID: "one", Stream: "stdout", Text: "first"})
	b.Add(runner.LogLine{ID: "two", Stream: "stderr", Text: "second"})
	b.Add(runner.LogLine{ID: "one", Stream: "stdout", Text: "third"})

	select {
	case batch := <-batches:
		if len(batch.Lines) != 3 {
			t.Fatalf("batch size = %d, want 3", len(batch.Lines))
		}
		for i, want := range []string{"first", "second", "third"} {
			if batch.Lines[i].Text != want {
				t.Fatalf("line %d = %q, want %q", i, batch.Lines[i].Text, want)
			}
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for size-triggered batch")
	}
}

func TestLogBatcherFlushesPartialBatchOnInterval(t *testing.T) {
	batches := make(chan LogBatch, 1)
	b := newLogBatcher(10*time.Millisecond, 10, func(batch LogBatch) { batches <- batch })
	t.Cleanup(b.Close)
	b.Add(runner.LogLine{ID: "one", Stream: "stdout", Text: "partial"})

	select {
	case batch := <-batches:
		if len(batch.Lines) != 1 || batch.Lines[0].Text != "partial" {
			t.Fatalf("batch = %+v, want one partial line", batch)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for interval-triggered batch")
	}
}

func TestLogBatcherCloseFlushesQueuedLines(t *testing.T) {
	batches := make(chan LogBatch, 1)
	b := newLogBatcher(time.Hour, 10, func(batch LogBatch) { batches <- batch })
	b.Add(runner.LogLine{ID: "one", Stream: "stderr", Text: "last"})
	b.Close()

	select {
	case batch := <-batches:
		if len(batch.Lines) != 1 || batch.Lines[0].Text != "last" {
			t.Fatalf("batch = %+v, want final queued line", batch)
		}
	default:
		t.Fatal("Close returned without flushing queued lines")
	}
}

func TestLogBatchJSONUsesFrontendContractFieldNames(t *testing.T) {
	data, err := json.Marshal(LogBatch{Lines: []runner.LogLine{{
		ID: "run", Stream: "stdout", Text: "hello",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{`"lines"`, `"id"`, `"stream"`, `"text"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("payload %s is missing %s", got, want)
		}
	}
}

func TestLogBatcherConcurrentAddAndCloseDoesNotBlock(t *testing.T) {
	b := newLogBatcher(time.Millisecond, 20, func(LogBatch) {})
	start := make(chan struct{})
	var adds sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		adds.Add(1)
		go func(worker int) {
			defer adds.Done()
			<-start
			for line := 0; line < 100; line++ {
				b.Add(runner.LogLine{ID: "run", Stream: "stdout", Text: "line"})
			}
		}(worker)
	}
	close(start)

	closed := make(chan struct{})
	go func() {
		b.Close()
		close(closed)
	}()
	adds.Wait()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("concurrent Add and Close did not complete within the shutdown budget")
	}
}
