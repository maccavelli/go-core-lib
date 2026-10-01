package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	gitHubAPIVersion  = "2026-03-10"
	gitHubAcceptJSON  = "application/vnd.github+json"
	gitHubAcceptAsset = "application/octet-stream"
	defaultAPIOrigin  = "https://api.github.com"
	maxRedirects      = 10
)

var (
	lookupEnv = os.Getenv
	timeNow   = time.Now
)

// GitHubSource is the canonical GitHub Releases implementation of ReleaseSource.
type GitHubSource struct {
	repo      Repository
	client    *http.Client
	apiBase   *url.URL
	userAgent string
	token     string // the explicit Token, else GH_TOKEN, else GITHUB_TOKEN
	explicit  bool   // token came from GitHubOptions.Token
	envName   string // the variable token came from, when not explicit
	provider  CredentialProvider
	observer  CredentialObserver
	limits    Limits
	now       func() time.Time
	cred      credentialState
}

// credentialState is the source's one resolved credential, resolved on
// the first API request and shared by every later one.
type credentialState struct {
	mu           sync.Mutex
	resolved     bool
	has          bool
	cred         Credential
	fromProvider bool
	retried      bool
	accepted     bool
}

// NewGitHubSource validates options, clones the supplied client, and resolves
// the token once. It never mutates the caller's client or URL.
func NewGitHubSource(opts GitHubOptions) (*GitHubSource, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("selfupdate: github client is required")
	}
	if err := opts.Limits.valid(); err != nil {
		return nil, err
	}
	if err := validateGitHubName("owner", opts.Repository.Owner); err != nil {
		return nil, err
	}
	if err := validateGitHubName("repository", opts.Repository.Name); err != nil {
		return nil, err
	}
	if err := validateUserAgent(opts.UserAgent); err != nil {
		return nil, err
	}
	base, err := normalizeAPIBase(opts.APIBaseURL)
	if err != nil {
		return nil, err
	}
	cloned := *opts.Client
	token, envName := resolveToken(opts.Token)
	src := &GitHubSource{
		repo:      opts.Repository,
		client:    &cloned,
		apiBase:   base,
		userAgent: opts.UserAgent,
		token:     token,
		explicit:  opts.Token != "",
		envName:   envName,
		limits:    opts.Limits,
		now:       timeNow,
	}
	if !isNil(opts.Credentials) {
		src.provider = opts.Credentials
	}
	if !isNil(opts.Observer) {
		src.observer = opts.Observer
	}
	src.client.CheckRedirect = src.checkRedirect
	return src, nil
}

func validateGitHubName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("selfupdate: github %s is required", kind)
	}
	if strings.ContainsAny(name, `/\:?`) || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fmt.Errorf("selfupdate: github %s contains illegal characters", kind)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("selfupdate: github %s must not be a dot segment", kind)
	}
	return nil
}

func validateUserAgent(ua string) error {
	if strings.TrimSpace(ua) == "" {
		return fmt.Errorf("selfupdate: user-agent is required")
	}
	if strings.IndexFunc(ua, unicode.IsControl) >= 0 {
		return fmt.Errorf("selfupdate: user-agent contains illegal characters")
	}
	return nil
}

// resolveToken returns the explicit token, else GH_TOKEN, else
// GITHUB_TOKEN, with the name of the variable it came from.
func resolveToken(explicit string) (token, envName string) {
	if explicit != "" {
		return explicit, ""
	}
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if v := lookupEnv(name); v != "" {
			return v, name
		}
	}
	return "", ""
}

