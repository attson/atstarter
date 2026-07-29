package filetree

import (
	"errors"
	"fmt"
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
	big := make([]byte, maxReadBytes+100) // 超过预览上限
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
	if len(fc.Content) != maxReadBytes {
		t.Errorf("want content len %d, got %d", maxReadBytes, len(fc.Content))
	}
	if fc.Size != maxReadBytes+100 {
		t.Errorf("want size %d, got %d", maxReadBytes+100, fc.Size)
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

func TestSearchPathsMatchesNamesAndRelativePaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"cmd/server/main.go",
		"cmd/server/config.yml",
		"docs/server-guide.md",
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	matches, truncated, err := SearchPaths(root, "server", 20)
	if err != nil {
		t.Fatal(err)
	}
	if truncated {
		t.Fatal("small search should not be truncated")
	}
	got := make([]string, 0, len(matches))
	for _, match := range matches {
		got = append(got, match.Path)
	}
	for _, want := range []string{"cmd/server/", "cmd/server/main.go", "cmd/server/config.yml", "docs/server-guide.md"} {
		if !containsString(got, want) {
			t.Errorf("want %q in %v", want, got)
		}
	}
}

func TestSearchPathsSkipsHeavyDirectories(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "objects"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "Vendor", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		".git/config",
		"node_modules/pkg/config.js",
		"Vendor/pkg/config.go",
		"src/config.go",
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	matches, _, err := SearchPaths(root, "config", 20)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(matches))
	for _, match := range matches {
		got = append(got, match.Path)
	}
	if !containsString(got, "src/config.go") {
		t.Errorf("want src/config.go in %v", got)
	}
	for _, skipped := range []string{".git/config", "node_modules/pkg/config.js", "Vendor/pkg/config.go"} {
		if containsString(got, skipped) {
			t.Errorf("did not expect skipped path %q in %v", skipped, got)
		}
	}
}

