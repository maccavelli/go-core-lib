//go:build unix

package cli

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/maccavelli/go-selfupdate-lib/selfupdate"
	"github.com/maccavelli/go-selfupdate-lib/selfupdate/selfupdatetest"
)

// signalHelperEnv names the target a re-run test binary updates.
const signalHelperEnv = "CLI_SIGNAL_HELPER_TARGET"

// TestSignalCancels sends a real SIGTERM to a child running Run with the
// default signals, once the child says it is blocked in a download (A17).
// The child must exit 1 with a cancellation, and the target must not change.
// Windows cannot signal a child this way; TestSignalsWired covers it there.
func TestSignalCancels(t *testing.T) {
	if path := os.Getenv(signalHelperEnv); path != "" {
		signalHelper(path)
		return
	}
	tg := newTarget(t)
	cmd := exec.Command(os.Args[0], "-test.run=^TestSignalCancels$", "-test.count=1") //nolint:gosec // the test binary itself
	cmd.Env = append(os.Environ(), signalHelperEnv+"="+tg.path)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var seen bytes.Buffer
	sc := bufio.NewScanner(stderr)
	ready := false
	for sc.Scan() {
		seen.WriteString(sc.Text() + "\n")
		if sc.Text() == "READY" {
			ready = true
			break
		}
	}
	if !ready {
		t.Fatalf("the child never got ready (wait: %v):\n%s", cmd.Wait(), seen.String())
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	rest, err := io.ReadAll(stderr)
	if err != nil {
		t.Fatal(err)
	}
	seen.Write(rest)
	err = cmd.Wait()
	ee, ok := err.(*exec.ExitError) //nolint:errorlint // Wait returns *exec.ExitError itself
	if !ok || ee.ExitCode() != 1 {
		t.Fatalf("child exit: %v\n%s", err, seen.String())
	}
	if out := seen.String(); !strings.Contains(out, "update failed:") || !strings.Contains(out, "context canceled") {
		t.Fatalf("child stderr lacks the cancellation:\n%s", out)
	}
	tg.unchanged(t)
}

// signalHelper runs in the child: Run with the default signals against a
// source that blocks in OpenAsset, printing READY once blocked.
func signalHelper(path string) {
	tg := target{path: path}
	fake := selfupdatetest.NewFakeSource("v1.1.0", release("demo", "v1.1.0"))
	src := blockSource{FakeSource: fake, ready: func() {
		if _, err := io.WriteString(os.Stderr, "READY\n"); err != nil {
			os.Exit(3)
		}
	}}
	u, err := buildUpdater(src, tg)
	if err != nil {
		os.Exit(3)
	}
	res, err := Run(context.Background(), u, selfupdate.Request{Product: "demo", CurrentVersion: "v1.0.0",
		CurrentBuild: selfupdate.ReleaseBuild, Yes: true}, Options{Stderr: os.Stderr})
	os.Exit(Exit(os.Stderr, res, err))
}