func normalizeAPIBase(raw *url.URL) (*url.URL, error) {
	var base *url.URL
	if raw == nil {
		parsed, err := url.Parse(defaultAPIOrigin)
		if err != nil {
			return nil, err
		}
		base = parsed
	} else {
		cloned := *raw
		base = &cloned
	}
	if base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("selfupdate: github API base must not include user, query, or fragment")
	}
	if base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("selfupdate: github API base is incomplete")
	}
	if !isLoopbackHost(base.Hostname()) && !strings.EqualFold(base.Scheme, "https") {
		return nil, fmt.Errorf("selfupdate: github API base must be https")
	}
	base.Path = strings.TrimSuffix(base.Path, "/")
	return base, nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *GitHubSource) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("selfupdate: too many redirects")
	}
	// Any non-loopback hop must stay on HTTPS (mcplib 0005-PLAN §4.2;
	// 0003-MADR A3). The URL is not echoed: a redirect target can carry
	// signed query parameters.
	if !strings.EqualFold(req.URL.Scheme, "https") && !isLoopbackHost(req.URL.Hostname()) {
		return fmt.Errorf("selfupdate: refusing redirect to a non-https location")
	}
	if !sameOrigin(req.URL, s.apiBase) {
		req.Header.Del("Authorization")
		// A provider's own header is scrubbed too: a credential goes only
		// to the origin it was requested for (0004-MADR G10).
		s.cred.mu.Lock()
		header := s.cred.cred.Header
		s.cred.mu.Unlock()
		if header != "" {
			req.Header.Del(header)
		}
	}
	return nil
}

func sameOrigin(u, base *url.URL) bool {
	return strings.EqualFold(u.Scheme, base.Scheme) && strings.EqualFold(u.Host, base.Host)
}

func (s *GitHubSource) apiURL(elem ...string) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(s.apiBase.String(), "/"))
	for _, e := range elem {
		b.WriteByte('/')
		b.WriteString(url.PathEscape(e))
	}
	return b.String()
}

// Latest implements ReleaseSource.
func (s *GitHubSource) Latest(ctx context.Context) (Release, error) {
	return s.getRelease(ctx, s.apiURL("repos", s.repo.Owner, s.repo.Name, "releases", "latest"))
}

// ByTag implements ReleaseSource.
func (s *GitHubSource) ByTag(ctx context.Context, tag string) (Release, error) {
	if tag == "" || strings.ContainsAny(tag, `/\:`) || strings.IndexFunc(tag, unicode.IsControl) >= 0 {
		return Release{}, fmt.Errorf("selfupdate: invalid release tag %q", tag)
	}
	rel, err := s.getRelease(ctx, s.apiURL("repos", s.repo.Owner, s.repo.Name, "releases", "tags", tag))
	if err != nil {
		return Release{}, err
	}
	if rel.Tag != tag {
		return Release{}, fmt.Errorf("selfupdate: github returned release %q for tag %q: %w", rel.Tag, tag, ErrIntegrity)
	}
	return rel, nil
}

type githubReleaseJSON struct {
	ID         int64             `json:"id"`
	TagName    string            `json:"tag_name"`
	HTMLURL    string            `json:"html_url"`
	Draft      bool              `json:"draft"`
	Prerelease bool              `json:"prerelease"`
	Immutable  bool              `json:"immutable"`
	Assets     []githubAssetJSON `json:"assets"`
}

type githubAssetJSON struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

func (s *GitHubSource) getRelease(ctx context.Context, rawURL string) (rel Release, err error) {
	resp, err := s.send(ctx, rawURL, gitHubAcceptJSON)
	if err != nil {
		return Release{}, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	// Look at the status before the body: an error body is read only up to
	// ErrorBody, truncated rather than refused, so an oversized 429 still
	// maps to RateLimitError (0003-MADR A8).
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		errBody, rerr := readTruncated(resp.Body, s.limits.ErrorBody)
		if rerr != nil {
			return Release{}, rerr
		}
		return Release{}, s.mapStatus(resp, errBody)
	}
	body, err := readBounded(resp.Body, s.limits.ReleaseJSON)
	if err != nil {
		return Release{}, err
	}
	var raw githubReleaseJSON
	if err := decodeJSON(body, &raw); err != nil {
		return Release{}, err
	}
	rel, err = mapRelease(raw)
	if err != nil {
		return Release{}, err
	}
	if err := validateFetchedRelease(rel); err != nil {
		return Release{}, err
	}
	return rel, nil
}

