//go:build !windows

package codexadapter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"rootyard/internal/projectenv"
)

func TestInspectRejectsFIFOWithoutBlocking(t *testing.T) {
	repo := makeRepository(t)
	dir := filepath.Join(repo, ".codex")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(filepath.Join(dir, "config.toml"), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := New()
	target, _ := adapter.Identify(context.Background(), repo)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := adapter.Inspect(ctx, target)
		done <- err
	}()
	select {
	case err := <-done:
		if !projectenv.IsCode(err, projectenv.ErrUnsafeCodexConfigPath) {
			t.Fatalf("err = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Inspect blocked while opening a FIFO")
	}
}
