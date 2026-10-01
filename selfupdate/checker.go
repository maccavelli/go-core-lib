package selfupdate

import (
	"context"
	"fmt"
)

// CheckerConfig composes a Checker: the discovery half of Config, with no
// Installer, Reporter or Confirmer
// (docs/decisions/0004-MADR-evolve-selfupdate-api-and-tui-support.md G3).
type CheckerConfig struct {
	// Source discovers releases.
	Source ReleaseSource
	// Versions validates and compares tags.
	Versions VersionPolicy
	// Assets selects exact raw-binary names.
	Assets AssetSelector
	// Limits bound the selected assets' advertised sizes.
	Limits Limits
}

// Checker answers "is there an update?" without resolving, locking or
// touching a target, prompting, reporting, or downloading an asset body. It
// holds no mutable state and is safe for concurrent use.
type Checker struct {
	source   ReleaseSource
	versions VersionPolicy
	assets   AssetSelector
	limits   Limits
}

// CheckRequest is one availability question.
type CheckRequest struct {
	// Product is the executable basename used in exact asset names.
	Product string
	// CurrentVersion is the running identity. For ReleaseBuild it must
	// satisfy the configured VersionPolicy.
	CurrentVersion string
	// CurrentBuild distinguishes release and local binaries.
	CurrentBuild BuildKind
	// TargetVersion selects an exact tag. Empty means the latest stable
	// release.
	TargetVersion string
	// Platform selects the asset matrix entry. Zero means runtime GOOS/GOARCH.
	Platform Platform
	// Channel selects a release channel, as Request.Channel does. It is part
	// of CheckCached's key, so each channel keeps its own answer.
	Channel string
}

// Availability is the answer to a CheckRequest. Its names match Result
// (0004-MADR amendment A3).
type Availability struct {
	// Product is the requested product name.
	Product string
	// CurrentVersion is the running identity supplied in the request.
	CurrentVersion string
	// TargetVersion is the selected release tag.
	TargetVersion string
	// ReleaseURL is the selected release's HTML URL when known.
	ReleaseURL string
	// AssetName is the exact selected executable asset name.
	AssetName string
	// Operation is the action an apply would take.
	Operation Operation
	// Available is true when Operation is not OperationNone.
	Available bool
	// ForceRequired is true when applying needs Request.Force: the running
	// binary is a local build.
	ForceRequired bool
}

// NewChecker constructs a Checker. Source, Versions and Assets are required,
// and none may be a typed nil; Limits must be valid.
func NewChecker(cfg CheckerConfig) (*Checker, error) {
	if isNil(cfg.Source) {
		return nil, fmt.Errorf("selfupdate: source is required")
	}
	if isNil(cfg.Versions) {
		return nil, fmt.Errorf("selfupdate: version policy is required")
	}
	if isNil(cfg.Assets) {
		return nil, fmt.Errorf("selfupdate: asset selector is required")
	}
	if err := cfg.Limits.valid(); err != nil {
		return nil, err
	}
	return &Checker{source: cfg.Source, versions: cfg.Versions, assets: cfg.Assets, limits: cfg.Limits}, nil
}

// Checker returns a Checker that shares the Updater's source, version
// policy, asset selector and limits.
func (u *Updater) Checker() *Checker {
	return &Checker{source: u.source, versions: u.versions, assets: u.assets, limits: u.limits}
}

// Check reports whether an update is available. Unlike Run with CheckOnly,
// it resolves no target and needs no Installer, Confirmer or Reporter, and
// it reports availability as a value rather than as ErrUpdateAvailable. A
// latest release older than the running one is ErrLatestOlder, as in Run.
func (c *Checker) Check(ctx context.Context, cr CheckRequest) (Availability, error) {
	req, err := c.prepare(cr)
	if err != nil {
		return Availability{}, err
	}
	return c.checkPrepared(ctx, req)
}

