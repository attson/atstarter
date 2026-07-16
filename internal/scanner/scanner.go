// Package scanner 遍历工作区根目录的直接子目录,对每个调用 detector,
// 产出候选 store.Project 列表(command/args 已拆分,ID 已生成)。
package scanner

import (
	"os"
	"path/filepath"

	"atstarter/internal/cmdparse"
	"atstarter/internal/detector"
	"atstarter/internal/store"
)

// Scan 扫描每个 root 的直接子目录。识别为 unknown 的也会列出(命令留空)。
// 无法读取的 root 被静默跳过。
func Scan(roots []string) []store.Project {
	var out []store.Project
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue // 跳过不存在/不可读的 root
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(root, e.Name())
			res := detector.Detect(dir)
			p := store.Project{
				ID:           store.IDForPath(dir),
				Name:         e.Name(),
				Path:         dir,
				DetectedType: res.Type,
				AutoDetected: true,
			}
			if res.Command != "" {
				if cmd, args, err := cmdparse.Parse(res.Command); err == nil {
					p.Command = cmd
					p.Args = args
				}
			}
			out = append(out, p)
		}
	}
	return out
}
