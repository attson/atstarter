package projectenv

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

type Store struct {
	paths Paths
}

func NewStore(configPath string) *Store {
	return &Store{paths: PathsForConfig(configPath)}
}

func (s *Store) Load(ctx context.Context) (Document, error) {
	if err := ctx.Err(); err != nil {
		return Document{}, err
	}
	return s.loadUnlocked()
}

func (s *Store) Update(ctx context.Context, expectedRevision string, mutate func(*Config) error) (Document, error) {
	var updated Document
	err := withFileLock(ctx, s.paths.ConfigLock, func() error {
		current, err := s.loadUnlocked()
		if err != nil {
			return err
		}
		if expectedRevision != "" && expectedRevision != current.Revision {
			return newError(ErrEnvironmentStale, "project environment configuration changed", "reload and retry")
		}
		next := current.Config
		if err := mutate(&next); err != nil {
			return err
		}
		normalizeConfigCollections(&next)
		if err := ValidateConfig(next); err != nil {
			return err
		}
		data, err := marshalConfig(next)
		if err != nil {
			return err
		}
		if err := atomicWriteFile(s.paths.Config, data, 0o600); err != nil {
			return &Error{Code: ErrInvalidEnvironment, Message: "write project environment configuration", Detail: err.Error(), Cause: err}
		}
		updated = Document{Config: next, Revision: revisionFor(data)}
		return nil
	})
	return updated, err
}

func (s *Store) loadUnlocked() (Document, error) {
	data, err := os.ReadFile(s.paths.Config)
	if errors.Is(err, os.ErrNotExist) {
		cfg := emptyConfig()
		canonical, marshalErr := marshalConfig(cfg)
		if marshalErr != nil {
			return Document{}, marshalErr
		}
		return Document{Config: cfg, Revision: revisionFor(canonical)}, nil
	}
	if err != nil {
		return Document{}, &Error{Code: ErrInvalidEnvironment, Message: "read project environment configuration", Detail: err.Error(), Cause: err}
	}
	var cfg Config
	if err := decodeStrictJSON(data, &cfg); err != nil {
		return Document{}, &Error{Code: ErrInvalidEnvironment, Message: "parse project environment configuration", Detail: err.Error(), Cause: err}
	}
	normalizeConfigCollections(&cfg)
	if err := ValidateConfig(cfg); err != nil {
		return Document{}, err
	}
	return Document{Config: cfg, Revision: revisionFor(data)}, nil
}

func emptyConfig() Config {
	return Config{Version: 1, Presets: []Preset{}, Rules: []Rule{}}
}

func normalizeConfigCollections(cfg *Config) {
	if cfg.Presets == nil {
		cfg.Presets = []Preset{}
	}
	if cfg.Rules == nil {
		cfg.Rules = []Rule{}
	}
}

func marshalConfig(cfg Config) ([]byte, error) {
	normalizeConfigCollections(&cfg)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func revisionFor(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) (returnErr error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
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
	if err := temp.Chmod(mode); err != nil {
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
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	if err := syncDirectory(dir); err != nil {
		return err
	}
	return nil
}

func syncDirectory(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	file, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}
