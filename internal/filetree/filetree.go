// Package filetree 提供项目目录的只读浏览:列目录与读文件,
// 全部限定在给定 root 之内(防止 ../ 路径穿越)。
package filetree

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry 是目录下的一个直接子项。
type Entry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"` // 文件字节数;目录为 0
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

// maxReadBytes 是预览读取上限。超过则截断。
const maxReadBytes = 1 << 20 // 1MB

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
