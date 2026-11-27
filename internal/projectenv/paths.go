package projectenv

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Paths struct {
	Config     string `json:"config"`
	State      string `json:"state"`
	ConfigLock string `json:"configLock"`
	StateLock  string `json:"stateLock"`
}

func PathsForConfig(configPath string) Paths {
	dir := filepath.Dir(configPath)
	config := filepath.Join(dir, "environments.json")
	state := filepath.Join(dir, "environment-state.json")
	return Paths{
		Config:     config,
		State:      state,
		ConfigLock: config + ".lock",
		StateLock:  state + ".lock",
	}
}

func CanonicalRulePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", newError(ErrInvalidEnvironment, "directory rule path is required", "")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", newError(ErrInvalidEnvironment, "resolve directory rule path", err.Error())
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", newError(ErrInvalidEnvironment, "resolve directory rule path", err.Error())
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", newError(ErrInvalidEnvironment, "inspect directory rule path", err.Error())
	}
	if !info.IsDir() {
		return "", newError(ErrInvalidEnvironment, "directory rule path is not a directory", resolved)
	}
	return normalizePath(resolved), nil
}

func PathContains(root, candidate string) bool {
	root = normalizePath(root)
	candidate = normalizePath(candidate)
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func normalizePath(path string) string {
	clean := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		clean = strings.ToLower(clean)
	}
	return clean
}

func canonicalPathWithoutIO(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return normalizePath(abs), nil
}

func canonicalResolutionPath(path string) (string, error) {
	canonical, err := canonicalPathWithoutIO(path)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(canonical); err == nil {
		return normalizePath(resolved), nil
	}
	existing := canonical
	missing := []string{}
	for {
		parent := filepath.Dir(existing)
		if parent == existing {
			return canonical, nil
		}
		missing = append(missing, filepath.Base(existing))
		existing = parent
		resolved, resolveErr := filepath.EvalSymlinks(existing)
		if resolveErr != nil {
			continue
		}
		for index := len(missing) - 1; index >= 0; index-- {
			resolved = filepath.Join(resolved, missing[index])
		}
		return normalizePath(resolved), nil
	}
}
