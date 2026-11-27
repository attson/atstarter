package codexadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	"rootyard/internal/projectenv"
)

func (a *Adapter) Inspect(ctx context.Context, target projectenv.Target) (projectenv.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return projectenv.Snapshot{}, err
	}
	expected := filepath.Join(target.WorktreePath, ".codex", "config.toml")
	if !normalizedPathEqual(expected, target.ConfigPath) {
		return projectenv.Snapshot{}, unsafePathError(target.ConfigPath, "configuration path is outside the worktree target")
	}
	dir := filepath.Dir(target.ConfigPath)
	if info, err := os.Lstat(dir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return projectenv.Snapshot{}, unsafePathError(dir, "Codex configuration directory must be a real directory")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return projectenv.Snapshot{}, unsafePathError(dir, err.Error())
	}

	info, err := os.Lstat(target.ConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		return emptySnapshot(target), nil
	}
	if err != nil {
		return projectenv.Snapshot{}, unsafePathError(target.ConfigPath, err.Error())
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return projectenv.Snapshot{}, unsafePathError(target.ConfigPath, "Codex configuration must be a regular file")
	}
	data, err := os.ReadFile(target.ConfigPath)
	if err != nil {
		return projectenv.Snapshot{}, unsafePathError(target.ConfigPath, err.Error())
	}
	parsed, err := parseDocument(data)
	if err != nil {
		return projectenv.Snapshot{}, err
	}
	values := make(map[string][]projectenv.OriginValue)
	for _, key := range projectenv.SupportedCodexKeys {
		assignment, ok := parsed.Assignments[key]
		if !ok {
			continue
		}
		values[key] = []projectenv.OriginValue{{Value: assignment.Value, Origin: target.ConfigPath, MainConfig: true}}
	}
	return projectenv.Snapshot{
		Target: target, Values: values, Fingerprint: configFingerprint(true, data), Issues: []projectenv.Issue{},
		Exists: true, NativeContent: append([]byte(nil), data...),
	}, nil
}

func emptySnapshot(target projectenv.Target) projectenv.Snapshot {
	return projectenv.Snapshot{
		Target: target, Values: map[string][]projectenv.OriginValue{}, Fingerprint: configFingerprint(false, nil),
		Issues: []projectenv.Issue{}, Exists: false,
	}
}

func configFingerprint(exists bool, data []byte) string {
	prefix := []byte("absent\x00")
	if exists {
		prefix = []byte("present\x00")
	}
	hash := sha256.New()
	_, _ = hash.Write(prefix)
	_, _ = hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

func unsafePathError(path, detail string) error {
	return &projectenv.Error{Code: projectenv.ErrUnsafeCodexConfigPath, Message: "unsafe Codex project configuration path", Detail: detail + ": " + path}
}
