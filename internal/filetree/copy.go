package filetree

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// maxCopyEntries 是一次递归拷贝允许处理的条目上限。
// 与 maxWalkEntries 取同一量级:误选 node_modules 这类目录时宁可报错,
// 也不要拷到界面假死。
const maxCopyEntries = maxWalkEntries

// maxUniqueAttempts 是 UniqueName 尝试后缀的次数上限,防止病态目录下死循环。
const maxUniqueAttempts = 1000

// ErrCopyIntoSelf 表示目标位于源目录之内(会无限递归)。
var ErrCopyIntoSelf = errors.New("destination is inside source")

// ErrCopyTooManyEntries 表示递归拷贝超过 maxCopyEntries 条目上限。
var ErrCopyTooManyEntries = errors.New("copy aborted: too many entries")

// splitRelDir 把 relPath 拆成父目录与基名,兼容 "/" 与平台分隔符。
func splitRelDir(relPath string) (dir, base string) {
	i := strings.LastIndexAny(relPath, `/\`)
	if i < 0 {
		return "", relPath
	}
	return relPath[:i], relPath[i+1:]
}

// joinRel 拼回 relPath,父目录为空时不加前导分隔符(root 下的直接子项)。
func joinRel(dir, base string) string {
	if dir == "" {
		return base
	}
	return dir + "/" + base
}

// splitStemExt 把基名拆成主干与扩展名。前导点(.gitignore)不算扩展名;
// isDir 为 true 时整个基名都算主干(目录名里的点不是扩展名)。
func splitStemExt(base string, isDir bool) (stem, ext string) {
	if isDir {
		return base, ""
	}
	i := strings.LastIndex(base, ".")
	if i <= 0 {
		return base, ""
	}
	return base[:i], base[i:]
}

// UniqueName 在 dstRoot 下为 dstRel 找一个不冲突的 relPath。
// dstRel 本身不存在则原样返回;否则依次尝试 "name copy.ext"、"name copy 2.ext" …
// 该函数是「永不覆盖」策略的唯一实现点,Copy 与 Move 都经由它。
func UniqueName(dstRoot, dstRel string) (string, error) {
	full, err := resolve(dstRoot, dstRel)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(full)
	if os.IsNotExist(err) {
		return dstRel, nil
	}
	if err != nil {
		return "", err
	}
	dir, base := splitRelDir(dstRel)
	stem, ext := splitStemExt(base, info.IsDir())
	for i := 1; i <= maxUniqueAttempts; i++ {
		candidate := stem + " copy" + ext
		if i > 1 {
			candidate = stem + " copy " + strconv.Itoa(i) + ext
		}
		next := joinRel(dir, candidate)
		nextFull, err := resolve(dstRoot, next)
		if err != nil {
			return "", err
		}
		if _, err := os.Lstat(nextFull); os.IsNotExist(err) {
			return next, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", errors.New("cannot find a free name for: " + dstRel)
}

// ensureNotNested 拒绝把目录拷/移到它自己内部(dst 在 src 之下会无限递归)。
func ensureNotNested(src, dst string, srcIsDir bool) error {
	if !srcIsDir {
		return nil
	}
	if dst == src || strings.HasPrefix(dst, src+string(filepath.Separator)) {
		return ErrCopyIntoSelf
	}
	return nil
}

// Copy 把 srcRoot/srcRel 递归拷贝到 dstRoot/dstRel,返回实际写入的目标 relPath。
// 源与目标各自过自己 root 的 resolve,因此可以跨项目(两个不同 root)。
//
// 契约:
//   - 永不覆盖。目标已存在时经 UniqueName 自动改名,调用方按返回值定位新文件。
//   - 目标落在源目录内时返回 ErrCopyIntoSelf。
//   - 条目数超过 maxCopyEntries 时返回 ErrCopyTooManyEntries。
//   - 符号链接原样复制链接本身,不跟随(与 resolve 的词法 guard 限制一致,也避免环)。
//   - 普通文件流式拷贝,不进内存,因此不受写字节接口的 16MB 上限约束。
func Copy(srcRoot, srcRel, dstRoot, dstRel string) (string, error) {
	src, err := resolve(srcRoot, srcRel)
	if err != nil {
		return "", err
	}
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return "", err
	}
	finalRel, err := UniqueName(dstRoot, dstRel)
	if err != nil {
		return "", err
	}
	dst, err := resolve(dstRoot, finalRel)
	if err != nil {
		return "", err
	}
	if err := ensureNotNested(src, dst, srcInfo.IsDir()); err != nil {
		return "", err
	}
	count := 0
	if err := copyPath(src, dst, srcInfo, &count); err != nil {
		return "", err
	}
	return finalRel, nil
}

// Move 把 srcRoot/srcRel 移动到 dstRoot/dstRel,返回实际写入的目标 relPath。
// 同 root 内走 os.Rename;跨 root(跨项目/跨设备)Rename 失败时降级为拷贝后删除。
// 与 Copy 一样永不覆盖:目标已存在会自动改名。
func Move(srcRoot, srcRel, dstRoot, dstRel string) (string, error) {
	src, err := resolve(srcRoot, srcRel)
	if err != nil {
		return "", err
	}
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return "", err
	}
	rawDst, err := resolve(dstRoot, dstRel)
	if err != nil {
		return "", err
	}
	if rawDst == src {
		return "", errors.New("source and destination are the same: " + srcRel)
	}
	finalRel, err := UniqueName(dstRoot, dstRel)
	if err != nil {
		return "", err
	}
	dst, err := resolve(dstRoot, finalRel)
	if err != nil {
		return "", err
	}
	if err := ensureNotNested(src, dst, srcInfo.IsDir()); err != nil {
		return "", err
	}
	if err := os.Rename(src, dst); err == nil {
		return finalRel, nil
	}
	// 跨设备/跨文件系统的 Rename 会失败(EXDEV),降级为拷贝 + 删除。
	count := 0
	if err := copyPath(src, dst, srcInfo, &count); err != nil {
		return "", err
	}
	if err := os.RemoveAll(src); err != nil {
		return "", err
	}
	return finalRel, nil
}

// copyPath 递归拷贝单个条目。info 必须来自 Lstat(不跟随符号链接)。
func copyPath(src, dst string, info os.FileInfo, count *int) error {
	*count++
	if *count > maxCopyEntries {
		return ErrCopyTooManyEntries
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case info.IsDir():
		if err := os.Mkdir(dst, info.Mode().Perm()); err != nil {
			return err
		}
		items, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, it := range items {
			childInfo, err := it.Info() // DirEntry.Info 是 lstat 语义
			if err != nil {
				return err
			}
			if err := copyPath(filepath.Join(src, it.Name()), filepath.Join(dst, it.Name()), childInfo, count); err != nil {
				return err
			}
		}
		return nil
	case info.Mode().IsRegular():
		return copyFile(src, dst, info.Mode().Perm())
	default:
		return errors.New("unsupported file type: " + src)
	}
}

// copyFile 流式拷贝普通文件并保留权限位。目标用 O_EXCL 打开,已存在即失败
// (UniqueName 已保证不冲突,这里是最后一道防覆盖闸)。
func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}
