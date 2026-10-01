package selfupdate_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-core-lib/selfupdate"
	"github.com/maccavelli/go-core-lib/selfupdate/selfupdatetest"
)

// Tests for docs/decisions/0004-PLAN-v1-2-0-interaction-stream.md Step 4:
// whole runs through a Stream.

func streamUpdater(t *testing.T, src selfupdate.ReleaseSource) (*selfupdate.Updater, string) {
	t.Helper()
	exe, policy := tempTarget(t)
	inst, err := selfupdate.NewStandaloneInstaller(selfupdate.InstallOptions{TargetPolicy: policy})
	if err != nil {
		t.Fatal(err)
	}
	sel, err := selfupdate.NewExactAssetSelector([]selfupdate.Platform{goldenPlatform})
	if err != nil {
		t.Fatal(err)
	}
	u, err := selfupdate.New(selfupdate.Config{
		Source: src, Versions: selfupdate.NewStrictVersionPolicy(), Assets: sel, Installer: inst,
		Reporter: selfupdate.DiscardReporter(), Confirmer: selfupdate.NonInteractiveConfirmer(),
		Limits: selfupdate.DefaultLimits(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return u, exe
}

func streamReq(yes bool) selfupdate.Request {
	return selfupdate.Request{
		Product: "demo", CurrentVersion: "v1.0.0", CurrentBuild: selfupdate.ReleaseBuild,
		Platform: goldenPlatform, Yes: yes,
	}
}

// drain reads until Finished, answering any ConfirmNeeded with answer. It
// fails when an interaction follows Finished.
func drain(t *testing.T, s *selfupdate.Stream, answer *bool) ([]selfupdate.Interaction, selfupdate.Finished) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var seen []selfupdate.Interaction
	for {
		it, err := s.Next(ctx)
		if err != nil {
			t.Fatalf("Next after %d interactions: %v", len(seen), err)
		}
		seen = append(seen, it)
		switch v := it.(type) {
		case *selfupdate.ConfirmNeeded:
			if answer == nil {
				t.Fatalf("unexpected ConfirmNeeded %+v", v.Prompt)
			}
			v.Answer(*answer)
		case selfupdate.Finished:
			if it, err := s.Next(ctx); !errors.Is(err, io.EOF) {
				t.Fatalf("after Finished: %#v, %v; want io.EOF", it, err)
			}
			return seen, v
		}
	}
}

func kinds(seen []selfupdate.Interaction) []string {
	var out []string
	for _, it := range seen {
		if p, ok := it.(selfupdate.Progressed); ok {
			out = append(out, p.Event.Kind.String())
		}
	}
	return out
}

func TestStreamDeliversRunInOrder(t *testing.T) {
	selfupdate.CheckNoLeak(t)
	u, exe := streamUpdater(t, selfupdatetest.NewFakeSource("v1.1.0", goldenRelease()))
	seen, fin := drain(t, selfupdate.Start(context.Background(), u, streamReq(true)), nil)
	if fin.Err != nil || !fin.Result.Applied {
		t.Fatalf("Finished = %+v", fin)
	}
	f, err := os.Open(filepath.Join("testdata", "golden", "jsonl-upgrade.golden"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var want []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var line struct{ Kind string }
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			t.Fatal(err)
		}
		want = append(want, line.Kind)
	}
	if got := kinds(seen); !slices.Equal(got, want) {
		t.Fatalf("kinds = %v, want the golden upgrade's %v", got, want)
	}
	if got, _ := os.ReadFile(exe); string(got) != string(releaseBody("v1.1.0")(goldenPlatform)) {
		t.Fatalf("target = %q", got)
	}
}

func TestStreamConfirm(t *testing.T) {
	selfupdate.CheckNoLeak(t)
	for _, answer := range []bool{true, false} {
		u, exe := streamUpdater(t, selfupdatetest.NewFakeSource("v1.1.0", goldenRelease()))
		s := selfupdate.Start(context.Background(), u, streamReq(false))
		var asked *selfupdate.ConfirmNeeded
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		for asked == nil {
			it, err := s.Next(ctx)
			if err != nil {
				t.Fatal(err)
			}
			asked, _ = it.(*selfupdate.ConfirmNeeded)
		}
		cancel()
		if p := asked.Prompt; p.Target != "v1.1.0" || p.Current != "v1.0.0" || p.Operation != selfupdate.OperationUpgrade {
			t.Fatalf("prompt = %+v", p)
		}
		asked.Answer(answer)
		asked.Answer(!answer) // ignored, and must not block
		_, fin := drain(t, s, nil)
		if fin.Err != nil || fin.Result.Applied != answer || fin.Result.Declined == answer {
			t.Fatalf("answer %v: Finished = %+v", answer, fin)
		}
		got, _ := os.ReadFile(exe)
		if answer != (string(got) != "old-bytes") {
			t.Fatalf("answer %v: target = %q", answer, got)
		}
	}
}

func TestStreamCancelWhileConfirming(t *testing.T) {
	selfupdate.CheckNoLeak(t)
	u, exe := streamUpdater(t, selfupdatetest.NewFakeSource("v1.1.0", goldenRelease()))
	s := selfupdate.Start(context.Background(), u, streamReq(false))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var asked *selfupdate.ConfirmNeeded
	for asked == nil {
		it, err := s.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		asked, _ = it.(*selfupdate.ConfirmNeeded)
	}
	s.Cancel()
	_, fin := drain(t, s, nil)
	if !errors.Is(fin.Err, context.Canceled) || fin.Result.Applied {
		t.Fatalf("Finished = %+v, want context.Canceled", fin)
	}
	asked.Answer(true) // after the run: ignored, and must not block
	if got, _ := os.ReadFile(exe); string(got) != "old-bytes" {
		t.Fatalf("target changed: %q", got)
	}
}

// blockingBinary serves the manifest, and a binary body that blocks until
// the run's context ends.
type blockingBinary struct {
	selfupdate.ReleaseSource
	once    sync.Once
	started chan struct{}
}

type ctxReader struct{ ctx context.Context }

func (r ctxReader) Read([]byte) (int, error) {
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

func (b *blockingBinary) OpenAsset(ctx context.Context, rel selfupdate.Release, a selfupdate.Asset) (io.ReadCloser, error) {
	if a.Name == "SHA256SUMS" {
		return b.ReleaseSource.OpenAsset(ctx, rel, a)
	}
	b.once.Do(func() { close(b.started) })
	return io.NopCloser(ctxReader{ctx}), nil
}

// TestStreamCancelDuringDownload: Cancel stops a download in progress, and
// Finished arrives only once the run has returned (O6).
func TestStreamCancelDuringDownload(t *testing.T) {
	selfupdate.CheckNoLeak(t)
	src := &blockingBinary{ReleaseSource: selfupdatetest.NewFakeSource("v1.1.0", goldenRelease()), started: make(chan struct{})}
	u, exe := streamUpdater(t, src)
	s := selfupdate.Start(context.Background(), u, streamReq(true))
	select {
	case <-src.started:
	case <-time.After(10 * time.Second):
		t.Fatal("the binary download never started")
	}
	s.Cancel()
	s.Cancel() // idempotent
	seen, fin := drain(t, s, nil)
	if !errors.Is(fin.Err, context.Canceled) {
		t.Fatalf("Finished = %+v, want context.Canceled", fin)
	}
	if k := kinds(seen); len(k) == 0 || k[len(k)-1] != "failed" {
		t.Fatalf("kinds = %v, want the run's own EventFailed before Finished", k)
	}
	entries, err := os.ReadDir(filepath.Dir(exe))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".demo.selfupdate-") {
			t.Fatalf("staging %s still present when Finished was read", e.Name())
		}
	}
	if got, _ := os.ReadFile(exe); string(got) != "old-bytes" {
		t.Fatalf("target changed: %q", got)
	}
	// A run on an ended context returns at once, and is refused only if the
	// first run still holds the Updater.
	ended, endNow := context.WithCancel(context.Background())
	endNow()
	if _, err := u.Run(ended, streamReq(true)); errors.Is(err, selfupdate.ErrConcurrentUpdate) {
		t.Fatal("the run still held the Updater after Finished")
	}
}

func TestStreamAll(t *testing.T) {
	selfupdate.CheckNoLeak(t)
	u, _ := streamUpdater(t, selfupdatetest.NewFakeSource("v1.1.0", goldenRelease()))
	var last selfupdate.Interaction
	n := 0
	for it := range selfupdate.Start(context.Background(), u, streamReq(true)).All(context.Background()) {
		last, n = it, n+1
	}
	if fin, ok := last.(selfupdate.Finished); !ok || !fin.Result.Applied || n < 2 {
		t.Fatalf("All yielded %d, ending with %#v", n, last)
	}

	u2, _ := streamUpdater(t, selfupdatetest.NewFakeSource("v1.1.0", goldenRelease()))
	s := selfupdate.Start(context.Background(), u2, streamReq(false))
	for it := range s.All(context.Background()) {
		if _, ok := it.(*selfupdate.ConfirmNeeded); ok {
			break
		}
	}
	short, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if it, err := s.Next(short); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("after break: %#v, %v; want the run still waiting", it, err)
	}
	s.Cancel()
	if _, fin := drain(t, s, nil); !errors.Is(fin.Err, context.Canceled) {
		t.Fatalf("Finished = %+v", fin)
	}
}

