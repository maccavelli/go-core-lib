package buildinfo

import (
	"runtime/debug"
	"testing"
)

// stamp sets the two linker variables for one test.
func stamp(t *testing.T, v, k string) {
	t.Helper()
	oldV, oldK := version, kind
	version, kind = v, k
	t.Cleanup(func() { version, kind = oldV, oldK })
}

// buildInfo replaces debug.ReadBuildInfo for one test.
func buildInfo(t *testing.T, bi *debug.BuildInfo, ok bool) {
	t.Helper()
	old := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) { return bi, ok }
	t.Cleanup(func() { readBuildInfo = old })
}

func TestIdentityKinds(t *testing.T) {
	buildInfo(t, nil, false)
	for _, tc := range []struct {
		version, kind string
		want          Kind
		reason        string
	}{
		{"v1.2.3", "release", KindRelease, ""},
		{"v0.0.0", "release", KindRelease, ""},
		{"v10.20.30", "release", KindRelease, ""},
		{"v1.4.0-rc.1", "release", KindRelease, ""},
		{"v1.4.0-beta2.0", "release", KindRelease, ""},
		{"v1.2", "release", KindLocal, `buildinfo: stamped version "v1.2" is not a release tag`},
		{"1.2.3", "release", KindLocal, `buildinfo: stamped version "1.2.3" is not a release tag`},
		{"v1.2.3-rc.01", "release", KindLocal, `buildinfo: stamped version "v1.2.3-rc.01" is not a release tag`},
		{"v1.2.3-RC.1", "release", KindLocal, `buildinfo: stamped version "v1.2.3-RC.1" is not a release tag`},
		{"v1.2.3-rc", "release", KindLocal, `buildinfo: stamped version "v1.2.3-rc" is not a release tag`},
		{"v1.2.3+meta", "release", KindLocal, `buildinfo: stamped version "v1.2.3+meta" is not a release tag`},
		{"v01.2.3", "release", KindLocal, `buildinfo: stamped version "v01.2.3" is not a release tag`},
		{"", "release", KindLocal, `buildinfo: stamped version "" is not a release tag`},
		{"v1.2.3\n", "release", KindLocal, `buildinfo: stamped version "v1.2.3\n" is not a release tag`},
		{"v1.2.3", "Release", KindLocal, ""},
		{"v1.2.3", "RELEASE", KindLocal, ""},
		{"v1.2.3", "local", KindLocal, ""},
		{"v1.2.3", "", KindLocal, ""},
		{"", "", KindLocal, ""},
	} {
		stamp(t, tc.version, tc.kind)
		got := Identity()
		if got.Kind != tc.want || got.Reason != tc.reason || got.Version != tc.version {
			t.Errorf("version=%q kind=%q: got kind=%s reason=%q version=%q; want kind=%s reason=%q",
				tc.version, tc.kind, got.Kind, got.Reason, got.Version, tc.want, tc.reason)
		}
	}
}

func TestIdentityIgnoresModuleVersion(t *testing.T) {
	buildInfo(t, &debug.BuildInfo{
		GoVersion: "go1.27.1",
		Main:      debug.Module{Path: "example.com/demo", Version: "v9.9.9"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "0123456789abcdef0123"},
			{Key: "vcs.time", Value: "2026-10-02T00:00:00Z"},
			{Key: "vcs.modified", Value: "true"},
		},
	}, true)
	stamp(t, "", "")
	got := Identity()
	if got.Kind != KindLocal {
		t.Fatalf("unstamped with module version v9.9.9: kind %s, want local", got.Kind)
	}
	want := Info{
		Kind: KindLocal, Module: "example.com/demo", ModuleVersion: "v9.9.9", GoVersion: "go1.27.1",
		Revision: "0123456789abcdef0123", Time: "2026-10-02T00:00:00Z", Modified: true,
	}
	if got != want {
		t.Fatalf("Identity() = %+v, want %+v", got, want)
	}
}

func TestInfoCurrent(t *testing.T) {
	for _, tc := range []struct {
		info Info
		want string
	}{
		{Info{Version: "v1.2.3", ModuleVersion: "v9.9.9"}, "v1.2.3"},
		{Info{ModuleVersion: "v9.9.9"}, "v9.9.9"},
		{Info{ModuleVersion: "(devel)"}, "dev"},
		{Info{}, "dev"},
	} {
		if got := tc.info.Current(); got != tc.want {
			t.Errorf("%+v.Current() = %q, want %q", tc.info, got, tc.want)
		}
	}
}

func TestKindString(t *testing.T) {
	for _, tc := range []struct {
		k    Kind
		want string
	}{{KindUnknown, "unknown"}, {KindRelease, "release"}, {KindLocal, "local"}, {Kind(9), "unknown"}} {
		if got := tc.k.String(); got != tc.want {
			t.Errorf("Kind(%d).String() = %q, want %q", tc.k, got, tc.want)
		}
	}
}

func TestInfoString(t *testing.T) {
	for _, tc := range []struct {
		info Info
		want string
	}{
		{Info{Version: "v1.2.3", Kind: KindRelease}, "v1.2.3 (release)"},
		{Info{Kind: KindLocal, Revision: "0123456789abcdef"}, "dev (local) 0123456789ab"},
		{Info{Kind: KindLocal, Revision: "abc", Modified: true}, "dev (local) abc-dirty"},
		{Info{Kind: KindLocal, Modified: true}, "dev (local)"},
	} {
		if got := tc.info.String(); got != tc.want {
			t.Errorf("%+v.String() = %q, want %q", tc.info, got, tc.want)
		}
	}
}

func TestLDFlags(t *testing.T) {
	want := "-X github.com/maccavelli/go-core-lib/buildinfo.version=v1.2.3 -X github.com/maccavelli/go-core-lib/buildinfo.kind=release"
	if got := LDFlags("v1.2.3"); got != want {
		t.Fatalf("LDFlags = %q, want %q", got, want)
	}
}
