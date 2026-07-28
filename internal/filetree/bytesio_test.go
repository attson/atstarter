package filetree

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadFileBytes(t *testing.T) {
	root := setupTree(t)
	fb, err := ReadFileBytes(root, "a.txt", 0)
	if err != nil {
		t.Fatal(err)
	}
	if string(fb.Data) != "hello" {
		t.Errorf("want hello, got %q", string(fb.Data))
	}
	if fb.IsBinary || fb.TruncatedAt != 0 {
		t.Errorf("unexpected: %+v", fb)
	}
	if fb.ModTime == 0 {
		t.Error("want non-zero ModTime")
	}
}

func TestReadFileBytesTruncated(t *testing.T) {
	root := t.TempDir()
	big := make([]byte, 100)
	for i := range big {
		big[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(root, "big"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	fb, err := ReadFileBytes(root, "big", 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(fb.Data) != 40 {
		t.Errorf("want 40 bytes, got %d", len(fb.Data))
	}
	if fb.TruncatedAt != 100 {
		t.Errorf("want TruncatedAt=100, got %d", fb.TruncatedAt)
	}
}

func TestReadFileBytesBinary(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bin"), []byte{0x1, 0x0, 0x2}, 0o644); err != nil {
		t.Fatal(err)
	}
	fb, err := ReadFileBytes(root, "bin", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !fb.IsBinary {
		t.Error("want IsBinary=true")
	}
}

func TestReadFileBytesTraversalRejected(t *testing.T) {
	root := setupTree(t)
	if _, err := ReadFileBytes(root, "../a.txt", 0); err == nil {
		t.Error("want error for traversal")
	}
}

func TestWriteFileBytesNew(t *testing.T) {
	root := setupTree(t)
	mt, err := WriteFileBytes(root, "created.txt", []byte("new content"), 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if mt == 0 {
		t.Error("want non-zero modtime")
	}
	got, err := os.ReadFile(filepath.Join(root, "created.txt"))
	if err != nil || string(got) != "new content" {
		t.Errorf("content mismatch: %q %v", string(got), err)
	}
}

func TestWriteFileBytesModTimeConflict(t *testing.T) {
	root := setupTree(t)
	// 先读取拿 modtime
	fb, err := ReadFileBytes(root, "a.txt", 0)
	if err != nil {
		t.Fatal(err)
	}
	// 用正确 modtime 写:成功,返回新 modtime
	newMT, err := WriteFileBytes(root, "a.txt", []byte("v2"), fb.ModTime, false)
	if err != nil {
		t.Fatalf("write with correct modtime should succeed: %v", err)
	}
	// 确保磁盘 modtime 前进(避免同毫秒),否则冲突检测无从触发。
	// 若首写后 modtime 未变(同毫秒),手动把磁盘 modtime 推后 10ms。
	if newMT == fb.ModTime {
		future := time.UnixMilli(fb.ModTime + 10)
		if err := os.Chtimes(filepath.Join(root, "a.txt"), future, future); err != nil {
			t.Fatal(err)
		}
	}
	// 用旧 modtime 再写:应 ErrStaleModTime
	_, err = WriteFileBytes(root, "a.txt", []byte("v3"), fb.ModTime, false)
	if !errors.Is(err, ErrStaleModTime) {
		t.Errorf("want ErrStaleModTime, got %v", err)
	}
}

func TestWriteFileBytesNotFoundNoCreate(t *testing.T) {
	root := setupTree(t)
	if _, err := WriteFileBytes(root, "nope.txt", []byte("x"), 0, false); err == nil {
		t.Error("want error writing missing file without createIfMissing")
	}
}

func TestWriteFileBytesTraversalRejected(t *testing.T) {
	root := setupTree(t)
	if _, err := WriteFileBytes(root, "../evil.txt", []byte("x"), 0, true); err == nil {
		t.Error("want error for traversal")
	}
}
