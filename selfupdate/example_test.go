package selfupdate_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/maccavelli/go-core-lib/selfupdate"
)

func ExampleNew_standalone() {
	src, err := selfupdate.NewGitHubSource(selfupdate.GitHubOptions{
		Repository: selfupdate.Repository{Owner: "maccavelli", Name: "prepare-commit-msg"},
		Client:     &http.Client{Timeout: 15 * time.Minute},
		UserAgent:  "prepare-commit-msg/v1.2.0",
		Limits:     selfupdate.DefaultLimits(),
	})
	if err != nil {
		fmt.Println("source:", err)
		return
	}
	selector, err := selfupdate.NewExactAssetSelector([]selfupdate.Platform{
		{OS: "linux", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
	})
	if err != nil {
		fmt.Println("selector:", err)
		return
	}
	installer, err := selfupdate.NewStandaloneInstaller(selfupdate.InstallOptions{})
	if err != nil {
		fmt.Println("installer:", err)
		return
	}
	updater, err := selfupdate.New(selfupdate.Config{
		Source:    src,
		Versions:  selfupdate.NewStrictVersionPolicy(),
		Assets:    selector,
		Installer: installer,
		Reporter:  selfupdate.NewTextReporter(os.Stderr),
		Confirmer: selfupdate.NewTerminalConfirmer(os.Stdin, os.Stderr),
		Limits:    selfupdate.DefaultLimits(),
	})
	if err != nil {
		fmt.Println("new:", err)
		return
	}
	_ = updater
	fmt.Println("standalone updater ready")
	// Output: standalone updater ready
}

func ExampleNewTextReporter() {
	reporter := selfupdate.NewTextReporter(os.Stdout)
	_ = reporter.Report(context.Background(), selfupdate.Event{
		Kind:    selfupdate.EventSelected,
		Product: "demo",
		Target:  "v1.1.0",
		Asset:   "demo-linux-amd64",
	})
	// Output: selfupdate: selected product=demo target=v1.1.0 asset=demo-linux-amd64
}

// exampleService is a stand-in service manager. A real program binds its
// systemd, launchd or Windows service adapter here.
type exampleService struct{}

func (exampleService) Installed(context.Context, string) (bool, error) { return true, nil }
func (exampleService) Running(context.Context, string) (bool, error)   { return true, nil }
func (exampleService) Stop(context.Context, string) error              { return nil }
func (exampleService) Start(context.Context, string) error             { return nil }
func (exampleService) WaitHealthy(context.Context, string) error       { return nil }

// exampleDefinition rewrites nothing; a real Reconciler updates the service
// definition to point at the new executable and can restore it.
type exampleDefinition struct{}

func (exampleDefinition) Reconcile(context.Context, string, string) (selfupdate.ReconcileResult, error) {
	return selfupdate.ReconcileResult{}, nil
}

func (exampleDefinition) Restore(context.Context, string, selfupdate.ReconcileResult) error {
	return nil
}

func ExampleNewManagedInstaller() {
	inner, err := selfupdate.NewStandaloneInstaller(selfupdate.InstallOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}
	managed, err := selfupdate.NewManagedInstaller(inner, exampleService{}, exampleDefinition{})
	if err != nil {
		fmt.Println(err)
		return
	}
	// managed is passed as Config.Installer: Install stops the running
	// service, replaces the binary, reconciles, starts, and waits healthy,
	// rolling everything back if any step fails.
	fmt.Printf("%T\n", managed)
	// Output: *selfupdate.ManagedInstaller
}

func ExampleExitCode() {
	checkFound := fmt.Errorf("wrapped: %w", selfupdate.ErrUpdateAvailable)
	fmt.Println(selfupdate.ExitCode(selfupdate.Result{}, nil))
	fmt.Println(selfupdate.ExitCode(selfupdate.Result{Checked: true}, checkFound))
	fmt.Println(selfupdate.ExitCode(selfupdate.Result{}, fmt.Errorf("network down")))
	// Output:
	// 0
	// 10
	// 1
}
