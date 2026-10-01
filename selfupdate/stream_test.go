package selfupdate

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// Tests for docs/decisions/0004-PLAN-v1-2-0-interaction-stream.md Step 4:
// the queue itself. Whole runs are in stream_run_test.go.

func newTestStream() *Stream {
	return &Stream{cancel: func() {}, wake: make(chan struct{}, 1), closed: make(chan struct{})}
}

func progressAt(n int64) Progressed {
	return Progressed{Event: Event{Kind: EventProgress, Bytes: n}}
}

func TestStreamProgressLatestWins(t *testing.T) {
	checkNoLeak(t)
	s := newTestStream()
	for i := range int64(1000) {
		s.push(progressAt(i))
	}
	s.push(Progressed{Event: Event{Kind: EventVerified}})
	for i := range int64(1000) {
		s.push(progressAt(1000 + i))
	}
	s.mu.Lock()
	n := len(s.queue)
	s.mu.Unlock()
	if n != 3 {
		t.Fatalf("queue holds %d items, want 3: the last progress, the lifecycle event, the last progress", n)
	}
	want := []Event{{Kind: EventProgress, Bytes: 999}, {Kind: EventVerified}, {Kind: EventProgress, Bytes: 1999}}
	for i, w := range want {
		it, err := s.Next(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if p, ok := it.(Progressed); !ok || p.Event != w {
			t.Fatalf("item %d = %#v, want %+v", i, it, w)
		}
	}
}

func TestStreamNextContext(t *testing.T) {
	checkNoLeak(t)
	s := newTestStream()
	s.push(Progressed{Event: Event{Kind: EventSelected}})
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if it, err := s.Next(ended); !errors.Is(err, context.Canceled) || it != nil {
		t.Fatalf("Next on an ended context = %v, %v", it, err)
	}
	it, err := s.Next(context.Background())
	if p, ok := it.(Progressed); err != nil || !ok || p.Event.Kind != EventSelected {
		t.Fatalf("the item was consumed by the cancelled Next: %v, %v", it, err)
	}
	short, cancel2 := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel2()
	if _, err := s.Next(short); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Next on an empty stream = %v, want the deadline", err)
	}
}

// TestStreamFinishedThenEOF: Finished is returned once, then io.EOF, and a
// second waiter is released by Finished rather than left blocked.
func TestStreamFinishedThenEOF(t *testing.T) {
	checkNoLeak(t)
	s := newTestStream()
	results := make(chan error, 2)
	for range 2 {
		go func() {
			it, err := s.Next(context.Background())
			if _, ok := it.(Finished); ok {
				err = errFinishedSeen
			}
			results <- err
		}()
	}
	// Not an ordering handshake: the assertions hold in any interleaving. The
	// pause makes both waiters block first, so the path where Finished must
	// release a blocked waiter is the one exercised.
	time.Sleep(20 * time.Millisecond)
	s.push(Finished{})
	var finished, eof int
	for range 2 {
		select {
		case err := <-results:
			switch {
			case errors.Is(err, errFinishedSeen):
				finished++
			case errors.Is(err, io.EOF):
				eof++
			default:
				t.Fatalf("waiter got %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("a waiter was left blocked after Finished")
		}
	}
	if finished != 1 || eof != 1 {
		t.Fatalf("finished=%d eof=%d, want one each", finished, eof)
	}
	if _, err := s.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("Next after Finished = %v, want io.EOF", err)
	}
}

var errFinishedSeen = errors.New("finished")

func TestConfirmNeededRepliesOnce(t *testing.T) {
	checkNoLeak(t)
	c := &ConfirmNeeded{reply: make(chan confirmReply, 1)}
	done := make(chan struct{})
	go func() {
		c.Answer(true)
		c.Answer(false)
		c.Cancel(errors.New("late"))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a second reply blocked")
	}
	if r := <-c.reply; !r.ok || r.err != nil {
		t.Fatalf("reply = %+v, want the first Answer(true)", r)
	}
	cancelled := &ConfirmNeeded{reply: make(chan confirmReply, 1)}
	cancelled.Cancel(nil)
	if r := <-cancelled.reply; r.ok || !errors.Is(r.err, context.Canceled) {
		t.Fatalf("Cancel(nil) reply = %+v, want context.Canceled", r)
	}
}

func TestStartInvalid(t *testing.T) {
	checkNoLeak(t)
	env := newContractEnv(t)
	u := env.build(t)
	cases := []struct {
		name string
		u    *Updater
		opts []RunOption
		want string
	}{
		{"nil updater", nil, nil, "selfupdate: updater is nil"},
		{"nil reporter", u, []RunOption{WithReporter(nil)}, "selfupdate: WithReporter: reporter is nil"},
	}
	for _, c := range cases {
		s := Start(context.Background(), c.u, yesReq(), c.opts...)
		it, err := s.Next(context.Background())
		f, ok := it.(Finished)
		if err != nil || !ok || f.Err == nil || f.Err.Error() != c.want {
			t.Errorf("%s: first interaction = %#v, %v; want Finished with %q", c.name, it, err, c.want)
		}
		if _, err := s.Next(context.Background()); !errors.Is(err, io.EOF) {
			t.Errorf("%s: after Finished = %v, want io.EOF", c.name, err)
		}
	}
	if len(*env.log) != 0 {
		t.Fatalf("a run started: %v", *env.log)
	}
}