func TestStreamConcurrentRun(t *testing.T) {
	selfupdate.CheckNoLeak(t)
	u, _ := streamUpdater(t, selfupdatetest.NewFakeSource("v1.1.0", goldenRelease()))
	first := selfupdate.Start(context.Background(), u, streamReq(false))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var asked *selfupdate.ConfirmNeeded
	for asked == nil {
		it, err := first.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		asked, _ = it.(*selfupdate.ConfirmNeeded)
	}
	if _, fin := drain(t, selfupdate.Start(context.Background(), u, streamReq(true)), nil); !errors.Is(fin.Err, selfupdate.ErrConcurrentUpdate) {
		t.Fatalf("second Start = %+v, want ErrConcurrentUpdate", fin)
	}
	asked.Answer(true)
	if _, fin := drain(t, first, nil); fin.Err != nil || !fin.Result.Applied {
		t.Fatalf("first run = %+v", fin)
	}
}

func TestStartReporterFansOut(t *testing.T) {
	selfupdate.CheckNoLeak(t)
	u, _ := streamUpdater(t, selfupdatetest.NewFakeSource("v1.1.0", goldenRelease()))
	rec := &selfupdatetest.RecordingReporter{}
	seen, _ := drain(t, selfupdate.Start(context.Background(), u, streamReq(true), selfupdate.WithReporter(rec)), nil)
	var recorded []string
	for _, k := range rec.Kinds() {
		recorded = append(recorded, k.String())
	}
	if got := kinds(seen); len(got) == 0 || !slices.Equal(got, recorded) {
		t.Fatalf("stream kinds %v, reporter kinds %v", got, recorded)
	}
}

func TestStartConfirmerReplaces(t *testing.T) {
	selfupdate.CheckNoLeak(t)
	u, _ := streamUpdater(t, selfupdatetest.NewFakeSource("v1.1.0", goldenRelease()))
	conf := &selfupdatetest.ScriptedConfirmer{Answers: []bool{true}}
	_, fin := drain(t, selfupdate.Start(context.Background(), u, streamReq(false), selfupdate.WithConfirmer(conf)), nil)
	if fin.Err != nil || !fin.Result.Applied || len(conf.Prompts()) != 1 {
		t.Fatalf("Finished = %+v, prompts = %d", fin, len(conf.Prompts()))
	}
}
