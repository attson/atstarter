package detector

import (
	"bytes"
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

// ReadScripts 读取 dir 下 package.json 的 scripts 字段;读不到或解析失败返回 nil。
// 只读取固定文件名 package.json,不遍历、不递归。
func ReadScripts(dir string) map[string]string {
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
	scripts := ReadScripts(dir)
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

// mavenCommand 依据是否 Spring Boot、是否带 mvnw wrapper,给出建议启动命令。
// 非 Spring Boot 的普通 Maven 项目只给构建命令,启动方式交由用户覆盖。
func mavenCommand(dir string) string {
	mvn := "mvn"
	if exists(dir, "mvnw") {
		mvn = "./mvnw"
	}
	if fileContains(dir, "pom.xml", "spring-boot") {
		return mvn + " spring-boot:run"
	}
	return mvn + " package"
}

// gradleCommand 依据是否 Spring Boot、是否带 gradlew wrapper,给出建议启动命令。
func gradleCommand(dir string) string {
	gradle := "gradle"
	if exists(dir, "gradlew") {
		gradle = "./gradlew"
	}
	if fileContains(dir, "build.gradle", "org.springframework.boot") ||
		fileContains(dir, "build.gradle.kts", "org.springframework.boot") {
		return gradle + " bootRun"
	}
	return gradle + " run"
}

// fileContains 判断 dir 下相对文件是否存在且内容包含 sub;读不到返回 false。
func fileContains(dir, rel, sub string) bool {
	b, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		return false
	}
	return bytes.Contains(b, []byte(sub))
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
