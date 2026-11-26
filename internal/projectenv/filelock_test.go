package projectenv

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestFileLockSerializesCallers(t *testing.T) {
	lockPath := t.TempDir() + "/environment.lock"
	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- withFileLock(context.Background(), lockPath, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	var mu sync.Mutex
	secondEntered := false
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- withFileLock(context.Background(), lockPath, func() error {
			mu.Lock()
			secondEntered = true
			mu.Unlock()
			return nil
		})
	}()
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	enteredEarly := secondEntered
	mu.Unlock()
	if enteredEarly {
		t.Fatal("second caller entered while first held lock")
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := <-secondDone; err != nil {
		t.Fatal(err)
	}
}

func TestFileLockHonorsContextTimeout(t *testing.T) {
	lockPath := t.TempDir() + "/environment.lock"
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- withFileLock(context.Background(), lockPath, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := withFileLock(ctx, lockPath, func() error {
		t.Fatal("timed-out lock callback ran")
		return nil
	})
	if !IsCode(err, ErrLockTimeout) {
		t.Fatalf("err = %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
