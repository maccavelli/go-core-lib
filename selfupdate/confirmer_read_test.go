package selfupdate

import (
	"context"
	"io"
	"os"
	"runtime"
	"testing"
	"time"
)

var upgradePrompt = Prompt{Product: "demo", Current: "v1.0.0", Target: "v1.1.0", Operation: OperationUpgrade}

// ttyPipe is a pipe the confirmer treats as a terminal.
func ttyPipe(t *testing.T) (r, w *os.File) {
	t.Helper()
	setSeam(t, &isTerminal, func(int) bool { return true })
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})
	return r, w
}

// TestConfirmLeavesHostInput: an answered Confirm reads its line and no
// further, so the host program reads its own next input (0004-MADR R2).
func TestConfirmLeavesHostInput(t *testing.T) {
	r, w := ttyPipe(t)
	if _, err := io.WriteString(w, "y\nhost-command\n"); err != nil {
		t.Fatal(err)
	}
	ok, err := NewTerminalConfirmer(r, io.Discard).Confirm(context.Background(), upgradePrompt)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	got := make(chan string, 1)
	go func() {
		buf := make([]byte, 64)
		n, _ := r.Read(buf)
		got <- string(buf[:n])
	}()
	select {
	case s := <-got:
		if s != "host-command\n" {
			t.Fatalf("host read %q, want its own line", s)
		}
	case <-time.After(2 * time.Second):
		_ = w.Close() // release the blocked read
		t.Fatal("the host's input was consumed by the confirmer")
	}
}

// TestConfirmersShareInput: a second confirmer on the same input receives
// its own answer; the first leaves nothing reading (0004-MADR R2).
func TestConfirmersShareInput(t *testing.T) {
	r, w := ttyPipe(t)
	if _, err := io.WriteString(w, "n\n"); err != nil {
		t.Fatal(err)
	}
	if ok, err := NewTerminalConfirmer(r, io.Discard).Confirm(context.Background(), upgradePrompt); err != nil || ok {
		t.Fatalf("first: ok=%v err=%v", ok, err)
	}
	// Not an ordering handshake: the pause only gives a reader wrongly left
	// behind by the first confirmer time to block on the pipe, so the
	// failure it causes is observed rather than raced.
	time.Sleep(100 * time.Millisecond)
	if _, err := io.WriteString(w, "y\n"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ok, err := NewTerminalConfirmer(r, io.Discard).Confirm(ctx, upgradePrompt)
	if err != nil || !ok {
		t.Fatalf("second: ok=%v err=%v, want its own \"y\"", ok, err)
	}
}

// TestConfirmLeaksNoReader: once a Confirm is answered, no goroutine is
// left reading the input (0004-MADR R2).
func TestConfirmLeaksNoReader(t *testing.T) {
	r, w := ttyPipe(t)
	base := runtime.NumGoroutine()
	if _, err := io.WriteString(w, "y\n"); err != nil {
		t.Fatal(err)
	}
	if ok, err := NewTerminalConfirmer(r, io.Discard).Confirm(context.Background(), upgradePrompt); err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > base {
		if time.Now().After(deadline) {
			t.Fatalf("goroutines: %d after an answered Confirm, %d before", runtime.NumGoroutine(), base)
		}
		runtime.Gosched()
		time.Sleep(10 * time.Millisecond)
	}
}
