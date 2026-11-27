package codexadapter

import (
	"bytes"
	"testing"

	"rootyard/internal/projectenv"
)

func TestApplyTextEditsReplaysDescendingNonOverlappingRanges(t *testing.T) {
	original := []byte("alpha beta gamma")
	edits := []projectenv.TextEdit{
		{Start: 6, End: 10, Replacement: "B"},
		{Start: 0, End: 5, Replacement: "A"},
	}
	got, err := applyTextEdits(original, edits)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "A B gamma" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyTextEditsRejectsOverlappingAndOutOfBoundsRanges(t *testing.T) {
	tests := []struct {
		name  string
		edits []projectenv.TextEdit
	}{
		{name: "overlap", edits: []projectenv.TextEdit{{Start: 1, End: 4}, {Start: 3, End: 5}}},
		{name: "negative", edits: []projectenv.TextEdit{{Start: -1, End: 1}}},
		{name: "reverse", edits: []projectenv.TextEdit{{Start: 3, End: 2}}},
		{name: "past end", edits: []projectenv.TextEdit{{Start: 2, End: 8}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := applyTextEdits([]byte("value"), tt.edits); err == nil {
				t.Fatal("invalid edits were accepted")
			}
		})
	}
}

func TestTOMLBasicStringEscapesFreeFormValues(t *testing.T) {
	got := tomlBasicString("a\\b\"c\tline\nend")
	want := `"a\\b\"c\tline\nend"`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	parsed, err := parseDocument([]byte("model = " + got + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Assignments["model"].Value != "a\\b\"c\tline\nend" {
		t.Fatalf("decoded = %q", parsed.Assignments["model"].Value)
	}
}

func assertBytesEqual(t *testing.T, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Fatalf("bytes differ\ngot:  %q\nwant: %q", got, want)
	}
}
