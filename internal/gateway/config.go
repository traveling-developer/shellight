// Package gateway reads the active OpenShell gateway from the openshell CLI
// config, so shellight talks to the same gateway as the CLI.
//
// Layout (openshell 0.1.7):
//
//	<dir>/active_gateway                     gateway name, no trailing newline
//	<dir>/gateways/<name>/metadata.json      {gateway_endpoint, auth_mode, ...}
//
// <dir> is $XDG_CONFIG_HOME/openshell, else ~/.config/openshell.
package gateway

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Gateway is one gateway entry of the CLI config.
type Gateway struct {
	Name     string // e.g. "openshell" or "shellight-test"
	Endpoint string // https://host[:port], as in the CLI config
	Addr     string // host:port for the gRPC dial; port 443 if the endpoint has none
}

// Dir returns the openshell CLI config dir: $XDG_CONFIG_HOME/openshell if
// XDG_CONFIG_HOME is an absolute path, else ~/.config/openshell.
func Dir() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(x) {
		return filepath.Join(x, "openshell"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	return filepath.Join(home, ".config", "openshell"), nil
}

// Load reads the active gateway from the config dir.
func Load(dir string) (Gateway, error) {
	b, err := os.ReadFile(filepath.Join(dir, "active_gateway"))
	if err != nil {
		return Gateway{}, fmt.Errorf("active gateway: %w", err)
	}
	name := strings.TrimSpace(string(b))
	if name == "" || filepath.Base(name) != name {
		return Gateway{}, fmt.Errorf("active gateway: bad name %q", name)
	}

	b, err = os.ReadFile(filepath.Join(dir, "gateways", name, "metadata.json"))
	if err != nil {
		return Gateway{}, fmt.Errorf("gateway %q: %w", name, err)
	}
	var m struct {
		Endpoint string `json:"gateway_endpoint"`
		AuthMode string `json:"auth_mode"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return Gateway{}, fmt.Errorf("gateway %q: metadata: %w", name, err)
	}
	if m.AuthMode != "mtls" {
		return Gateway{}, fmt.Errorf("gateway %q: auth mode %q not supported (only mtls)", name, m.AuthMode)
	}
	u, err := url.Parse(m.Endpoint)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return Gateway{}, fmt.Errorf("gateway %q: endpoint %q is not https://host[:port]", name, m.Endpoint)
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}

	return Gateway{
		Name:     name,
		Endpoint: m.Endpoint,
		Addr:     net.JoinHostPort(u.Hostname(), port),
	}, nil
}
