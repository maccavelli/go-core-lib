//go:build windows

package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

type applyResult struct {
	backup    string
	oldDigest string
	// renamed reports that the staging file was consumed by the replace, so
	// the session must no longer remove it (0003-MADR B4).
	renamed bool
}

func isUnsupportedDirSync(err error) bool {
	return errors.Is(err, windows.ERROR_ACCESS_DENIED)
}

func replacePathOS(ctx context.Context, oldpath, newpath string) error {
	return moveFileReplace(ctx, oldpath, newpath)
}

func replaceTarget(ctx context.Context, target Target, staging string) (applyResult, error) {
	info, err := os.Lstat(target.Path)
	if err != nil {
		return applyResult{}, err
	}
	if err := chmodStaging(staging, info); err != nil {
		return applyResult{}, fmt.Errorf("selfupdate: chmod staging: %w", err)
	}
	oldDigest, err := fileSHA256(target.Path)
	if err != nil {
		return applyResult{}, err
	}
	backup, err := randomSibling(target.Dir, "."+target.Base+".selfupdate-bak-")
	if err != nil {
		return applyResult{}, fmt.Errorf("selfupdate: allocate backup: %w", err)
	}
	if err := backupFile(target.Path, backup); err != nil {
		return applyResult{}, fmt.Errorf("selfupdate: backup target: %w", err)
	}
	if err := replacePath(ctx, staging, target.Path); err != nil {
		return applyResult{}, joinRemove(fmt.Errorf("selfupdate: replace target: %w", err), backup)
	}
	if err := syncDirFn(target.Dir); err != nil && !isUnsupportedSync(err) {
		syncErr := fmt.Errorf("selfupdate: sync directory: %w", err)
		if rerr := replacePath(ctx, backup, target.Path); rerr != nil {
			// The new binary is live and the backup is kept: report both
			// (0003-MADR B1).
			return applyResult{backup: backup, oldDigest: oldDigest, renamed: true},
				errors.Join(syncErr, fmt.Errorf("selfupdate: restore backup: %w", rerr))
		}
		return applyResult{renamed: true}, syncErr
	}
	return applyResult{backup: backup, oldDigest: oldDigest, renamed: true}, nil
}

// moveFileExFn is windows.MoveFileEx, replaceable in tests.
var moveFileExFn = windows.MoveFileEx

// moveFileReplace retries a busy replacement until DefaultLockTimeout or the
// caller's context ends, whichever comes first (0003-MADR B10, B11).
func moveFileReplace(ctx context.Context, from, to string) error {
	fromW, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	toW, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(DefaultLockTimeout)
	var last error
	for {
		last = moveFileExFn(fromW, toW, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
		if last == nil {
			return nil
		}
		// A running image transiently refuses replacement with access
		// denied as well as a sharing violation (0003-MADR B11).
		if !isBusyRunningImage(last) || time.Now().After(deadline) {
			return last
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(last, ctx.Err())
		case <-timer.C:
		}
	}
}

func isSharingViolation(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION)
}

func isBusyRunningImage(err error) bool {
	return isSharingViolation(err) || errors.Is(err, windows.ERROR_ACCESS_DENIED)
}

func commitReplacement(target Target, result applyResult) (pending string, err error) {
	if result.backup == "" {
		return "", nil
	}
	if err := osRemove(result.backup); err != nil {
		if isBusyRunningImage(err) {
			if werr := writeCleanupReceipt(target, result); werr != nil {
				return "", errors.Join(fmt.Errorf("selfupdate: remove backup: %w", err), werr)
			}
			return result.backup, nil
		}
		return "", fmt.Errorf("selfupdate: remove backup: %w", err)
	}
	return "", syncDirFn(target.Dir)
}

func rollbackReplacement(ctx context.Context, target Target, result applyResult) error {
	if result.backup == "" {
		return fmt.Errorf("selfupdate: no backup to restore")
	}
	if err := replacePath(ctx, result.backup, target.Path); err != nil {
		return fmt.Errorf("selfupdate: restore backup: %w", err)
	}
	return syncDirFn(target.Dir)
}
