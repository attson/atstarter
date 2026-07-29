// Package filetree 提供项目目录的只读浏览:列目录与读文件,
// 全部限定在给定 root 之内(防止 ../ 路径穿越)。
package filetree

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	trash "github.com/hymkor/trash-go"
)

// Entry 是目录下的一个直接子项。
type Entry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"` // 文件字节数;目录为 0
}

// SearchMatch 是项目文件名搜索的一条结果。
type SearchMatch struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
}

// SearchResults 是项目文件名搜索结果集。
type SearchResults struct {
	Matches   []SearchMatch `json:"matches"`
	Truncated bool          `json:"truncated"`
}

// resolve 把 relPath 安全解析到 root 之内的绝对路径。
// 越出 root 返回错误。
// 注意:guard 是纯词法的,不解析符号链接;指向 root 外的软链不会被拦截(对本地项目浏览器可接受)。
func resolve(root, relPath string) (string, error) {
	// Join then Clean: filepath.Join already calls Clean internally.
	full := filepath.Join(root, relPath)
	// Ensure full is strictly within root (or equals root when relPath is empty).
	rootClean := filepath.Clean(root)
	sep := string(filepath.Separator)
	if full != rootClean && !strings.HasPrefix(full, rootClean+sep) {
		return "", errors.New("path escapes root: " + relPath)
	}
	return full, nil
}

// ResolveWithin 把 relPath 安全解析到 root 之内的绝对路径(导出版,供 asset handler 用)。
// 越出 root 返回错误。
func ResolveWithin(root, relPath string) (string, error) {
	return resolve(root, relPath)
}

