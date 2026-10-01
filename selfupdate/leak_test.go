package selfupdate

import (
	"runtime"
	"testing"
	"time"
)

// checkNoLeak fails the test when, once it has cleaned up, more goroutines
// run than when checkNoLeak was called. It polls for up to 2 s, because a
// goroutine that is ending is still counted for a moment (0004-MADR H6).
// Call it first, so its cleanup runs last.
func checkNoLeak(t testing.TB) {
	t.Helper()
	base := runtime.NumGoroutine()
	t.Cleanup(func() {
		deadline := time.Now().Add(2 * time.Second)
		for runtime.NumGoroutine() > base {
			if time.Now().After(deadline) {
				t.Errorf("goroutines: %d after the test, %d before", runtime.NumGoroutine(), base)
				return
			}
			runtime.Gosched()
			time.Sleep(10 * time.Millisecond)
		}
	})
}
