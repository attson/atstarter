//go:build !windows

package codexadapter

import "os"

func replaceConfigFile(source, destination string) error {
	return os.Rename(source, destination)
}