func TestSearchPathsTruncatesResults(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		path := filepath.Join(root, "match-"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	matches, truncated, err := SearchPaths(root, "match", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Fatal("want truncated=true")
	}
	if len(matches) != 3 {
		t.Fatalf("want 3 matches, got %d: %v", len(matches), matches)
	}
}

func TestSearchPathsClampsCallerLimit(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < defaultSearchLimit+5; i++ {
		path := filepath.Join(root, fmt.Sprintf("match-%03d.txt", i))
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	matches, truncated, err := SearchPaths(root, "match", defaultSearchLimit+1000)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Fatal("want truncated=true when caller asks above backend cap")
	}
	if len(matches) != defaultSearchLimit {
		t.Fatalf("want backend cap %d matches, got %d", defaultSearchLimit, len(matches))
	}
}

func TestSearchPathsTruncatesWhenVisitedEntriesExceedLimit(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		path := filepath.Join(root, "file-"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	matches, truncated, err := searchPaths(root, "no-match", 100, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Fatal("want truncated=true when visited entries exceed backend cap")
	}
	if len(matches) != 0 {
		t.Fatalf("want no matches before visit cap, got %v", matches)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
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
	big := make([]byte, maxReadBytes+100) // 超过写入上限
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

func TestFileMetaText(t *testing.T) {
	root := setupTree(t)
	m, err := FileMeta(root, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if m.Size != 5 || m.IsBinary || m.IsDir {
		t.Errorf("unexpected meta: %+v", m)
	}
	if m.ModTime == 0 {
		t.Error("want non-zero ModTime")
	}
}

func TestFileMetaBinary(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bin"), []byte{0x1, 0x0, 0x2}, 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := FileMeta(root, "bin")
	if err != nil {
		t.Fatal(err)
	}
	if !m.IsBinary {
		t.Error("want IsBinary=true")
	}
}

func TestFileMetaDir(t *testing.T) {
	root := setupTree(t)
	m, err := FileMeta(root, "sub")
	if err != nil {
		t.Fatal(err)
	}
	if !m.IsDir {
		t.Error("want IsDir=true for directory")
	}
}

func TestFileMetaTraversalRejected(t *testing.T) {
	root := setupTree(t)
	if _, err := FileMeta(root, "../a.txt"); err == nil {
		t.Error("want error for traversal")
	}
}

func TestCreateFile(t *testing.T) {
	root := setupTree(t)
	if err := CreateFile(root, "new.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "new.txt")); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestCreateFileAlreadyExists(t *testing.T) {
	root := setupTree(t)
	if err := CreateFile(root, "a.txt"); err == nil {
		t.Error("want error creating existing file")
	}
}

func TestCreateFileTraversalRejected(t *testing.T) {
	root := setupTree(t)
	if err := CreateFile(root, "../x.txt"); err == nil {
		t.Error("want error for traversal")
	}
}

func TestMkdir(t *testing.T) {
	root := setupTree(t)
	if err := Mkdir(root, "newdir"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(root, "newdir"))
	if err != nil || !info.IsDir() {
		t.Errorf("dir not created: %v", err)
	}
}

func TestMkdirAlreadyExists(t *testing.T) {
	root := setupTree(t)
	if err := Mkdir(root, "sub"); err == nil {
		t.Error("want error creating existing dir")
	}
}

func TestRename(t *testing.T) {
	root := setupTree(t)
	if err := Rename(root, "a.txt", "renamed.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "renamed.txt")); err != nil {
		t.Errorf("renamed file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); !os.IsNotExist(err) {
		t.Error("original should be gone")
	}
}

func TestRenameSrcMissing(t *testing.T) {
	root := setupTree(t)
	if err := Rename(root, "nope.txt", "x.txt"); err == nil {
		t.Error("want error renaming missing src")
	}
}

func TestRenameDstExists(t *testing.T) {
	root := setupTree(t)
	if err := Rename(root, "a.txt", "a.txt"); err == nil {
		t.Error("want error when dst exists")
	}
}

func TestRenameTraversalRejected(t *testing.T) {
	root := setupTree(t)
	if err := Rename(root, "a.txt", "../evil.txt"); err == nil {
		t.Error("want error for traversal on dst")
	}
}

func TestRemoveFile(t *testing.T) {
	root := setupTree(t)
	if err := Remove(root, "a.txt", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); !os.IsNotExist(err) {
		t.Error("file should be removed")
	}
}

func TestRemoveDirRecursive(t *testing.T) {
	root := setupTree(t)
	if err := Remove(root, "sub", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "sub")); !os.IsNotExist(err) {
		t.Error("dir should be removed")
	}
}

func TestRemoveNonEmptyDirNonRecursive(t *testing.T) {
	root := setupTree(t)
	if err := Remove(root, "sub", false); err == nil {
		t.Error("want error removing non-empty dir without recursive")
	}
}

func TestRemoveMissing(t *testing.T) {
	root := setupTree(t)
	if err := Remove(root, "nope", false); err == nil {
		t.Error("want error removing missing path")
	}
}

func TestRemoveTraversalRejected(t *testing.T) {
	root := setupTree(t)
	if err := Remove(root, "../a.txt", false); err == nil {
		t.Error("want error for traversal")
	}
}

func TestTrashMissing(t *testing.T) {
	root := setupTree(t)
	if err := Trash(root, "nope"); err == nil {
		t.Error("want error trashing missing path")
	}
}

func TestTrashTraversalRejected(t *testing.T) {
	root := setupTree(t)
	if err := Trash(root, "../a.txt"); err == nil {
		t.Error("want error for traversal")
	}
}

func TestTrashUnavailableIsDetectable(t *testing.T) {
	// 存在的文件:trash 要么成功,要么返回可识别的 ErrTrashUnavailable。
	root := setupTree(t)
	err := Trash(root, "a.txt")
	if err != nil && !errors.Is(err, ErrTrashUnavailable) {
		t.Fatalf("unexpected trash error (want nil or ErrTrashUnavailable): %v", err)
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
