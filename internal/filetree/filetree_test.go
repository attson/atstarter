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

func TestWalkPathsFullTree(t *testing.T) {
	root := setupTree(t)
	paths, truncated, err := WalkPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if truncated {
		t.Error("small tree should not be truncated")
	}
	// 期望全量:a.txt、sub/(带尾斜杠)、sub/b.txt。顺序按 WalkDir 的字典序。
	got := map[string]bool{}
	for _, p := range paths {
		got[p] = true
	}
	for _, want := range []string{"a.txt", "sub/", "sub/b.txt"} {
		if !got[want] {
			t.Errorf("missing path %q in %v", want, paths)
		}
	}
	if got["sub"] {
		t.Error("directory must carry trailing slash, got bare \"sub\"")
	}
	if len(paths) != 3 {
		t.Errorf("want 3 paths, got %d: %v", len(paths), paths)
	}
}

func TestWalkPathsUsesForwardSlash(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "c.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths, _, err := WalkPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if filepath.Separator != '/' && containsBackslash(p) {
			t.Errorf("path must use forward slashes, got %q", p)
		}
	}
	// 深层文件应以 / 分隔的相对路径出现。
	want := "a/b/c.txt"
	found := false
	for _, p := range paths {
		if p == want {
			found = true
		}
	}
	if !found {
		t.Errorf("want %q in %v", want, paths)
	}
}

func TestWalkPathsEmptyDir(t *testing.T) {
	root := t.TempDir()
	paths, truncated, err := WalkPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if truncated {
		t.Error("empty dir should not be truncated")
	}
	if len(paths) != 0 {
		t.Errorf("empty dir should yield no paths, got %v", paths)
	}
}

func TestWalkPathsTruncates(t *testing.T) {
	root := t.TempDir()
	// 建 5 个文件,用注入的小上限 3 验证截断:命中上限即停,列表不超过上限。
	for i := 0; i < 5; i++ {
		name := filepath.Join(root, string(rune('a'+i))+".txt")
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	paths, truncated, err := walkPaths(root, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Error("want truncated=true when entries exceed the limit")
	}
	if len(paths) != 3 {
		t.Errorf("want exactly 3 paths at the limit, got %d: %v", len(paths), paths)
	}
}

func TestWriteFileText(t *testing.T) {
	root := setupTree(t)
	if err := WriteFile(root, "a.txt", "updated content"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "updated content" {
		t.Errorf("want %q, got %q", "updated content", string(got))
	}
}

func TestWriteFilePreservesPerm(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "script.sh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(root, "script.sh", "#!/bin/sh\necho hi\n"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("want perm 0755 preserved, got %o", info.Mode().Perm())
	}
}

func TestWriteFileTraversalRejected(t *testing.T) {
	root := setupTree(t)
	if err := WriteFile(root, "../a.txt", "x"); err == nil {
		t.Error("want error for traversal")
	}
}

func TestWriteFileBinaryRejected(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bin"), []byte{0x1, 0x0, 0x2}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(root, "bin", "text"); err == nil {
		t.Error("want error writing over a binary file")
	}
}

func TestWriteFileTooLargeRejected(t *testing.T) {
	root := t.TempDir()
	big := make([]byte, (1<<20)+100) // 1MB + 100
	for i := range big {
		big[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(root, "big.txt"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(root, "big.txt", "small"); err == nil {
		t.Error("want error writing over a too-large file")
	}
}

func TestWriteFileIsDirRejected(t *testing.T) {
	root := setupTree(t)
	if err := WriteFile(root, "sub", "x"); err == nil {
		t.Error("want error writing to a directory")
	}
}

func TestWriteFileNotExistRejected(t *testing.T) {
	root := setupTree(t)
	if err := WriteFile(root, "nope.txt", "x"); err == nil {
		t.Error("want error writing a non-existent file")
	}
}

func containsBackslash(s string) bool {
	for _, r := range s {
		if r == '\\' {
			return true
		}
	}
	return false
}
