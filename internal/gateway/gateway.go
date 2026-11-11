// Package gateway polls the active gateway of the openshell CLI through
// NVIDIA's OpenShell Go SDK, and the CLI's port forwards through the CLI.
package gateway

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"time"

	v1 "github.com/NVIDIA/OpenShell/sdk/go/openshell/v1"
	sdkgw "github.com/NVIDIA/OpenShell/sdk/go/openshell/v1/gateway"
)

const (
	pollEvery    = 5 * time.Second
	pollBackoff  = 30 * time.Second // after quickRetries failed rounds
	quickRetries = 6                // failed rounds retried at pollEvery
	fetchTimeout = 5 * time.Second
)

// Status is the result of one poll round.
type Status struct {
	Name     string // active gateway; empty if the CLI config cannot be read
	Endpoint string // as in the CLI config, e.g. https://127.0.0.1:17690
	AuthMode string // as in the CLI config: mtls, plaintext, none, ...
	Info     Info
	Err      error // reading the config, connecting or fetching failed
	// ForwardErr is set if listing the port forwards failed; the gateway
	// status is still valid then.
	ForwardErr error
	// ProposalErr is set if reading the policy proposals failed; the gateway
	// status is still valid then, but no sandbox has proposals.
	ProposalErr error
	// OtherForwards are running forwards to sandboxes the gateway does not
	// list, e.g. of another gateway: the CLI keeps the forwards of all
	// gateways together. Sorted by workspace, sandbox, then port.
	OtherForwards []Forward
	// LastSeen is when the last round succeeded; set only if Err is set,
	// zero if no round has succeeded since start.
	LastSeen time.Time
	// Retry is the wait until the next round; set only if Err is set.
	Retry time.Duration
}

// Info is what shellight shows about the gateway.
type Info struct {
	Version   string
	Drivers   []string  // compute drivers
	Sandboxes []Sandbox // sorted by workspace, then name
}

// Sandbox is one sandbox of any workspace.
type Sandbox struct {
	Name          string
	Workspace     string
	Phase         string // Provisioning, Ready, Error, ...
	Reason        string // why the sandbox is in phase Error, else empty
	Created       time.Time
	PolicyVersion uint32
	Forwards      []Forward        // running ones, sorted by port
	Proposals     []PolicyProposal // pending ones, sorted by ID
}

