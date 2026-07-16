package detector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// exists 判断项目 dir 下的相对路径是否存在。
func exists(dir, rel string) bool {
	_, err := os.Stat(filepath.Join(dir, rel))
	return err == nil
}

// readScripts 读取 package.json 的 scripts 字段;失败返回 nil。
func readScripts(dir string) map[string]string {
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return nil
	}
	return pkg.Scripts
}

// pickNodeScript 依次挑选存在的脚本名,优先 dev,其次 serve、start;都没有则返回 "dev"(兜底)。
func pickNodeScript(dir string) string {
	scripts := readScripts(dir)
	for _, name := range []string{"dev", "serve", "start"} {
		if _, ok := scripts[name]; ok {
			return name
		}
	}
	return "dev"
}

// firstCmdMain 返回按字母序第一个含 main.go 的 cmd/<name> 目录名;无则返回 ""。
func firstCmdMain(dir string) string {
	entries, err := os.ReadDir(filepath.Join(dir, "cmd"))
	if err != nil {
		return ""
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && exists(dir, filepath.Join("cmd", e.Name(), "main.go")) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// firstExisting 返回候选相对路径中第一个存在的;都不存在返回 ""。
func firstExisting(dir string, candidates ...string) string {
	for _, c := range candidates {
		if exists(dir, c) {
			return c
		}
	}
	return ""
}
