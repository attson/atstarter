package filetree

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestUniqueNameKeepsFreeName(t *testing.T) {
	root := t.TempDir()
	got, err := UniqueName(root, "fresh.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "fresh.txt" {
		t.Errorf("want fresh.txt unchanged, got %q", got)
	}
}

func TestUniqueNameSuffixSequence(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.go"), "x", 0o644)

	first, err := UniqueName(root, "app.go")
	if err != nil {
		t.Fatal(err)
	}
	if first != "app copy.go" {
		t.Fatalf("want \"app copy.go\", got %q", first)
	}

	writeFile(t, filepath.Join(root, first), "x", 0o644)
	second, err := UniqueName(root, "app.go")
	if err != nil {
		t.Fatal(err)
	}
	if second != "app copy 2.go" {
		t.Fatalf("want \"app copy 2.go\", got %q", second)
	}
}

func TestUniqueNameDotfileHasNoExtension(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "x", 0o644)
	got, err := UniqueName(root, ".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	if got != ".gitignore copy" {
		t.Errorf("want \".gitignore copy\", got %q", got)
	}
}

func TestUniqueNameDirectoryKeepsDotsInStem(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "my.dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := UniqueName(root, "my.dir")
	if err != nil {
		t.Fatal(err)
	}
	if got != "my.dir copy" {
		t.Errorf("want \"my.dir copy\", got %q", got)
	}
}

func TestUniqueNameNested(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "sub", "b.txt"), "x", 0o644)
	got, err := UniqueName(root, "sub/b.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sub/b copy.txt" {
		t.Errorf("want \"sub/b copy.txt\", got %q", got)
	}
}

func TestUniqueNameRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := UniqueName(root, "../escape.txt"); err == nil {
		t.Fatal("want traversal error, got nil")
	}
}

func TestCopyFileSameRoot(t *testing.T) {
	root := setupTree(t)
	got, err := Copy(root, "a.txt", root, "sub/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sub/a.txt" {
		t.Fatalf("want sub/a.txt, got %q", got)
	}
	if content := readFile(t, filepath.Join(root, "sub", "a.txt")); content != "hello" {
		t.Errorf("want copied content hello, got %q", content)
	}
	if content := readFile(t, filepath.Join(root, "a.txt")); content != "hello" {
		t.Errorf("source must stay intact, got %q", content)
	}
}

func TestCopyNeverOverwrites(t *testing.T) {
	root := setupTree(t)
	// 同目录复制一份 = 目标与源同名,必须自动改名而不是覆盖。
	got, err := Copy(root, "a.txt", root, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "a copy.txt" {
		t.Fatalf("want \"a copy.txt\", got %q", got)
	}
	if content := readFile(t, filepath.Join(root, "a.txt")); content != "hello" {
		t.Errorf("original must be untouched, got %q", content)
	}
	if content := readFile(t, filepath.Join(root, "a copy.txt")); content != "hello" {
		t.Errorf("want copy content hello, got %q", content)
	}
}

func TestCopyPreservesPermissions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "run.sh"), "#!/bin/sh\n", 0o755)
	if _, err := Copy(root, "run.sh", root, "run2.sh"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(root, "run2.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("want perm 0755, got %v", info.Mode().Perm())
	}
}

func TestCopyDirectoryRecursive(t *testing.T) {
	root := setupTree(t)
	if err := os.Mkdir(filepath.Join(root, "sub", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "sub", "deep", "c.txt"), "deep", 0o644)

	got, err := Copy(root, "sub", root, "subcopy")
	if err != nil {
		t.Fatal(err)
	}
	if got != "subcopy" {
		t.Fatalf("want subcopy, got %q", got)
	}
	if content := readFile(t, filepath.Join(root, "subcopy", "b.txt")); content != "world" {
		t.Errorf("want world, got %q", content)
	}
	if content := readFile(t, filepath.Join(root, "subcopy", "deep", "c.txt")); content != "deep" {
		t.Errorf("want deep, got %q", content)
	}
}

