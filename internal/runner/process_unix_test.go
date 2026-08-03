//go:build !windows

package runner

import "testing"

func TestShellExportValue(t *testing.T) {
	// FOO 用于验证非 PATH 的 $VAR 仍由 Go 展开;PATH 引用则保留给 shell。
	t.Setenv("FOO", "/real/foo")

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "append PATH keeps shell reference",
			in:   "/go/bin:$PATH",
			want: `'/go/bin:'"$PATH"`,
		},
		{
			name: "brace form PATH keeps shell reference",
			in:   "/go/bin:${PATH}",
			want: `'/go/bin:'"$PATH"`,
		},
		{
			name: "no PATH reference stays fully quoted (override semantics)",
			in:   "/fixed/only",
			want: `'/fixed/only'`,
		},
		{
			name: "other vars expanded by Go, PATH left for shell",
			in:   "$FOO:$PATH",
			want: `'/real/foo:'"$PATH"`,
		},
		{
			name: "PATHEXTRA is not treated as PATH",
			in:   "/x:$PATHEXTRA",
			want: `'/x:'`, // $PATHEXTRA expands to empty via Go (unset), not preserved
		},
		{
			name: "PATH reference in the middle",
			in:   "$PATH:/tail",
			want: `"$PATH"':/tail'`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shellExportValue(tc.in); got != tc.want {
				t.Errorf("shellExportValue(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIndexPathRef(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"$PATH", 0},
		{"a:$PATH", 2},
		{"${PATH}", 0},
		{"$PATHEXTRA", -1},
		{"$PATH2", -1},
		{"no ref", -1},
		{"$FOO", -1},
	}
	for _, tc := range cases {
		if got := indexPathRef(tc.in); got != tc.want {
			t.Errorf("indexPathRef(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