// Poll fetches from the active gateway of the openshell CLI every 5 s
// until ctx is done; after quickRetries failed rounds in a row it backs off
// to 30 s. It passes the first result and every change to update, so an
// open menu is not rebuilt for nothing.
func Poll(ctx context.Context, update func(Status)) {
	var last *Status
	var h history
	for {
		s := poll(ctx)
		wait := h.note(&s, time.Now())
		if last == nil || !s.equal(*last) {
			update(s)
			last = &s
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// history is what Poll remembers of the rounds before, for the active
// gateway only: a switch to another gateway starts it over.
type history struct {
	name, endpoint string    // of the active gateway
	lastSeen       time.Time // last round that succeeded, zero if none
	failures       int       // failed rounds in a row
}

// note records round s at now, sets s.LastSeen and s.Retry if s failed,
// and returns the wait until the next round.
func (h *history) note(s *Status, now time.Time) time.Duration {
	if s.Name != h.name || s.Endpoint != h.endpoint {
		*h = history{name: s.Name, endpoint: s.Endpoint}
	}
	if s.Err == nil {
		h.failures, h.lastSeen = 0, now
		return pollEvery
	}
	h.failures++
	s.LastSeen, s.Retry = h.lastSeen, retryAfter(h.failures)
	return s.Retry
}

// retryAfter is the wait after the given number of failed rounds in a row.
func retryAfter(failures int) time.Duration {
	if failures <= quickRetries {
		return pollEvery
	}
	return pollBackoff
}

// equal reports whether s and o would show the same.
func (s Status) equal(o Status) bool {
	return s.Name == o.Name && s.Endpoint == o.Endpoint && s.AuthMode == o.AuthMode &&
		errText(s.Err) == errText(o.Err) && errText(s.ForwardErr) == errText(o.ForwardErr) &&
		errText(s.ProposalErr) == errText(o.ProposalErr) &&
		reflect.DeepEqual(s.Info, o.Info) && reflect.DeepEqual(s.OtherForwards, o.OtherForwards) &&
		s.LastSeen.Equal(o.LastSeen) && s.Retry == o.Retry
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// poll runs one round. It reads the CLI config and connects anew each time,
// so it follows a gateway switch and picks up new certificates.
func poll(ctx context.Context) Status {
	cfg, err := loadConfig()
	if err != nil {
		return Status{Err: err}
	}
	s := Status{Name: cfg.Name, Endpoint: cfg.Endpoint, AuthMode: string(cfg.AuthMode)}
	conf, err := clientConfig(cfg)
	if err != nil {
		s.Err = err
		return s
	}
	// The connection opens on the first call.
	sdk, err := v1.NewClient(conf)
	if err != nil {
		s.Err = fmt.Errorf("gateway %q: %w", cfg.Name, err)
		return s
	}
	defer sdk.Close()

	fctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	s.Info, s.Err = fetch(fctx, sdk)
	if s.Err != nil {
		return s
	}

	// Proposals and forwards only show under their sandbox, so skip them
	// when the gateway is unreachable.
	pctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	s.ProposalErr = attachProposals(pctx, sdk, s.Info.Sandboxes)

	lctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	fwds, err := listForwards(lctx)
	if err != nil {
		s.ForwardErr = err
		return s
	}
	s.OtherForwards = attachForwards(s.Info.Sandboxes, fwds)
	return s
}

// loadConfig reads the active gateway from the openshell CLI config.
func loadConfig() (*sdkgw.Config, error) {
	cfg, err := sdkgw.LoadConfig("")
	if err != nil {
		return nil, fmt.Errorf("gateway config: %w", err)
	}
	return cfg, nil
}

// UnsupportedAuthModeError says that the active gateway uses an auth mode
// shellight does not support: only those of a local gateway are (ADR 0001).
type UnsupportedAuthModeError struct {
	Mode string // oidc, cloudflare_jwt
}

func (e *UnsupportedAuthModeError) Error() string {
	return fmt.Sprintf("shellight supports the auth modes mtls, plaintext and none, not %s", e.Mode)
}

// clientConfig maps the auth mode of cfg to an SDK client config. mtls
// uses the CLI's client bundle. plaintext and none connect without client
// credentials; the CLI registers them for http:// endpoints.
func clientConfig(cfg *sdkgw.Config) (v1.Config, error) {
	conf := v1.Config{Address: cfg.Endpoint}
	switch cfg.AuthMode {
	case sdkgw.AuthModeMTLS:
		// The CLI keeps the client mTLS bundle next to metadata.json; the
		// SDK does not load it for auth mode mtls.
		mtls := filepath.Join(cfg.Dir, "mtls")
		conf.TLS = &v1.TLSConfig{
			CAFile:   filepath.Join(mtls, "ca.crt"),
			CertFile: filepath.Join(mtls, "tls.crt"),
			KeyFile:  filepath.Join(mtls, "tls.key"),
		}
	case sdkgw.AuthModePlaintext, sdkgw.AuthModeNone:
	default:
		return v1.Config{}, &UnsupportedAuthModeError{Mode: string(cfg.AuthMode)}
	}
	return conf, nil
}

// fetch reads the gateway info and the sandboxes of all workspaces.
func fetch(ctx context.Context, sdk v1.ClientInterface) (Info, error) {
	gi, err := sdk.Health().GetGatewayInfo(ctx)
	if err != nil {
		return Info{}, fmt.Errorf("gateway info: %w", err)
	}
	sbs, err := sdk.Sandboxes().ListAll(ctx, "", v1.ListOptions{AllWorkspaces: true})
	if err != nil {
		return Info{}, fmt.Errorf("sandboxes: %w", err)
	}

	info := Info{Version: gi.Version}
	for _, d := range gi.ComputeDrivers {
		info.Drivers = append(info.Drivers, d.Name)
	}
	for _, s := range sbs {
		info.Sandboxes = append(info.Sandboxes, Sandbox{
			Name:          s.Name,
			Workspace:     s.Workspace,
			Phase:         string(s.Status.Phase),
			Reason:        reason(s),
			Created:       s.CreatedAt,
			PolicyVersion: s.Status.CurrentPolicyVersion,
		})
	}
	slices.SortFunc(info.Sandboxes, func(a, b Sandbox) int {
		return cmp.Or(cmp.Compare(a.Workspace, b.Workspace), cmp.Compare(a.Name, b.Name))
	})
	return info, nil
}

// reason returns why a sandbox is in phase Error: the message (else the
// reason) of the first condition that is not "True".
func reason(s *v1.Sandbox) string {
	if s.Status.Phase != v1.SandboxError {
		return ""
	}
	for _, c := range s.Status.Conditions {
		if c.Status == "True" {
			continue
		}
		if c.Message != "" {
			return c.Message
		}
		return c.Reason
	}
	return ""
}
