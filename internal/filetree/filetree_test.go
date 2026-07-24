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
