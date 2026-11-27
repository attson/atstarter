package projectenv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const fileLockRetryInterval = 10 * time.Millisecond

func withFileLock(ctx context.Context, path string, fn func() error) error {
	unlock, err := AcquireFileLock(ctx, path)
	if err != nil {
		return err
	}
	defer unlock()
	return fn()
}

func AcquireFileLock(ctx context.Context, path string) (func() error, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, &Error{Code: ErrLockTimeout, Message: "timed out waiting for environment file lock", Detail: path, Cause: err}
		}
		unlock, acquired, err := tryFileLock(path)
		if err != nil {
			return nil, &Error{Code: ErrInvalidEnvironment, Message: "acquire environment file lock", Detail: err.Error(), Cause: err}
		}
		if acquired {
			return unlock, nil
		}

		timer := time.NewTimer(fileLockRetryInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, &Error{Code: ErrLockTimeout, Message: "timed out waiting for environment file lock", Detail: path, Cause: ctx.Err()}
		case <-timer.C:
		}
	}
}

func AdapterFileLockPath(adapter, targetID string) string {
	sum := sha256.Sum256([]byte(adapter + "\x00" + targetID))
	name := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, adapter)
	return filepath.Join(os.TempDir(), "rootyard-environment-locks", name+"-"+hex.EncodeToString(sum[:])+".lock")
}
