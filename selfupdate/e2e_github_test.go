package selfupdate_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-core-lib/selfupdate"
	"github.com/maccavelli/go-core-lib/selfupdate/selfupdatetest"
)

// Tests for docs/decisions/0004-PLAN-v1-1-0-core-api.md Step 11. The fixed
// platform keeps asset names, and so the golden files, identical on every
// host.
var goldenPlatform = selfupdate.Platform{OS: "linux", Arch: "amd64"}

func releaseBody(tag string) func(selfupdate.Platform) []byte {
	return func(p selfupdate.Platform) []byte { return []byte("demo " + tag + " " + p.OS + "/" + p.Arch + "\n") }
}

// tempTarget writes an old binary in a temporary directory the target
// policy allows. It is a plain file, not the running binary.
func tempTarget(t *testing.T) (string, selfupdate.TargetPolicy) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "demo")
	if err := os.WriteFile(exe, []byte("old-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe, selfupdate.TargetPolicy{ExecutablePath: exe, AllowedRoots: []string{dir}}
}

func TestE2EGitHubRedirectedDownload(t *testing.T) {
	plats := []selfupdate.Platform{goldenPlatform}
	gh := selfupdatetest.NewGitHubServer(t, "owner", "demo",
		selfupdatetest.NewRelease("demo", "v1.0.0", plats, releaseBody("v1.0.0")),
		selfupdatetest.NewRelease("demo", "v1.1.0", plats, releaseBody("v1.1.0")))
	src, err := selfupdate.NewGitHubSource(selfupdate.GitHubOptions{
		Repository: selfupdate.Repository{Owner: "owner", Name: "demo"},
		Client:     gh.Client, APIBaseURL: gh.APIBase, UserAgent: "demo/v1.0.0",
		Token: "e2e-token", Limits: selfupdate.DefaultLimits(),
	})
	if err != nil {
		t.Fatal(err)
	}
	exe, policy := tempTarget(t)
	inst, err := selfupdate.NewStandaloneInstaller(selfupdate.InstallOptions{TargetPolicy: policy})
	if err != nil {
		t.Fatal(err)
	}
	sel, err := selfupdate.NewExactAssetSelector(plats)
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
	res, err := u.Run(context.Background(), selfupdate.Request{
		Product: "demo", CurrentVersion: "v1.0.0", CurrentBuild: selfupdate.ReleaseBuild,
		Platform: goldenPlatform, Yes: true,
	})
	if code := selfupdate.ExitCode(res, err); code != 0 || !res.Applied {
		t.Fatalf("ExitCode = %d, res = %+v, err = %v", code, res, err)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if want := releaseBody("v1.1.0")(goldenPlatform); string(got) != string(want) {
		t.Fatalf("target = %q, want %q", got, want)
	}

	reqs := gh.Requests()
	assetHops := 0
	for i, r := range reqs {
		onAPI := r.Host == gh.APIBase.Host
		if onAPI && !r.Authorization {
			t.Errorf("API request %s had no Authorization", r.Path)
		}
		if !onAPI {
			if r.Authorization {
				t.Errorf("the asset host received Authorization on %s", r.Path)
			}
			continue
		}
		if !strings.Contains(r.Path, "/releases/assets/") {
			continue
		}
		// One cross-origin hop per asset: the next request is the download,
		// on the other origin, and the one after is not.
		if i+1 >= len(reqs) || reqs[i+1].Host == gh.APIBase.Host {
			t.Fatalf("asset request %s was not redirected to another origin: %+v", r.Path, reqs)
		}
		if i+2 < len(reqs) && reqs[i+2].Host != gh.APIBase.Host {
			t.Fatalf("asset request %s took more than one hop: %+v", r.Path, reqs)
		}
		assetHops++
	}
	if assetHops != 2 {
		t.Fatalf("asset requests = %d, want the manifest and the binary: %+v", assetHops, reqs)
	}
}
