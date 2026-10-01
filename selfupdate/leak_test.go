package selfupdate

import (
	"runtime"
	"testing"
	"time"
)

// checkNoLeak fails the test when more goroutines run than when it was
// called. It checks once the test has cleaned up, and also whenever the
// returned function is called. It polls for up to 2 s, because a goroutine
// that is ending is still counted for a moment (0004-MADR H6). The sleep
// only paces that poll; it orders nothing.
//
// Call it first. A test whose cleanup would release a leaked goroutine,
// such as closing the pipe a reader blocks on, checks before that cleanup:
//
//	defer checkNoLeak(t)()
func checkNoLeak(t testing.TB) (checkNow func()) {
	t.Helper()
	base := runtime.NumGoroutine()
	check := func() {
		deadline := time.Now().Add(2 * time.Second)
		for runtime.NumGoroutine() > base {
			if time.Now().After(deadline) {
				t.Errorf("goroutines: %d now, %d when the test began", runtime.NumGoroutine(), base)
				return
			}
			runtime.Gosched()
			time.Sleep(10 * time.Millisecond)
		}
	}
	t.Cleanup(check)
	return check
}