// mapRelease checks only the structure of every asset. State, size and digest
// are validated for the selected binary and manifest alone, by the Updater,
// so an unrelated extra asset cannot make a release unusable (0003-MADR A1).
func mapRelease(raw githubReleaseJSON) (Release, error) {
	rel := Release{
		ID:         raw.ID,
		Tag:        raw.TagName,
		URL:        raw.HTMLURL,
		Draft:      raw.Draft,
		Prerelease: raw.Prerelease,
		Immutable:  raw.Immutable,
		Assets:     make([]Asset, 0, len(raw.Assets)),
	}
	for _, a := range raw.Assets {
		asset := Asset(a)
		if err := validateAssetStructure(asset); err != nil {
			return Release{}, err
		}
		rel.Assets = append(rel.Assets, asset)
	}
	return rel, nil
}

// validateAssetStructure checks the identity every asset must have: an ID,
// and a name that is a basename without control characters.
func validateAssetStructure(a Asset) error {
	if a.ID <= 0 || a.Name == "" {
		return fmt.Errorf("selfupdate: asset metadata is incomplete")
	}
	// "." and ".." contain no separator but are not names of files
	// (0004-MADR R9).
	if a.Name == "." || a.Name == ".." || strings.ContainsAny(a.Name, `/\`) || strings.IndexFunc(a.Name, unicode.IsControl) >= 0 {
		return fmt.Errorf("selfupdate: asset name %q is not a basename", a.Name)
	}
	return nil
}

func validateFetchedRelease(rel Release) error {
	if rel.ID <= 0 || rel.Tag == "" {
		return fmt.Errorf("selfupdate: release metadata is incomplete")
	}
	if rel.Draft {
		return fmt.Errorf("selfupdate: release %q is a draft", rel.Tag)
	}
	if rel.Prerelease {
		return fmt.Errorf("selfupdate: release %q is a prerelease", rel.Tag)
	}
	if !rel.Immutable {
		return fmt.Errorf("selfupdate: release %q is not immutable: %w", rel.Tag, ErrMutableRelease)
	}
	return nil
}

func assetBelongsToRelease(rel Release, asset Asset) error {
	if len(rel.Assets) == 0 {
		return nil
	}
	for _, a := range rel.Assets {
		if a.ID == asset.ID {
			return nil
		}
	}
	return fmt.Errorf("selfupdate: asset %d is not part of release %q", asset.ID, rel.Tag)
}

// validateAssetMetadata is the full check for an asset about to be used:
// structure, uploaded state, a positive size within maxSize, and digest
// syntax.
func validateAssetMetadata(a Asset, maxSize int64) error {
	if err := validateAssetStructure(a); err != nil {
		return err
	}
	if a.State != AssetStateUploaded {
		return fmt.Errorf("selfupdate: asset %s is not uploaded", a.Name)
	}
	if a.Size <= 0 {
		return fmt.Errorf("selfupdate: asset %s has non-positive size: %w", a.Name, ErrIntegrity)
	}
	if a.Size > maxSize {
		return fmt.Errorf("selfupdate: asset %s size %d exceeds limit %d: %w", a.Name, a.Size, maxSize, ErrIntegrity)
	}
	if a.Digest != "" {
		if _, err := parseGitHubDigest(a.Digest); err != nil {
			return err
		}
	}
	return nil
}

// OpenAsset implements ReleaseSource. The body is fetched from the asset API
// path derived from owner, repository, and asset ID.
func (s *GitHubSource) OpenAsset(ctx context.Context, rel Release, asset Asset) (io.ReadCloser, error) {
	if asset.ID <= 0 {
		return nil, fmt.Errorf("selfupdate: asset id is required")
	}
	if err := assetBelongsToRelease(rel, asset); err != nil {
		return nil, err
	}
	if err := validateAssetMetadata(asset, s.limits.Executable); err != nil {
		return nil, err
	}
	rawURL := s.apiURL("repos", s.repo.Owner, s.repo.Name, "releases", "assets", strconv.FormatInt(asset.ID, 10))
	resp, err := s.send(ctx, rawURL, gitHubAcceptAsset)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, readErr := readTruncated(resp.Body, s.limits.ErrorBody)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, errors.Join(readErr, closeErr)
		}
		statusErr := s.mapStatus(resp, body)
		if closeErr != nil {
			return nil, errors.Join(statusErr, closeErr)
		}
		return nil, statusErr
	}
	return resp.Body, nil
}

func (s *GitHubSource) newRequest(ctx context.Context, method, rawURL, accept string, cred *Credential) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", gitHubAPIVersion)
	req.Header.Set("User-Agent", s.userAgent)
	if cred != nil && sameOrigin(req.URL, s.apiBase) {
		if cred.Header == "" {
			req.Header.Set("Authorization", "Bearer "+string(cred.Value))
		} else {
			req.Header.Set(cred.Header, string(cred.Value))
		}
	}
	return req, nil
}

// send issues one GET with the source's credential attached when the URL
// is on the API origin. A 401 to a provider's credential asks the
// provider once more and retries once with a different credential; this
// happens at most once per source, so a provider that prompts is never
// asked in a loop. The first 2xx to a credentialed request tells the
// Observer (0004-MADR G10).
func (s *GitHubSource) send(ctx context.Context, rawURL, accept string) (*http.Response, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	var cred *Credential
	if sameOrigin(parsed, s.apiBase) {
		if cred, err = s.credential(ctx); err != nil {
			return nil, err
		}
	}
	resp, err := s.do(ctx, rawURL, accept, cred)
	if err != nil {
		return nil, err
	}
	if cred != nil && resp.StatusCode == http.StatusUnauthorized {
		if next := s.refresh(ctx, cred); next != nil {
			drainClose(resp)
			cred = next
			if resp, err = s.do(ctx, rawURL, accept, cred); err != nil {
				return nil, err
			}
		}
	}
	if cred != nil && resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		s.accept(ctx, *cred)
	}
	return resp, nil
}

func (s *GitHubSource) do(ctx context.Context, rawURL, accept string, cred *Credential) (*http.Response, error) {
	req, err := s.newRequest(ctx, http.MethodGet, rawURL, accept, cred)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req)
}

// drainClose discards a refused response before its retry. Nothing depends
// on it, so its errors must not change the outcome.
func drainClose(resp *http.Response) {
	_, cerr := io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	advisory(errors.Join(cerr, resp.Body.Close()))
}

// WithCredentials implements CredentialedSource. The copy shares the
// repository, API base, user agent, explicit and environment tokens,
// observer and limits. Its provider is p (none when p is nil), its
// credential state is new, and its client is its own copy, so redirect
// scrubbing reads the copy's credential.
func (s *GitHubSource) WithCredentials(p CredentialProvider) ReleaseSource {
	client := *s.client
	c := &GitHubSource{
		repo:      s.repo,
		client:    &client,
		apiBase:   s.apiBase,
		userAgent: s.userAgent,
		token:     s.token,
		explicit:  s.explicit,
		envName:   s.envName,
		observer:  s.observer,
		limits:    s.limits,
		now:       s.now,
	}
	if !isNil(p) {
		c.provider = p
	}
	c.client.CheckRedirect = c.checkRedirect
	return c
}

// credential resolves the source's credential once: the explicit Token,
// else the provider, else the environment token. Nil means anonymous.
func (s *GitHubSource) credential(ctx context.Context) (*Credential, error) {
	s.cred.mu.Lock()
	defer s.cred.mu.Unlock()
	if !s.cred.resolved {
		if err := s.resolveLocked(ctx); err != nil {
			return nil, err
		}
		s.cred.resolved = true
	}
	if !s.cred.has {
		return nil, nil
	}
	c := s.cred.cred
	return &c, nil
}

func (s *GitHubSource) resolveLocked(ctx context.Context) error {
	if s.explicit {
		s.cred.cred, s.cred.has = Credential{Value: []byte(s.token), Source: "token"}, true
		return nil
	}
	if s.provider != nil {
		c, err := s.provider.Credential(ctx, CredentialRequest{Origin: s.origin()})
		switch {
		case err == nil:
			if verr := validateCredential(c); verr != nil {
				return verr
			}
			s.cred.cred, s.cred.has, s.cred.fromProvider = c, true, true
			return nil
		case !errors.Is(err, ErrNoCredential):
			return err
		}
	}
	if s.token != "" {
		s.cred.cred, s.cred.has = Credential{Value: []byte(s.token), Source: "env:" + s.envName}, true
	}
	return nil
}

// refresh asks the provider once more after a 401. It returns the new
// credential when it differs from the refused one, and nil otherwise: the
// caller then returns the 401.
func (s *GitHubSource) refresh(ctx context.Context, refused *Credential) *Credential {
	s.cred.mu.Lock()
	defer s.cred.mu.Unlock()
	if !s.cred.fromProvider || s.cred.retried {
		return nil
	}
	s.cred.retried = true
	c, err := s.provider.Credential(ctx, CredentialRequest{
		Origin: s.origin(),
		Cause:  fmt.Errorf("selfupdate: github http %d: credential from %s refused", http.StatusUnauthorized, sanitizeText(refused.Source)),
	})
	if err != nil || validateCredential(c) != nil || sameCredential(c, *refused) {
		return nil
	}
	s.cred.cred = c
	return &c
}

func (s *GitHubSource) accept(ctx context.Context, c Credential) {
	s.cred.mu.Lock()
	first := !s.cred.accepted
	s.cred.accepted = true
	s.cred.mu.Unlock()
	if first && s.observer != nil {
		s.observer.Accepted(ctx, c)
	}
}

func (s *GitHubSource) origin() *url.URL {
	u := *s.apiBase
	return &u
}

func (s *GitHubSource) mapStatus(resp *http.Response, body []byte) error {
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	}
	if resp.StatusCode == http.StatusTooManyRequests || rateLimitedForbidden(resp) {
		return parseRateLimit(resp, s.now)
	}
	diag := sanitizeDiagnostic(string(body), s.limits.ErrorBody)
	return fmt.Errorf("selfupdate: github http %d: %s", resp.StatusCode, diag)
}

// maxRetryAfterSeconds is the largest Retry-After, in seconds, that fits in a
// time.Duration.
const maxRetryAfterSeconds = math.MaxInt64 / int64(time.Second)

func rateLimitedForbidden(resp *http.Response) bool {
	if resp.StatusCode != http.StatusForbidden {
		return false
	}
	if resp.Header.Get("Retry-After") != "" {
		return true
	}
	return resp.Header.Get("X-RateLimit-Remaining") == "0"
}

func parseRateLimit(resp *http.Response, now func() time.Time) error {
	if now == nil {
		now = timeNow
	}
	err := &RateLimitError{StatusCode: resp.StatusCode}
	if v := strings.TrimSpace(resp.Header.Get("X-RateLimit-Remaining")); v != "" {
		if n, perr := strconv.ParseInt(v, 10, 64); perr == nil {
			err.Remaining = n
		}
	}
	if v := strings.TrimSpace(resp.Header.Get("X-RateLimit-Reset")); v != "" {
		if n, perr := strconv.ParseInt(v, 10, 64); perr == nil {
			err.Reset = time.Unix(n, 0).UTC()
		}
	}
	if v := strings.TrimSpace(resp.Header.Get("Retry-After")); v != "" {
		if secs, perr := strconv.ParseInt(v, 10, 64); perr == nil {
			// Seconds beyond what a Duration holds would overflow into a
			// negative value; treat them as malformed (0003-MADR A7).
			if secs > 0 && secs <= maxRetryAfterSeconds {
				err.RetryAfter = time.Duration(secs) * time.Second
			}
		} else if when, perr := http.ParseTime(v); perr == nil {
			d := when.Sub(now())
			if d > 0 {
				err.RetryAfter = d
			}
		}
	}
	return err
}
