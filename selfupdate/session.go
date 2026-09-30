package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type installSession struct {
	target   Target
	policy   TargetPolicy
	root     *os.Root
	lock     lockHandle
	staging  map[string]struct{}
	closed   bool
	mu       sync.Mutex
	closeErr error
	// dirInfo identifies the locked target directory (0003-MADR B10).
	dirInfo os.FileInfo
}

func (s *installSession) Target() Target {
	return s.target
}

func (s *installSession) CreateStaging(ctx context.Context) (*os.File, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, "", fmt.Errorf("selfupdate: session is closed")
	}
	f, err := os.CreateTemp(s.target.Dir, "."+s.target.Base+".selfupdate-")
	if err != nil {
		return nil, "", fmt.Errorf("selfupdate: create staging: %w", err)
	}
	name := f.Name()
	base := filepath.Base(name)
	if _, err := s.root.Lstat(base); err != nil {
		err = fmt.Errorf("selfupdate: staging escaped target directory: %w", err)
		err = joinClose(err, f)
		return nil, "", joinRemove(err, name)
	}
	if s.staging == nil {
		s.staging = make(map[string]struct{})
	}
	s.staging[name] = struct{}{}
	return f, name, nil
}

func (s *installSession) owns(path string) bool {
	_, ok := s.staging[path]
	return ok
}

func (s *installSession) Install(ctx context.Context, req InstallRequest) (InstallResult, error) {
	if err := ctx.Err(); err != nil {
		return InstallResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	applied, err := s.replaceLocked(ctx, req.Artifact.Path)
	if err != nil {
		// Backup is non-empty only when the new binary is live and the
		// restore failed (0003-MADR B1).
		return InstallResult{Target: s.target.Path, Backup: applied.backup}, err
	}
	if err := s.checkDir(); err != nil {
		return InstallResult{Target: s.target.Path, Backup: applied.backup, Applied: true}, err
	}
	pending, err := commitReplacement(s.target, applied)
	if err != nil {
		return InstallResult{
			Target:        s.target.Path,
			Backup:        applied.backup,
			Applied:       true,
			PendingBackup: pending,
		}, err
	}
	return InstallResult{
		Target:        s.target.Path,
		Backup:        applied.backup,
		Applied:       true,
		PendingBackup: pending,
	}, nil
}

func (s *installSession) apply(ctx context.Context, req InstallRequest) (applyResult, error) {
	if err := ctx.Err(); err != nil {
		return applyResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.replaceLocked(ctx, req.Artifact.Path)
}

// replaceLocked replaces the target with an owned staging file. The caller
// holds s.mu. Staging is deregistered only once the rename has consumed it,
// so a failure before that leaves it for Close to remove (0003-MADR B4).
func (s *installSession) replaceLocked(ctx context.Context, path string) (applyResult, error) {
	if s.closed {
		return applyResult{}, fmt.Errorf("selfupdate: session is closed")
	}
	if !s.owns(path) {
		return applyResult{}, fmt.Errorf("selfupdate: artifact is not owned by this session")
	}
	// The directory first: once it has been swapped, every path below it
	// names something other than what was locked.
	if err := s.checkDir(); err != nil {
		return applyResult{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return applyResult{}, fmt.Errorf("selfupdate: stat staging: %w", err)
	}
	if !info.Mode().IsRegular() {
		return applyResult{}, fmt.Errorf("selfupdate: staging is not a regular file")
	}
	applied, err := replaceTarget(ctx, s.target, path)
	if applied.renamed {
		delete(s.staging, path)
	}
	return applied, err
}

// checkDir requires the target directory to be the one the session locked:
// a directory swapped in after Begin would put the replacement outside the
// lock (0003-MADR B10).
func (s *installSession) checkDir() error {
	if s.dirInfo == nil {
		return nil
	}
	cur, err := os.Stat(s.target.Dir)
	if err != nil || !os.SameFile(s.dirInfo, cur) {
		return fmt.Errorf("selfupdate: target directory changed during the update: %w", ErrConcurrentUpdate)
	}
	return nil
}

func (s *installSession) commit(applied applyResult) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkDir(); err != nil {
		return "", err
	}
	return commitReplacement(s.target, applied)
}

func (s *installSession) rollback(ctx context.Context, applied applyResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return rollbackReplacement(ctx, s.target, applied)
}

func (s *installSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	var errs []error
	for name := range s.staging {
		if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	s.staging = nil
	if s.root != nil {
		if err := s.root.Close(); err != nil {
			errs = append(errs, err)
		}
		s.root = nil
	}
	if err := s.lock.release(); err != nil {
		errs = append(errs, err)
	}
	s.closeErr = errors.Join(errs...)
	return s.closeErr
}

func beginSession(ctx context.Context, policy TargetPolicy, original Target, timeout time.Duration) (*installSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(original.Dir)
	if err != nil {
		return nil, fmt.Errorf("selfupdate: open target directory: %w", err)
	}
	lock, err := acquireLock(ctx, root, original.Base, timeout)
	if err != nil {
		err = joinClose(err, root)
		if errors.Is(err, ErrConcurrentUpdate) {
			return nil, err
		}
		return nil, fmt.Errorf("selfupdate: acquire lock: %w", err)
	}
	if err := processCleanupReceipt(original, root); err != nil {
		return nil, errors.Join(err, lock.release(), root.Close())
	}
	if err := revalidateTarget(original, policy); err != nil {
		return nil, errors.Join(err, lock.release(), root.Close())
	}
	// The directory opened as root must be the one at the path, and that
	// path identity is what later steps re-check.
	rootInfo, err := root.Stat(".")
	if err != nil {
		return nil, errors.Join(fmt.Errorf("selfupdate: stat target directory: %w", err), lock.release(), root.Close())
	}
	dirInfo, err := os.Stat(original.Dir)
	if err != nil || !os.SameFile(rootInfo, dirInfo) {
		return nil, errors.Join(fmt.Errorf("selfupdate: target directory changed while locking: %w", ErrConcurrentUpdate), lock.release(), root.Close())
	}
	return &installSession{
		target:  original,
		policy:  policy,
		root:    root,
		lock:    lock,
		staging: make(map[string]struct{}),
		dirInfo: dirInfo,
	}, nil
}