// ListDir 列出 root/relPath 这一层的直接子项。
// 目录在前,组内按名称升序。relPath 为空表示 root 本身。
func ListDir(root, relPath string) ([]Entry, error) {
	dir, err := resolve(root, relPath)
	if err != nil {
		return nil, err
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(items))
	for _, it := range items {
		e := Entry{Name: it.Name(), IsDir: it.IsDir()}
		if !it.IsDir() {
			if info, err := it.Info(); err == nil {
				e.Size = info.Size()
			}
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir // 目录在前
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

// maxWalkEntries 是 WalkPaths 返回路径数的上限。超过即停止并标记截断,
// 防止误选巨型目录(如含 node_modules 的仓库)后一次性返回海量路径卡死前端。
const maxWalkEntries = 50000

// WalkPaths 递归 walk root 下所有文件与目录,返回相对 root 的路径列表。
// 目录路径以 "/" 结尾(@pierre/trees 用尾斜杠区分目录);文件不带尾斜杠。
// 路径统一用 "/" 分隔(即使在 Windows 上),直接符合 trees 期望的输入形态。
// 仍限定在 root 内(WalkDir 天然不越界,不跟随软链出 root)。
//
// 本版不做任何忽略目录剪枝:全量返回。若后续需要剪掉 .git / node_modules 等,
// 在下方 walk 回调里对目录名做判断 fs.SkipDir 即可。
//
// 返回的 truncated 为 true 表示命中 maxWalkEntries 上限,列表被截断。
func WalkPaths(root string) (paths []string, truncated bool, err error) {
	return walkPaths(root, maxWalkEntries)
}

// walkPaths 是 WalkPaths 的内部实现,limit 可注入以便测试截断逻辑。
func walkPaths(root string, limit int) (paths []string, truncated bool, err error) {
	rootClean := filepath.Clean(root)
	walkErr := filepath.WalkDir(rootClean, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == rootClean {
			return nil // 跳过 root 自身
		}
		if len(paths) >= limit {
			truncated = true
			return filepath.SkipAll
		}
		rel, err := filepath.Rel(rootClean, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			rel += "/"
		}
		paths = append(paths, rel)
		return nil
	})
	if walkErr != nil {
		return nil, false, walkErr
	}
	return paths, truncated, nil
}

const defaultSearchLimit = 100
const maxSearchVisitedEntries = 50000

var searchSkipDirs = map[string]bool{
	".git":            true,
	"node_modules":    true,
	"vendor":          true,
	"dist":            true,
	"build":           true,
	"target":          true,
	".next":           true,
	".nuxt":           true,
	".vite":           true,
	".scannerwork":    true,
	".review-tmp":     true,
	".review-reports": true,
}

// SearchPaths searches project-relative file and directory names under root.
// It matches case-insensitively against both basename and slash-separated
// relative path. Directory results carry a trailing "/" to match WalkPaths.
func SearchPaths(root, query string, limit int) (matches []SearchMatch, truncated bool, err error) {
	return searchPaths(root, query, limit, maxSearchVisitedEntries)
}

func searchPaths(root, query string, limit int, visitLimit int) (matches []SearchMatch, truncated bool, err error) {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil, false, nil
	}
	if limit <= 0 || limit > defaultSearchLimit {
		limit = defaultSearchLimit
	}
	if visitLimit <= 0 {
		visitLimit = maxSearchVisitedEntries
	}

	rootClean := filepath.Clean(root)
	visited := 0
	walkErr := filepath.WalkDir(rootClean, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == rootClean {
			return nil
		}
		name := d.Name()
		if d.IsDir() && searchSkipDirs[strings.ToLower(name)] {
			return filepath.SkipDir
		}
		if visited >= visitLimit {
			truncated = true
			return filepath.SkipAll
		}
		visited++
		rel, err := filepath.Rel(rootClean, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		isDir := d.IsDir()
		if isDir {
			rel += "/"
		}
		lowerName := strings.ToLower(name)
		lowerPath := strings.ToLower(rel)
		if !strings.Contains(lowerName, query) && !strings.Contains(lowerPath, query) {
			return nil
		}
		if len(matches) >= limit {
			truncated = true
			return filepath.SkipAll
		}
		matches = append(matches, SearchMatch{Path: rel, Name: name, IsDir: isDir})
		return nil
	})
	if walkErr != nil {
		return nil, false, walkErr
	}
	return matches, truncated, nil
}

// Search returns file-name search results in a Wails-friendly envelope.
func Search(root, query string, limit int) (SearchResults, error) {
	matches, truncated, err := SearchPaths(root, query, limit)
	if err != nil {
		return SearchResults{}, err
	}
	return SearchResults{Matches: matches, Truncated: truncated}, nil
}

// maxReadBytes 是预览读取上限。超过则截断。
// 4MB:前端用 VirtualizedFile 按可视区渲染,该量级实测首帧约 1.9s 可接受;
// 再大(8MB+)pierre 首次行索引化会重新变卡。
const maxReadBytes = 4 << 20 // 4MB

// FileContent 是一次文件预览的结果。
type FileContent struct {
	Content   string `json:"content"`   // 文本内容;Binary 时为空
	Size      int64  `json:"size"`      // 原始文件字节数
	Truncated bool   `json:"truncated"` // 超过上限只读了前 maxReadBytes
	Binary    bool   `json:"binary"`    // 检测到二进制,不返回内容
}

// ReadFile 读取 root/relPath 文件,做大小截断与二进制检测。
func ReadFile(root, relPath string) (FileContent, error) {
	full, err := resolve(root, relPath)
	if err != nil {
		return FileContent{}, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return FileContent{}, err
	}
	if info.IsDir() {
		return FileContent{}, errors.New("is a directory: " + relPath)
	}
	f, err := os.Open(full)
	if err != nil {
		return FileContent{}, err
	}
	defer f.Close()

	buf := make([]byte, maxReadBytes)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return FileContent{}, err
	}
	data := buf[:n]

	fc := FileContent{Size: info.Size(), Truncated: info.Size() > maxReadBytes}
	if bytes.IndexByte(data, 0x00) >= 0 {
		fc.Binary = true
		return fc, nil
	}
	fc.Content = string(data)
	return fc, nil
}

