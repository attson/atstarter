package projectenv

import (
	"context"
	"time"
)

const fileLockRetryInterval = 10 * time.Millisecond

func withFileLock(ctx context.Context, path string, fn func() error) error {
	for {
		if err := ctx.Err(); err != nil {
			return &Error{Code: ErrLockTimeout, Message: "timed out waiting for environment file lock", Detail: path, Cause: err}
		}
		unlock, acquired, err := tryFileLock(path)
		if err != nil {
			return &Error{Code: ErrInvalidEnvironment, Message: "acquire environment file lock", Detail: err.Error(), Cause: err}
		}
		if acquired {
			defer unlock()
			return fn()
		}

		timer := time.NewTimer(fileLockRetryInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return &Error{Code: ErrLockTimeout, Message: "timed out waiting for environment file lock", Detail: path, Cause: ctx.Err()}
		case <-timer.C:
		}
	}
}
