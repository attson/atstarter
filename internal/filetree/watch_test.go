package filetree

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherFiresOnChange(t *testing.T) {
	root := t.TempDir()
	w, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	events := make(chan string, 8)
	w.OnChange(func(dir string) { events <- dir })

	id, err := w.Watch(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("want non-zero watch id")
	}

	// 触发变化
	if err := os.WriteFile(filepath.Join(root, "x.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case dir := <-events:
		// 监听的是 root(relPath=""),所以回调收到的 relDir 就是 ""(root 的相对路径)。
		if dir != "" {
			t.Errorf("want root relPath \"\", got %q", dir)
		}
	case <-time.After(2 * time.Second):
		t.Error("watcher did not fire within 2s")
	}
}

func TestWatcherFiresWithSubdirRelPath(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	events := make(chan string, 8)
	w.OnChange(func(dir string) { events <- dir })
	if _, err := w.Watch(root, "sub"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "y.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case dir := <-events:
		if dir != "sub" {
			t.Errorf("want changed relDir \"sub\", got %q", dir)
		}
	case <-time.After(2 * time.Second):
		t.Error("watcher did not fire within 2s")
	}
}

func TestWatcherUnwatch(t *testing.T) {
	root := t.TempDir()
	w, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	id, err := w.Watch(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Unwatch(id); err != nil {
		t.Errorf("unwatch failed: %v", err)
	}
}

func TestWatcherTraversalRejected(t *testing.T) {
	root := t.TempDir()
	w, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err := w.Watch(root, "../.."); err == nil {
		t.Error("want error watching path outside root")
	}
}