// WriteFile 把 content 写回 root/relPath。只允许写「可完整读取的文本文件」:
// 目标必须已存在、是文件、非二进制、且原大小 ≤ maxReadBytes(1MB)。
// 校验基于磁盘上的原文件(不信任调用方),不支持新建文件/目录。
// 覆盖写并保留原文件权限位。
func WriteFile(root, relPath, content string) error {
	full, err := resolve(root, relPath)
	if err != nil {
		return err
	}
	info, err := os.Stat(full)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("is a directory: " + relPath)
	}
	if info.Size() > maxReadBytes {
		return errors.New("file too large to edit: " + relPath)
	}
	// 读原文件头部按 ReadFile 相同规则判定二进制,二进制拒绝写。
	f, err := os.Open(full)
	if err != nil {
		return err
	}
	buf := make([]byte, maxReadBytes)
	n, err := io.ReadFull(f, buf)
	f.Close()
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return err
	}
	if bytes.IndexByte(buf[:n], 0x00) >= 0 {
		return errors.New("cannot edit binary file: " + relPath)
	}
	return os.WriteFile(full, []byte(content), info.Mode().Perm())
}

// FileMetaInfo 是文件/目录的元信息,供懒加载树与预览判断使用。
type FileMetaInfo struct {
	Size     int64 `json:"size"`     // 字节数;目录为 0
	ModTime  int64 `json:"modTime"`  // 修改时间(Unix 毫秒)
	IsDir    bool  `json:"isDir"`    // 是否目录
	IsBinary bool  `json:"isBinary"` // 文件头部含 NUL 判为二进制;目录恒 false
}

// binaryProbeBytes 是二进制探测读取的头部字节数。
const binaryProbeBytes = 4096

// FileMeta 返回 root/relPath 的元信息。二进制探测只读头部 binaryProbeBytes。
func FileMeta(root, relPath string) (FileMetaInfo, error) {
	full, err := resolve(root, relPath)
	if err != nil {
		return FileMetaInfo{}, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return FileMetaInfo{}, err
	}
	m := FileMetaInfo{
		Size:    info.Size(),
		ModTime: info.ModTime().UnixMilli(),
		IsDir:   info.IsDir(),
	}
	if !info.IsDir() {
		f, err := os.Open(full)
		if err == nil {
			probe := make([]byte, binaryProbeBytes)
			n, _ := f.Read(probe)
			f.Close()
			m.IsBinary = bytes.IndexByte(probe[:n], 0x00) >= 0
		}
	}
	return m, nil
}

// CreateFile 在 root/relPath 创建空文件。目标已存在则报错;父目录须已存在。
func CreateFile(root, relPath string) error {
	full, err := resolve(root, relPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(full); err == nil {
		return errors.New("already exists: " + relPath)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(full, nil, 0o644)
}

// Mkdir 在 root/relPath 创建单个目录。目标已存在则报错;父目录须已存在。
func Mkdir(root, relPath string) error {
	full, err := resolve(root, relPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(full); err == nil {
		return errors.New("already exists: " + relPath)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Mkdir(full, 0o755)
}

// Rename 把 root/from 重命名/移动到 root/to。两端都过 resolve(对称防穿越)。
// from 不存在或 to 已存在则报错。
func Rename(root, from, to string) error {
	src, err := resolve(root, from)
	if err != nil {
		return err
	}
	dst, err := resolve(root, to)
	if err != nil {
		return err
	}
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return errors.New("not found: " + from)
	} else if err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return errors.New("already exists: " + to)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Rename(src, dst)
}

// Remove 删除 root/relPath。recursive=false 用 os.Remove(非空目录会失败),
// recursive=true 用 os.RemoveAll。目标不存在则报错。
func Remove(root, relPath string, recursive bool) error {
	full, err := resolve(root, relPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(full); os.IsNotExist(err) {
		return errors.New("not found: " + relPath)
	} else if err != nil {
		return err
	}
	if recursive {
		return os.RemoveAll(full)
	}
	return os.Remove(full)
}

// ErrTrashUnavailable 表示当前平台没有可用的废纸篓机制,调用方可降级为硬删。
var ErrTrashUnavailable = errors.New("trash unavailable")

// Trash 把 root/relPath 移入操作系统废纸篓。目标不存在则报错;
// 平台无废纸篓机制时返回 ErrTrashUnavailable(调用方降级确认硬删)。
func Trash(root, relPath string) error {
	full, err := resolve(root, relPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(full); os.IsNotExist(err) {
		return errors.New("not found: " + relPath)
	} else if err != nil {
		return err
	}
	if err := trash.Throw(full); err != nil {
		return fmt.Errorf("%w: %v", ErrTrashUnavailable, err)
	}
	return nil
}
