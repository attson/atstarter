package docker

import (
	"os"
	"os/exec"
)

var dockerFallbackPaths = []string{
	"/usr/local/bin/docker",
	"/opt/homebrew/bin/docker",
	"/Applications/Docker.app/Contents/Resources/bin/docker",
}

func resolveDockerCommand() string {
	return resolveDockerCommandWith(exec.LookPath, func(path string) bool {
		st, err := os.Stat(path)
		return err == nil && !st.IsDir()
	})
}

func resolveDockerCommandWith(lookPath func(string) (string, error), exists func(string) bool) string {
	if path, err := lookPath("docker"); err == nil && path != "" {
		return path
	}
	for _, path := range dockerFallbackPaths {
		if exists(path) {
			return path
		}
	}
	return "docker"
}
