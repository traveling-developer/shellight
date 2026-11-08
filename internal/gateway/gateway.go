// Package gateway talks to the active gateway of the openshell CLI through
// NVIDIA's OpenShell Go SDK and fetches what shellight shows about it.
package gateway

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	v1 "github.com/NVIDIA/OpenShell/sdk/go/openshell/v1"
	sdkgw "github.com/NVIDIA/OpenShell/sdk/go/openshell/v1/gateway"
)

// Client is a client for the active gateway of the openshell CLI.
type Client struct {
	Name     string // gateway name, e.g. "openshell" or "shellight-test"
	Endpoint string // as in the CLI config, e.g. https://127.0.0.1:17690

	sdk v1.ClientInterface
}

// Info is what shellight shows about the gateway.
type Info struct {
	Version   string
	Drivers   []string // compute drivers
	Sandboxes []Sandbox
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

// Dial reads the active gateway from the openshell CLI config and sets up
// an mTLS client for it. The connection opens on the first call.
func Dial() (*Client, error) {
	cfg, err := sdkgw.LoadConfig("")
	if err != nil {
		return nil, fmt.Errorf("gateway config: %w", err)
	}
	if cfg.AuthMode != sdkgw.AuthModeMTLS {
		return nil, fmt.Errorf("gateway %q: auth mode %q not supported (only mtls)", cfg.Name, cfg.AuthMode)
	}
	// The CLI keeps the client mTLS bundle next to metadata.json.
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
	return &Client{Name: cfg.Name, Endpoint: cfg.Endpoint, sdk: sdk}, nil
}

// Close closes the connection.
func (c *Client) Close() error { return c.sdk.Close() }

// Fetch reads the gateway info and the sandboxes of all workspaces. An error
// means the gateway is not reachable or refused the call.
func (c *Client) Fetch(ctx context.Context) (Info, error) {
	gi, err := c.sdk.Health().GetGatewayInfo(ctx)
	if err != nil {
		return Info{}, fmt.Errorf("gateway info: %w", err)
	}
	sbs, err := c.sdk.Sandboxes().ListAll(ctx, "", v1.ListOptions{AllWorkspaces: true})
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
