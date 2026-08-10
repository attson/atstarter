package main

import (
	"path/filepath"
	"testing"
)

// TestListPackageScriptsSorted 验证 scripts 按 key 字典序返回,且 name/script 一一对应。
// 排序放在后端是刻意选择:前端补全列表还要按前缀分组重排,拿到稳定顺序的输入
// 才能让 filterScripts 的组内顺序可预测、可测。
func TestListPackageScriptsSorted(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{
		"name": "demo",
		"scripts": {
			"ios:open": "react-native run-ios",
			"dev": "vite",
			"build": "vite build"
		}
	}`)

	got, err := app.ListPackageScripts(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []PackageScript{
		{Name: "build", Script: "vite build"},
		{Name: "dev", Script: "vite"},
		{Name: "ios:open", Script: "react-native run-ios"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d scripts %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("scripts[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestListPackageScriptsMissingOrInvalid 验证读不到内容时返回空切片而非错误。
// 补全是锦上添花的能力:目录还没填完、不是 Node 项目、package.json 正被编辑到
// 一半都属于常态,弹错误提示只会打断用户输入。
func TestListPackageScriptsMissingOrInvalid(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir()

	cases := []struct {
		name  string
		setup func() string
	}{
		{"目录不存在", func() string { return filepath.Join(dir, "nope") }},
		{"无 package.json", func() string { return dir }},
		{"JSON 非法", func() string {
			d := t.TempDir()
			writeFile(t, filepath.Join(d, "package.json"), `{"scripts": {`)
			return d
		}},
		{"无 scripts 字段", func() string {
			d := t.TempDir()
			writeFile(t, filepath.Join(d, "package.json"), `{"name":"x"}`)
			return d
		}},
		{"空目录参数", func() string { return "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := app.ListPackageScripts(tc.setup())
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if len(got) != 0 {
				t.Fatalf("got %v, want empty", got)
			}
		})
	}
}

// TestListPackageScriptsOnlyReadsPackageJSON 验证只读取 dir 下固定文件名的
// package.json,不会因为 dir 里带 ".." 之类内容而越界读别的文件。
func TestListPackageScriptsOnlyReadsPackageJSON(t *testing.T) {
	app := newTestApp(t)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), `{"scripts":{"root":"echo root"}}`)
	writeFile(t, filepath.Join(root, "sub", "package.json"), `{"scripts":{"sub":"echo sub"}}`)

	got, err := app.ListPackageScripts(filepath.Join(root, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "sub" {
		t.Fatalf("got %v, want only sub script", got)
	}
}
