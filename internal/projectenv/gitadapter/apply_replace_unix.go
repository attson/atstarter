//go:build !windows

package gitadapter

import "os"

func replaceConfigFile(source, destination string) error {
	return os.Rename(source, destination)
}
