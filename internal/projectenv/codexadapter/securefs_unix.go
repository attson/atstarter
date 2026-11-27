//go:build !windows

package codexadapter

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

type secureConfigDir struct {
	root *os.File
	dir  *os.File
}

func openSecureConfigDir(worktree string) (*secureConfigDir, error) {
	fd, err := unix.Open(worktree, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	secure := &secureConfigDir{root: os.NewFile(uintptr(fd), worktree)}
	if err := secure.openConfigDir(); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = secure.Close()
		return nil, err
	}
	return secure, nil
}

func (s *secureConfigDir) openConfigDir() error {
	fd, err := unix.Openat(int(s.root.Fd()), ".codex", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	s.dir = os.NewFile(uintptr(fd), ".codex")
	return nil
}

func (s *secureConfigDir) ensureConfigDir() (bool, error) {
	if s.dir != nil {
		return false, s.verifyIdentity()
	}
	created := false
	if err := unix.Mkdirat(int(s.root.Fd()), ".codex", 0o700); err == nil {
		created = true
	} else if !errors.Is(err, unix.EEXIST) {
		return false, err
	}
	if err := s.openConfigDir(); err != nil {
		return false, err
	}
	return created, nil
}

func (s *secureConfigDir) readConfig() ([]byte, os.FileMode, bool, error) {
	if s.dir == nil {
		return nil, 0o600, false, nil
	}
	fd, err := unix.Openat(int(s.dir.Fd()), "config.toml", unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil, 0o600, false, nil
	}
	if err != nil {
		return nil, 0, false, err
	}
	file := os.NewFile(uintptr(fd), "config.toml")
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, 0, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, 0, false, errors.New("Codex configuration must be a regular file")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, 0, false, err
	}
	return data, info.Mode().Perm(), true, nil
}

func (s *secureConfigDir) verifyIdentity() error {
	if s.dir == nil {
		return errors.New("Codex configuration directory is unavailable")
	}
	fd, err := unix.Openat(int(s.root.Fd()), ".codex", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	current := os.NewFile(uintptr(fd), ".codex-current")
	defer current.Close()
	heldInfo, err := s.dir.Stat()
	if err != nil {
		return err
	}
	currentInfo, err := current.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(heldInfo, currentInfo) {
		return errors.New("Codex configuration directory changed during apply")
	}
	return nil
}

func secureAtomicWrite(s *secureConfigDir, data []byte, mode os.FileMode) (returnErr error) {
	if err := s.verifyIdentity(); err != nil {
		return err
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	tempName := ".rootyard-codex-config-" + hex.EncodeToString(random[:])
	fd, err := unix.Openat(int(s.dir.Fd()), tempName, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, uint32(mode.Perm()))
	if err != nil {
		return err
	}
	temp := os.NewFile(uintptr(fd), tempName)
	defer func() {
		_ = temp.Close()
		if returnErr != nil {
			_ = unix.Unlinkat(int(s.dir.Fd()), tempName, 0)
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
	if err := unix.Renameat(int(s.dir.Fd()), tempName, int(s.dir.Fd()), "config.toml"); err != nil {
		return err
	}
	if err := s.dir.Sync(); err != nil {
		return err
	}
	return nil
}

func (s *secureConfigDir) Close() error {
	var first error
	if s.dir != nil {
		first = s.dir.Close()
	}
	if err := s.root.Close(); first == nil {
		first = err
	}
	return first
}
