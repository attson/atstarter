package filetree

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// maxBytesHard 是二进制读写的硬上限(防超大文件把内存打爆)。
const maxBytesHard = 16 << 20 // 16MB

// ErrStaleModTime 表示写入时磁盘 ModTime 与调用方期望不符(有人在编辑期间改了文件)。
// 错误消息形如 "stale_modtime: current=<毫秒>",前端据此提示冲突。
var ErrStaleModTime = errors.New("stale_modtime")

// FileBytes 是 ReadFileBytes 的结果:原始字节 + 元信息。
type FileBytes struct {
	Data        []byte `json:"data"`                  // 原始字节(可能被 maxBytes 截断)
	ModTime     int64  `json:"modTime"`               // 修改时间(Unix 毫秒),写回时做冲突检测
	IsBinary    bool   `json:"isBinary"`              // 头部含 NUL
	TruncatedAt int64  `json:"truncatedAt,omitempty"` // >0 表示被截断,值为完整字节数
}

// ReadFileBytes 读取 root/relPath 的原始字节,最多 maxBytes(0 或超过硬上限时用硬上限)。
// 返回 ModTime 供写回时冲突检测。二进制/文本都返回字节,由前端处理。
func ReadFileBytes(root, relPath string, maxBytes int64) (FileBytes, error) {
	if maxBytes <= 0 || maxBytes > maxBytesHard {
		maxBytes = maxBytesHard
	}
	full, err := resolve(root, relPath)
	if err != nil {
		return FileBytes{}, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return FileBytes{}, err
	}
	if info.IsDir() {
		return FileBytes{}, errors.New("is a directory: " + relPath)
	}
	f, err := os.Open(full)
	if err != nil {
		return FileBytes{}, err
	}
	defer f.Close()

	size := info.Size()
	readLen := size
	var truncatedAt int64
	if size > maxBytes {
		readLen = maxBytes
		truncatedAt = size
	}
	data := make([]byte, readLen)
	n, err := io.ReadFull(f, data)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return FileBytes{}, err
	}
	data = data[:n]
	return FileBytes{
		Data:        data,
		ModTime:     info.ModTime().UnixMilli(),
		IsBinary:    bytes.IndexByte(data, 0x00) >= 0,
		TruncatedAt: truncatedAt,
	}, nil
}

// WriteFileBytes 原子写(临时文件 + rename)data 到 root/relPath。
// expectedModTime=0 关闭冲突检测;否则目标 ModTime 须等于 expectedModTime,
// 不符返回 ErrStaleModTime。createIfMissing=true 时允许新建。
// 返回新 ModTime。
func WriteFileBytes(root, relPath string, data []byte, expectedModTime int64, createIfMissing bool) (int64, error) {
	if int64(len(data)) > maxBytesHard {
		return 0, fmt.Errorf("write denied: exceeds %d bytes", maxBytesHard)
	}
	full, err := resolve(root, relPath)
	if err != nil {
		return 0, err
	}
	info, statErr := os.Stat(full)
	switch {
	case statErr == nil:
		if info.IsDir() {
			return 0, errors.New("is a directory: " + relPath)
		}
		if expectedModTime != 0 && info.ModTime().UnixMilli() != expectedModTime {
			return 0, fmt.Errorf("%w: current=%d", ErrStaleModTime, info.ModTime().UnixMilli())
		}
	case errors.Is(statErr, os.ErrNotExist):
		if !createIfMissing {
			return 0, errors.New("not found: " + relPath)
		}
	default:
		return 0, statErr
	}

	dir := filepath.Dir(full)
	tmp, err := os.CreateTemp(dir, ".atstarter-tmp-*")
	if err != nil {
		return 0, err
	}
	tmpName := tmp.Name()
	commit := false
	defer func() {
		tmp.Close()
		if !commit {
			os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return 0, err
	}
	if err := tmp.Sync(); err != nil {
		return 0, err
	}
	if err := tmp.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(tmpName, full); err != nil {
		return 0, err
	}
	commit = true

	newInfo, err := os.Stat(full)
	if err != nil {
		return 0, err
	}
	return newInfo.ModTime().UnixMilli(), nil
}
