package codexadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	secure, err := openSecureConfigDir(target.WorktreePath)
	if err != nil {
		return projectenv.Snapshot{}, unsafePathError(filepath.Dir(target.ConfigPath), err.Error())
	}
	defer secure.Close()
	snapshot, _, err := inspectSecureConfig(ctx, target, secure)
	return snapshot, err
}

func inspectSecureConfig(ctx context.Context, target projectenv.Target, secure *secureConfigDir) (projectenv.Snapshot, os.FileMode, error) {
	if err := ctx.Err(); err != nil {
		return projectenv.Snapshot{}, 0, err
	}
	data, mode, exists, err := secure.readConfig()
	if err != nil {
		return projectenv.Snapshot{}, 0, unsafePathError(target.ConfigPath, err.Error())
	}
	if !exists {
		return emptySnapshot(target), 0o600, nil
	}
	parsed, err := parseDocument(data)
	if err != nil {
		return projectenv.Snapshot{}, 0, err
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
	}, mode, nil
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
