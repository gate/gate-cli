package oauth2

import (
	"context"
	"path/filepath"

	"github.com/gate/gate-cli/internal/config"
)

// ConfigLockPath returns the path of the flock file that serializes all
// writers of ~/.gate-cli/config.yaml. The lock file is named after the
// protected resource (not the legacy tokens.yaml) so "what does this file
// lock?" is obvious to operators inspecting ~/.gate-cli.
func ConfigLockPath() (string, error) {
	dir, err := tokenDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml.lock"), nil
}

// WithConfigLock acquires an exclusive flock on ConfigLockPath and runs fn
// while the lock is held. Every read-modify-write path that touches
// ~/.gate-cli/config.yaml — SaveToken, RemoveToken, refreshInternal,
// saveCustomRedirectURI — must go through this helper so concurrent gate-cli
// processes cannot clobber each other's rename() at the end of atomicWrite.
//
// Not reentrant: callers that need to save while already holding the lock
// must use the *Locked internal variants (e.g. saveTokenLocked), not a
// nested WithConfigLock call.
func WithConfigLock(ctx context.Context, fn func() error) error {
	path, err := ConfigYAMLPath()
	if err != nil {
		return err
	}
	return config.WithFileLock(ctx, path, fn)
}
