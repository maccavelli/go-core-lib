package selfupdate

import (
	"context"
	"fmt"
	"time"
)

// StandaloneInstaller owns target resolution, session construction, and
// binary replacement without service lifecycle.
type StandaloneInstaller struct {
	policy       TargetPolicy
	lockTimeout  time.Duration
	postInstall  Prober
	keepPrevious bool
}

// NewStandaloneInstaller returns a binary-only installer. LockTimeout zero
// selects DefaultLockTimeout. A negative duration is invalid.
func NewStandaloneInstaller(opts InstallOptions) (*StandaloneInstaller, error) {
	timeout := opts.LockTimeout
	if timeout == 0 {
		timeout = DefaultLockTimeout
	}
	if timeout < 0 {
		return nil, fmt.Errorf("selfupdate: lock timeout must not be negative")
	}
	s := &StandaloneInstaller{policy: opts.TargetPolicy, lockTimeout: timeout, keepPrevious: opts.KeepPrevious}
	if !isNil(opts.PostInstall) {
		s.postInstall = opts.PostInstall
	}
	return s, nil
}

// ResolveTarget implements Installer.
func (s *StandaloneInstaller) ResolveTarget(context.Context) (Target, error) {
	return resolveTarget(s.policy)
}

// Begin implements Installer.
func (s *StandaloneInstaller) Begin(ctx context.Context, target Target) (InstallSession, error) {
	sess, err := beginSession(ctx, s.policy, target, s.lockTimeout)
	if err != nil {
		return nil, err
	}
	sess.postInstall = s.postInstall
	sess.keepPrevious = s.keepPrevious
	return sess, nil
}

// CleanupPending finishes what an earlier update left behind: it takes
// the target's lock, which processes a pending cleanup receipt (on
// Windows, the backup the running image kept open; elsewhere, a stale
// receipt), and releases it. It installs nothing.
//
// When another update holds the lock, the error matches
// ErrConcurrentUpdate. That is benign: retry later.
func (s *StandaloneInstaller) CleanupPending(ctx context.Context) error {
	target, err := resolveTarget(s.policy)
	if err != nil {
		return err
	}
	sess, err := beginSession(ctx, s.policy, target, s.lockTimeout)
	if err != nil {
		return err
	}
	return sess.Close()
}
