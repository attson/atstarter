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
			if got := shellExportValue(tc.in, nil); got != tc.want {
				t.Errorf("shellExportValue(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestShellExportsCrossReferencesUserVar 锁定核心修复:用户在同一份 env 里定义的
// 变量(如 JAVA_HOME)应能被同份 env 里后续值(PATH=$JAVA_HOME/bin:$PATH)引用到,
// 由 shell 在 rc 之后展开;且被引用者的 export 必须排在引用者之前。
func TestShellExportsCrossReferencesUserVar(t *testing.T) {
	env := map[string]string{
		"JAVA_HOME": "/home/u/.jdks/corretto-17",
		"PATH":      "$JAVA_HOME/bin:/opt/maven/bin:$PATH",
	}
	got := shellExports(env)
	want := `export 'JAVA_HOME'='/home/u/.jdks/corretto-17'; ` +
		`export 'PATH'="$JAVA_HOME"'/bin:/opt/maven/bin:'"$PATH"`
	if got != want {
		t.Errorf("shellExports cross-ref\n got = %q\nwant = %q", got, want)
	}
}

// TestShellExportValueKeepsUserVars 验证 userVars 集合内的变量引用留给 shell 展开,
// 集合外的普通变量仍由 Go 用进程环境展开(向后兼容)。
func TestShellExportValueKeepsUserVars(t *testing.T) {
	t.Setenv("FOO", "/real/foo")
	userVars := map[string]bool{"JAVA_HOME": true}

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"user var kept for shell", "$JAVA_HOME/bin:$PATH", `"$JAVA_HOME"'/bin:'"$PATH"`},
		{"brace user var kept", "${JAVA_HOME}/bin", `"$JAVA_HOME"'/bin'`},
		{"non-user var still Go-expanded", "$FOO:$PATH", `'/real/foo:'"$PATH"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shellExportValue(tc.in, userVars); got != tc.want {
				t.Errorf("shellExportValue(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIndexVarRef(t *testing.T) {
	names := map[string]bool{"PATH": true}
	cases := []struct {
		in       string
		wantIdx  int
		wantLen  int
		wantName string
	}{
		{"$PATH", 0, 5, "PATH"},
		{"a:$PATH", 2, 5, "PATH"},
		{"${PATH}", 0, 7, "PATH"},
		{"$PATHEXTRA", -1, 0, ""}, // 后跟标识符字符,不是 $PATH
		{"$PATH2", -1, 0, ""},
		{"no ref", -1, 0, ""},
		{"$FOO", -1, 0, ""}, // 不在 names 集合内
	}
	for _, tc := range cases {
		idx, n, name := indexVarRef(tc.in, names)
		if idx != tc.wantIdx || n != tc.wantLen || name != tc.wantName {
			t.Errorf("indexVarRef(%q) = (%d,%d,%q), want (%d,%d,%q)",
				tc.in, idx, n, name, tc.wantIdx, tc.wantLen, tc.wantName)
		}
	}
}
