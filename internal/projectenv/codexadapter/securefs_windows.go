//go:build windows

package codexadapter

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

type secureConfigDir struct {
	rootPath string
	dirPath  string
	dir      *os.File
	dirInfo  os.FileInfo
}

func openSecureConfigDir(worktree string) (*secureConfigDir, error) {
	rootInfo, err := os.Lstat(worktree)
	if err != nil {
		return nil, err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return nil, errors.New("Codex worktree must be a real directory")
	}
	secure := &secureConfigDir{rootPath: worktree, dirPath: filepath.Join(worktree, ".codex")}
	if err := secure.openConfigDir(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return secure, nil
}

func (s *secureConfigDir) openConfigDir() error {
	info, err := os.Lstat(s.dirPath)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("Codex configuration directory must be a real directory")
	}
	dir, err := os.Open(s.dirPath)
	if err != nil {
		return err
	}
	s.dir, s.dirInfo = dir, info
	return nil
}

func (s *secureConfigDir) ensureConfigDir() (bool, error) {
	if s.dir != nil {
		return false, s.verifyIdentity()
	}
	created := false
	if err := os.Mkdir(s.dirPath, 0o700); err == nil {
		created = true
	} else if !errors.Is(err, os.ErrExist) {
		return false, err
	}
	return created, s.openConfigDir()
}

func (s *secureConfigDir) readConfig() ([]byte, os.FileMode, bool, error) {
	if s.dir == nil {
		return nil, 0o600, false, nil
	}
	path := filepath.Join(s.dirPath, "config.toml")
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0o600, false, nil
	}
	if err != nil {
		return nil, 0, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, 0, false, errors.New("Codex configuration must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, false, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !os.SameFile(info, openedInfo) {
		return nil, 0, false, errors.New("Codex configuration changed while opening")
	}
	data, err := io.ReadAll(file)
	return data, info.Mode().Perm(), true, err
}

func (s *secureConfigDir) verifyIdentity() error {
	info, err := os.Lstat(s.dirPath)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !os.SameFile(s.dirInfo, info) {
		return errors.New("Codex configuration directory changed during apply")
	}
	return nil
}

func secureAtomicWrite(s *secureConfigDir, data []byte, mode os.FileMode) (returnErr error) {
	if err := s.verifyIdentity(); err != nil {
		return err
	}
	temp, err := os.CreateTemp(s.dirPath, ".rootyard-codex-config-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		if returnErr != nil {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(mode.Perm()); err != nil {
		return err
	}
	if _, err := temp.Write(data); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := s.verifyIdentity(); err != nil {
		return err
	}
	return replaceConfigFile(tempPath, filepath.Join(s.dirPath, "config.toml"))
}

func (s *secureConfigDir) Close() error {
	if s.dir == nil {
		return nil
	}
	return s.dir.Close()
}