// prepare validates a CheckRequest as a check-only Request and normalizes
// its platform. Errors are wrapped as Run wraps them.
func (c *Checker) prepare(cr CheckRequest) (Request, error) {
	req := Request{
		Product:        cr.Product,
		CurrentVersion: cr.CurrentVersion,
		CurrentBuild:   cr.CurrentBuild,
		TargetVersion:  cr.TargetVersion,
		Platform:       cr.Platform,
		CheckOnly:      true,
		Channel:        cr.Channel,
	}
	if err := validateRequest(req, c.versions); err != nil {
		if validateProduct(req.Product) != nil {
			// An invalid product name is not safe to put in the prefix.
			return Request{}, err
		}
		return Request{}, wrapRun(req, err)
	}
	req.Platform = normalizePlatform(req.Platform)
	return req, nil
}

func (c *Checker) checkPrepared(ctx context.Context, req Request) (Availability, error) {
	rel, sel, op, err := c.discover(ctx, req)
	if err != nil {
		return Availability{}, wrapRun(req, err)
	}
	available := op != OperationNone
	return Availability{
		Product:        req.Product,
		CurrentVersion: req.CurrentVersion,
		TargetVersion:  rel.Tag,
		ReleaseURL:     rel.URL,
		AssetName:      sel.Binary.Name,
		Operation:      op,
		Available:      available,
		ForceRequired:  available && req.CurrentBuild == LocalBuild,
	}, nil
}

// discover fetches the release for a validated request with a normalized
// platform, applies every release and asset check, and classifies the
// operation. Run and Check share it, so they cannot disagree. Errors are
// returned unwrapped.
func (c *Checker) discover(ctx context.Context, req Request) (Release, Selection, Operation, error) {
	rel, fromLatest, err := c.fetchRelease(ctx, req)
	if err != nil {
		return Release{}, Selection{}, OperationNone, err
	}
	// PLAN §4.6 step 4 order: immutable, then state, then tag. The tag is
	// untrusted until Validate passes, so it is always quoted.
	if !rel.Immutable {
		return Release{}, Selection{}, OperationNone, fmt.Errorf("selfupdate: release %q is not immutable: %w", rel.Tag, ErrMutableRelease)
	}
	if rel.Draft || rel.Prerelease {
		return Release{}, Selection{}, OperationNone, fmt.Errorf("selfupdate: release %q is not a stable published release", rel.Tag)
	}
	if err := c.versions.Validate(rel.Tag); err != nil {
		return Release{}, Selection{}, OperationNone, err
	}
	sel, err := c.assets.Select(rel, req.Product, req.Platform)
	if err != nil {
		return Release{}, Selection{}, OperationNone, err
	}
	// Only the selected binary and manifest are checked for state, size and
	// digest syntax, each against its own limit, and before check mode can
	// report anything (PLAN §4.6 step 4; 0003-MADR A1 and A5).
	if err := validateAssetMetadata(sel.Binary, c.limits.Executable); err != nil {
		return Release{}, Selection{}, OperationNone, err
	}
	if err := validateAssetMetadata(sel.Manifest, c.limits.Manifest); err != nil {
		return Release{}, Selection{}, OperationNone, err
	}
	op, err := classifyOperation(c.versions, req, rel.Tag, fromLatest)
	if err != nil {
		return Release{}, Selection{}, OperationNone, err
	}
	return rel, sel, op, nil
}

func (c *Checker) fetchRelease(ctx context.Context, req Request) (Release, bool, error) {
	if req.TargetVersion == "" {
		rel, err := c.source.Latest(ctx)
		return rel, true, err
	}
	rel, err := c.source.ByTag(ctx, req.TargetVersion)
	if err != nil {
		return Release{}, false, err
	}
	if rel.Tag != req.TargetVersion {
		return Release{}, false, fmt.Errorf("selfupdate: source returned release %q for requested %q: %w",
			rel.Tag, req.TargetVersion, ErrIntegrity)
	}
	return rel, false, nil
}
