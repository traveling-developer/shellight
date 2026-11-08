// Package gateway polls the active gateway of the openshell CLI through
// NVIDIA's OpenShell Go SDK.
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
	pollBackoff  = 30 * time.Second // after an error
	fetchTimeout = 5 * time.Second
)

// Status is the result of one poll round.
type Status struct {
	Name     string // active gateway; empty if the CLI config cannot be read
	Endpoint string // as in the CLI config, e.g. https://127.0.0.1:17690
	Info     Info
	Err      error // reading the config, connecting or fetching failed
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
}

// Poll fetches from the active gateway of the openshell CLI every 5 s, or
// 30 s after an error, until ctx is done. It passes the first result and
// every change to update, so an open menu is not rebuilt for nothing.
func Poll(ctx context.Context, update func(Status)) {
	var last *Status
	for {
		s := poll(ctx)
		if last == nil || !s.equal(*last) {
			update(s)
			last = &s
		}
		wait := pollEvery
		if s.Err != nil {
			wait = pollBackoff
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// equal reports whether s and o would show the same.
func (s Status) equal(o Status) bool {
	return s.Name == o.Name && s.Endpoint == o.Endpoint &&
		errText(s.Err) == errText(o.Err) && reflect.DeepEqual(s.Info, o.Info)
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
	s := Status{Name: cfg.Name, Endpoint: cfg.Endpoint}
	sdk, err := dial(cfg)
	if err != nil {
		s.Err = err
		return s
	}
	defer sdk.Close()

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	s.Info, s.Err = fetch(ctx, sdk)
	return s
}

// loadConfig reads the active gateway from the openshell CLI config.
func loadConfig() (*sdkgw.Config, error) {
	cfg, err := sdkgw.LoadConfig("")
	if err != nil {
		return nil, fmt.Errorf("gateway config: %w", err)
	}
	if cfg.AuthMode != sdkgw.AuthModeMTLS {
		return nil, fmt.Errorf("gateway %q: auth mode %q not supported (only mtls)", cfg.Name, cfg.AuthMode)
	}
	return cfg, nil
}

// dial sets up an mTLS client for cfg. The connection opens on the first
// call.
func dial(cfg *sdkgw.Config) (v1.ClientInterface, error) {
	// The CLI keeps the client mTLS bundle next to metadata.json; the SDK
	// does not load it for auth mode mtls.
	mtls := filepath.Join(cfg.Dir, "mtls")
	sdk, err := v1.NewClient(v1.Config{
		Address: cfg.Endpoint,
		TLS: &v1.TLSConfig{
			CAFile:   filepath.Join(mtls, "ca.crt"),
			CertFile: filepath.Join(mtls, "tls.crt"),
			KeyFile:  filepath.Join(mtls, "tls.key"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("gateway %q: %w", cfg.Name, err)
	}
	return sdk, nil
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
