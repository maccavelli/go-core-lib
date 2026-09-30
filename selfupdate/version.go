package selfupdate

import (
	"fmt"
	"regexp"

	"golang.org/x/mod/semver"
)

var (
	productRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	versionRe = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)
)

type strictVersionPolicy struct{}

// NewStrictVersionPolicy returns the canonical stable-tag policy. Tags must
// match vMAJOR.MINOR.PATCH with no leading zeroes, prerelease, or build
// metadata.
func NewStrictVersionPolicy() VersionPolicy {
	return strictVersionPolicy{}
}

// Validate implements VersionPolicy.
func (strictVersionPolicy) Validate(tag string) error {
	if !versionRe.MatchString(tag) {
		return fmt.Errorf("selfupdate: %q is not a strict stable tag", tag)
	}
	if !semver.IsValid(tag) {
		return fmt.Errorf("selfupdate: %q is not a strict stable tag", tag)
	}
	return nil
}

// Compare implements VersionPolicy. Both arguments are validated first.
func (p strictVersionPolicy) Compare(a, b string) (int, error) {
	if err := p.Validate(a); err != nil {
		return 0, err
	}
	if err := p.Validate(b); err != nil {
		return 0, err
	}
	return semver.Compare(a, b), nil
}

func validateProduct(product string) error {
	if !productRe.MatchString(product) {
		return fmt.Errorf("selfupdate: invalid product name %q", product)
	}
	return nil
}

// validateRequest checks a request's shape, and validates its versions with
// the configured policy rather than a hardcoded strict one (0004-MADR G2).
func validateRequest(req Request, versions VersionPolicy) error {
	if err := validateProduct(req.Product); err != nil {
		return err
	}
	switch req.CurrentBuild {
	case ReleaseBuild, LocalBuild:
	default:
		return fmt.Errorf("selfupdate: unknown build kind %s", req.CurrentBuild)
	}
	if (req.Platform.OS == "") != (req.Platform.Arch == "") {
		return fmt.Errorf("selfupdate: platform OS and architecture must both be set or both empty")
	}
	if req.CheckOnly && req.Yes {
		return fmt.Errorf("selfupdate: --check and --yes are contradictory")
	}
	if req.CheckOnly && req.Force {
		return fmt.Errorf("selfupdate: --check and --force are contradictory")
	}
	if req.CurrentBuild == ReleaseBuild {
		if err := versions.Validate(req.CurrentVersion); err != nil {
			return err
		}
	}
	if req.TargetVersion != "" {
		if err := versions.Validate(req.TargetVersion); err != nil {
			return err
		}
	}
	return nil
}

// classifyOperation reports the action for a validated request and already
// validated selected tag. fromLatest is true when the selection was the
// latest stable release rather than an exact --version.
func classifyOperation(versions VersionPolicy, req Request, selected string, fromLatest bool) (Operation, error) {
	if err := versions.Validate(selected); err != nil {
		return OperationNone, err
	}
	if req.CurrentBuild == LocalBuild {
		if req.CheckOnly {
			return OperationReplaceLocal, nil
		}
		if !req.Force {
			return OperationNone, fmt.Errorf("%w to install %s", ErrForceRequired, selected)
		}
		return OperationReplaceLocal, nil
	}
	cmp, err := versions.Compare(req.CurrentVersion, selected)
	if err != nil {
		return OperationNone, err
	}
	switch {
	case cmp < 0:
		return OperationUpgrade, nil
	case cmp == 0:
		if req.Force && !req.CheckOnly {
			return OperationReinstall, nil
		}
		return OperationNone, nil
	default:
		if fromLatest {
			return OperationNone, fmt.Errorf("selfupdate: latest release %s is older than running %s: %w",
				selected, req.CurrentVersion, ErrLatestOlder)
		}
		return OperationRollback, nil
	}
}
