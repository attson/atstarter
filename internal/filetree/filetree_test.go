package filetree

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	// root/
	//   a.txt
	//   sub/
	//     b.txt
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestListDirRootDirsFirst(t *testing.T) {
	root := setupTree(t)
	entries, err := ListDir(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if !entries[0].IsDir || entries[0].Name != "sub" {
		t.Errorf("want sub dir first, got %+v", entries[0])
	}
	if entries[1].Name != "a.txt" || entries[1].IsDir {
		t.Errorf("want a.txt second, got %+v", entries[1])
	}
	if entries[1].Size != 5 {
		t.Errorf("want size 5 for a.txt, got %d", entries[1].Size)
	}
}

func TestListDirSubPath(t *testing.T) {
	root := setupTree(t)
	entries, err := ListDir(root, "sub")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "b.txt" {
		t.Fatalf("want [b.txt], got %+v", entries)
	}
}

func TestListDirTraversalRejected(t *testing.T) {
	root := setupTree(t)
	for _, rel := range []string{"../", "../..", "sub/../../etc"} {
		if _, err := ListDir(root, rel); err == nil {
			t.Errorf("rel %q: want error, got nil", rel)
		}
	}
}

func TestListDirNotExist(t *testing.T) {
	root := setupTree(t)
	if _, err := ListDir(root, "nope"); err == nil {
		t.Error("want error for non-existent path")
	}
}

func TestReadFileText(t *testing.T) {
	root := setupTree(t)
	fc, err := ReadFile(root, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if fc.Content != "hello" || fc.Binary || fc.Truncated {
		t.Errorf("unexpected: %+v", fc)
	}
	if fc.Size != 5 {
		t.Errorf("want size 5, got %d", fc.Size)
	}
}

func TestReadFileBinary(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bin"), []byte{0x1, 0x0, 0x2}, 0o644); err != nil {
		t.Fatal(err)
	}
	fc, err := ReadFile(root, "bin")
	if err != nil {
		t.Fatal(err)
	}
	if !fc.Binary || fc.Content != "" {
		t.Errorf("want binary with empty content, got %+v", fc)
	}
}

func TestReadFileTruncated(t *testing.T) {
	root := t.TempDir()
	big := make([]byte, (1<<20)+100) // 1MB + 100
	for i := range big {
		big[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(root, "big.txt"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	fc, err := ReadFile(root, "big.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !fc.Truncated {
		t.Error("want Truncated=true")
	}
	if len(fc.Content) != (1 << 20) {
		t.Errorf("want content len 1MB, got %d", len(fc.Content))
	}
	if fc.Size != (1<<20)+100 {
		t.Errorf("want size 1MB+100, got %d", fc.Size)
	}
}

func TestReadFileTraversalRejected(t *testing.T) {
	root := setupTree(t)
	if _, err := ReadFile(root, "../a.txt"); err == nil {
		t.Error("want error for traversal")
	}
}

func TestReadFileIsDir(t *testing.T) {
	root := setupTree(t)
	if _, err := ReadFile(root, "sub"); err == nil {
		t.Error("want error when reading a directory")
	}
}
