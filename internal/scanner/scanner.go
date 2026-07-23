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
	seen := map[string]bool{}
	for _, root := range roots {
		children := scanChildren(root, seen, &out, true)
		scanWorktreeRoots(root, seen, &out)
		for _, child := range children {
			scanWorktreeRoots(child, seen, &out)
		}
	}
	return out
}

func scanWorktreeRoots(root string, seen map[string]bool, out *[]store.Project) {
	scanChildren(filepath.Join(root, ".worktrees"), seen, out, false)
	scanChildren(filepath.Join(root, ".claude", "worktrees"), seen, out, false)
}

func scanChildren(root string, seen map[string]bool, out *[]store.Project, skipWorktreeContainers bool) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil // 跳过不存在/不可读的 root
	}
	var added []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if skipWorktreeContainers && (e.Name() == ".worktrees" || e.Name() == ".claude") {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if seen[dir] {
			continue
		}
		seen[dir] = true
		*out = append(*out, projectForDir(dir, e.Name()))
		added = append(added, dir)
	}
	return added
}

func projectForDir(dir, name string) store.Project {
	res := detector.Detect(dir)
	p := store.Project{
		ID:           store.IDForPath(dir),
		Name:         name,
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
	p.DetectionOptions = detectionOptionsForDir(dir)
	return store.NormalizeProjectCommands(p)
}

// AddDetectionOptions 填充同一路径可切换的识别结果,不改变项目当前类型与命令。
func AddDetectionOptions(p store.Project) store.Project {
	if p.Path == "" {
		return p
	}
	p.DetectionOptions = detectionOptionsForDir(p.Path)
	return p
}

func detectionOptionsForDir(dir string) []store.DetectionOption {
	results := detector.DetectOptions(dir)
	options := make([]store.DetectionOption, 0, len(results))
	for _, res := range results {
		option := store.DetectionOption{Type: res.Type, Args: []string{}}
		if res.Command != "" {
			if cmd, args, err := cmdparse.Parse(res.Command); err == nil {
				option.Command = cmd
				option.Args = args
			}
		}
		options = append(options, option)
	}
	if len(options) <= 1 {
		return nil
	}
	return options
}
