package selfupdate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Pluggable credentials for release sources
// (docs/decisions/0004-MADR-evolve-selfupdate-api-and-tui-support.md G10).

// Credential is one HTTP credential. An empty Header means
// "Authorization: Bearer <Value>"; otherwise "Header: Value" is sent as
// given (0004-MADR amendment A6). Source labels where it came from, for
// diagnostics; it never holds the secret.
type Credential struct {
	// Header is the HTTP header name, or empty for a bearer token.
	Header string
	// Value is the secret. Callers must not modify it after returning it.
	Value []byte
	// Source is a label such as "env:GH_TOKEN".
	Source string
}

// CredentialRequest describes the request that needs a credential.
type CredentialRequest struct {
	// Origin is a copy of the API origin the credential will be sent to.
	Origin *url.URL
	// Cause is non-nil when a credential already sent was refused with 401.
	Cause error
	// Interactive reports whether the provider may prompt. Phase 1 sources
	// always set it false.
	Interactive bool
}

// CredentialProvider supplies a credential. Returning ErrNoCredential
// passes to the next provider in a chain, or to an anonymous request.
type CredentialProvider interface {
	Credential(context.Context, CredentialRequest) (Credential, error)
}

// CredentialObserver learns that a credential was accepted: a request that
// carried it succeeded. A program uses it to save a token only once it is
// known to work.
type CredentialObserver interface {
	Accepted(context.Context, Credential)
}

// ErrNoCredential reports that a provider has no credential to offer.
var ErrNoCredential = errors.New("selfupdate: no credential")

type chainCredentials []CredentialProvider

// CredentialedSource is a ReleaseSource that can take a per-run credential
// provider. WithCredentials returns a copy whose provider is p, with fresh
// credential state, and leaves the receiver unchanged. RunWith uses it for
// WithCredentials (0004-MADR amendment B1).
type CredentialedSource interface {
	ReleaseSource
	WithCredentials(p CredentialProvider) ReleaseSource
}

// ChainCredentials asks each provider in order. The first result other than
// ErrNoCredential wins, credential or error. Nil and typed-nil providers are
// dropped.
func ChainCredentials(providers ...CredentialProvider) CredentialProvider {
	kept := make(chainCredentials, 0, len(providers))
	for _, p := range providers {
		if !isNil(p) {
			kept = append(kept, p)
		}
	}
	return kept
}

func (c chainCredentials) Credential(ctx context.Context, req CredentialRequest) (Credential, error) {
	for _, p := range c {
		cred, err := p.Credential(ctx, req)
		if errors.Is(err, ErrNoCredential) {
			continue
		}
		return cred, err
	}
	return Credential{}, ErrNoCredential
}

type envCredential struct {
	header string
	names  []string
}

// EnvCredential returns the first non-empty environment variable among
// names as a credential for header, with Source "env:<NAME>". With none
// set it returns ErrNoCredential.
func EnvCredential(header string, names ...string) CredentialProvider {
	return envCredential{header: header, names: append([]string(nil), names...)}
}

func (e envCredential) Credential(context.Context, CredentialRequest) (Credential, error) {
	for _, name := range e.names {
		if v := lookupEnv(name); v != "" {
			return Credential{Header: e.header, Value: []byte(v), Source: "env:" + name}, nil
		}
	}
	return Credential{}, ErrNoCredential
}

// validateCredential refuses a header name that is not an RFC 7230 token
// and a value that could split or truncate the header. Errors never
// include the value.
func validateCredential(c Credential) error {
	if c.Header != "" {
		for i := 0; i < len(c.Header); i++ {
			if !isTokenChar(c.Header[i]) {
				return fmt.Errorf("selfupdate: invalid credential from %s", sanitizeText(c.Source))
			}
		}
	}
	if len(c.Value) == 0 || strings.ContainsAny(string(c.Value), "\r\n\x00") {
		return fmt.Errorf("selfupdate: invalid credential from %s", sanitizeText(c.Source))
	}
	return nil
}

func isTokenChar(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	}
	return strings.IndexByte("!#$%&'*+-.^_`|~", b) >= 0
}

func sameCredential(a, b Credential) bool {
	return a.Header == b.Header && bytes.Equal(a.Value, b.Value)
}
