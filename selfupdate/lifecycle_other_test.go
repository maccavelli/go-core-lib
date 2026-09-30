//go:build !windows

package selfupdate

import (
	"os"
	"testing"
)

// plantPendingCleanup leaves a stale cleanup receipt, which Begin removes
// on this OS, and returns the paths CleanupPending must remove.
func plantPendingCleanup(t *testing.T, target Target) []string {
	t.Helper()
	receipt := cleanupReceiptPath(target)
	if err := os.WriteFile(receipt, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	return []string{receipt}
}