func TestCopyAcrossRoots(t *testing.T) {
	src := setupTree(t)
	dst := t.TempDir()
	got, err := Copy(src, "sub", dst, "imported")
	if err != nil {
		t.Fatal(err)
	}
	if got != "imported" {
		t.Fatalf("want imported, got %q", got)
	}
	if content := readFile(t, filepath.Join(dst, "imported", "b.txt")); content != "world" {
		t.Errorf("want world, got %q", content)
	}
}

func TestCopyRejectsTraversalOnEitherSide(t *testing.T) {
	src := setupTree(t)
	dst := t.TempDir()
	if _, err := Copy(src, "../outside.txt", dst, "x.txt"); err == nil {
		t.Error("want source traversal rejected")
	}
	if _, err := Copy(src, "a.txt", dst, "../outside.txt"); err == nil {
		t.Error("want destination traversal rejected")
	}
}

func TestCopyRejectsDirectoryIntoItself(t *testing.T) {
	root := setupTree(t)
	_, err := Copy(root, "sub", root, "sub/inner")
	if !errors.Is(err, ErrCopyIntoSelf) {
		t.Fatalf("want ErrCopyIntoSelf, got %v", err)
	}
}

func TestCopyDoesNotFollowSymlinks(t *testing.T) {
	root := setupTree(t)
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(filepath.Join(root, "a.txt"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Copy(root, "link.txt", root, "link2.txt"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(root, "link2.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("want the copy to stay a symlink, not a dereferenced file")
	}
}

func TestCopyRefusesTooManyEntries(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "big"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "big")
	dst := filepath.Join(root, "big-copy")
	info, err := os.Lstat(src)
	if err != nil {
		t.Fatal(err)
	}
	// 直接把计数器推到上限,避免真的造 5 万个文件。
	count := maxCopyEntries
	if err := copyPath(src, dst, info, &count); !errors.Is(err, ErrCopyTooManyEntries) {
		t.Fatalf("want ErrCopyTooManyEntries, got %v", err)
	}
}

func TestMoveSameRoot(t *testing.T) {
	root := setupTree(t)
	got, err := Move(root, "a.txt", root, "sub/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sub/a.txt" {
		t.Fatalf("want sub/a.txt, got %q", got)
	}
	if _, err := os.Lstat(filepath.Join(root, "a.txt")); !os.IsNotExist(err) {
		t.Error("source must be gone after move")
	}
	if content := readFile(t, filepath.Join(root, "sub", "a.txt")); content != "hello" {
		t.Errorf("want hello, got %q", content)
	}
}

func TestMoveAcrossRootsLeavesNothingBehind(t *testing.T) {
	src := setupTree(t)
	dst := t.TempDir()
	got, err := Move(src, "sub", dst, "sub")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sub" {
		t.Fatalf("want sub, got %q", got)
	}
	if _, err := os.Lstat(filepath.Join(src, "sub")); !os.IsNotExist(err) {
		t.Error("source dir must be gone after cross-root move")
	}
	if content := readFile(t, filepath.Join(dst, "sub", "b.txt")); content != "world" {
		t.Errorf("want world, got %q", content)
	}
}

func TestMoveRenamesInsteadOfOverwriting(t *testing.T) {
	root := setupTree(t)
	writeFile(t, filepath.Join(root, "sub", "a.txt"), "existing", 0o644)
	got, err := Move(root, "a.txt", root, "sub/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sub/a copy.txt" {
		t.Fatalf("want \"sub/a copy.txt\", got %q", got)
	}
	if content := readFile(t, filepath.Join(root, "sub", "a.txt")); content != "existing" {
		t.Errorf("existing file must survive, got %q", content)
	}
}

func TestMoveRejectsSameSourceAndDestination(t *testing.T) {
	root := setupTree(t)
	if _, err := Move(root, "a.txt", root, "a.txt"); err == nil {
		t.Fatal("want error when moving onto itself")
	}
}

func TestMoveRejectsDirectoryIntoItself(t *testing.T) {
	root := setupTree(t)
	_, err := Move(root, "sub", root, "sub/inner")
	if !errors.Is(err, ErrCopyIntoSelf) {
		t.Fatalf("want ErrCopyIntoSelf, got %v", err)
	}
}
