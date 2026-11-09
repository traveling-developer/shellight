package gateway

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	v1 "github.com/NVIDIA/OpenShell/sdk/go/openshell/v1"
	"github.com/NVIDIA/OpenShell/sdk/go/openshell/v1/fake"
	sdkgw "github.com/NVIDIA/OpenShell/sdk/go/openshell/v1/gateway"
	"github.com/NVIDIA/OpenShell/sdk/go/openshell/v1/types"
)

func TestFetch(t *testing.T) {
	created := time.Date(2026, 11, 8, 14, 20, 0, 0, time.UTC)
	f := fake.NewClient(fake.WithGatewayInfo(&types.GatewayInfo{
		Version:        "0.1.7",
		ComputeDrivers: []types.ComputeDriverInfo{{Name: "podman"}},
	}))
	f.AddSandbox("default", &types.Sandbox{
		Name: "ok", Workspace: "default", CreatedAt: created,
		Status: types.SandboxStatus{Phase: types.SandboxReady, CurrentPolicyVersion: 2},
	})
	// As the test gateway reports a sandbox whose main process failed.
	f.AddSandbox("team", &types.Sandbox{
		Name: "broken", Workspace: "team", CreatedAt: created,
		Status: types.SandboxStatus{
			Phase: types.SandboxError,
			Conditions: []types.SandboxCondition{
				{Type: "Ready", Status: "False", Reason: "MainProcessFailed", Message: "Canonical main process exited with status 3"},
				{Type: "ConfigurationReady", Status: "True", Reason: "ConfigurationAccepted"},
			},
			CurrentPolicyVersion: 1,
		},
	})
	f.AddSandbox("default", &types.Sandbox{
		Name: "another", Workspace: "default", CreatedAt: created,
		Status: types.SandboxStatus{Phase: types.SandboxProvisioning},
	})
	info, err := fetch(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	want := Info{
		Version: "0.1.7",
		Drivers: []string{"podman"},
		Sandboxes: []Sandbox{ // sorted by workspace, then name
			{Name: "another", Workspace: "default", Phase: "Provisioning", Created: created},
			{Name: "ok", Workspace: "default", Phase: "Ready", Created: created, PolicyVersion: 2},
			{Name: "broken", Workspace: "team", Phase: "Error", Created: created, PolicyVersion: 1,
				Reason: "Canonical main process exited with status 3"},
		},
	}
	if !reflect.DeepEqual(info, want) {
		t.Errorf("fetch =\n%+v\nwant\n%+v", info, want)
	}
}

func TestFetchError(t *testing.T) {
	f := fake.NewClient()
	f.Close() // a closed fake fails every call as unavailable
	if _, err := fetch(context.Background(), f); err == nil {
		t.Error("fetch: no error")
	}
}

func TestStatusEqual(t *testing.T) {
	base := Status{Name: "gw", Info: Info{Version: "0.1.7", Sandboxes: []Sandbox{{Name: "a", Phase: "Ready"}}}}
	same := Status{Name: "gw", Info: Info{Version: "0.1.7", Sandboxes: []Sandbox{{Name: "a", Phase: "Ready"}}}}
	if !base.equal(same) {
		t.Error("equal statuses differ")
	}
	phase := same
	phase.Info.Sandboxes = []Sandbox{{Name: "a", Phase: "Error"}}
	if base.equal(phase) {
		t.Error("phase change not seen")
	}
	down := Status{Name: "gw", Err: errors.New("down")}
	if base.equal(down) || !down.equal(Status{Name: "gw", Err: errors.New("down")}) {
		t.Error("errors compared wrongly")
	}
	fwdErr := same
	fwdErr.ForwardErr = errors.New("openshell CLI not found")
	if base.equal(fwdErr) {
		t.Error("forward error not seen")
	}
	proposalErr := same
	proposalErr.ProposalErr = errors.New("policy draft of default/a: down")
	if base.equal(proposalErr) {
		t.Error("proposal error not seen")
	}
}

func TestReason(t *testing.T) {
	notReady := []types.SandboxCondition{{Type: "Ready", Status: "False", Reason: "MainProcessFailed"}}
	tests := []struct {
		name  string
		phase types.SandboxPhase
		conds []types.SandboxCondition
		want  string
	}{
		{"not in error", types.SandboxProvisioning, notReady, ""},
		{"reason without message", types.SandboxError, notReady, "MainProcessFailed"},
		{"all conditions true", types.SandboxError, []types.SandboxCondition{{Type: "Ready", Status: "True"}}, ""},
		{"no conditions", types.SandboxError, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &types.Sandbox{Status: types.SandboxStatus{Phase: tt.phase, Conditions: tt.conds}}
			if got := reason(s); got != tt.want {
				t.Errorf("reason = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientConfig(t *testing.T) {
	mtls := &v1.TLSConfig{
		CAFile:   "/gw/mtls/ca.crt",
		CertFile: "/gw/mtls/tls.crt",
		KeyFile:  "/gw/mtls/tls.key",
	}
	tests := []struct {
		mode     sdkgw.AuthMode
		endpoint string
		want     v1.Config // zero if unsupported
	}{
		{sdkgw.AuthModeMTLS, "https://127.0.0.1:17690", v1.Config{Address: "https://127.0.0.1:17690", TLS: mtls}},
		{sdkgw.AuthModePlaintext, "http://127.0.0.1:17691", v1.Config{Address: "http://127.0.0.1:17691"}},
		{sdkgw.AuthModeNone, "http://127.0.0.1:8080", v1.Config{Address: "http://127.0.0.1:8080"}},
		{sdkgw.AuthModeOIDC, "https://gw.example:443", v1.Config{}},
		{sdkgw.AuthModeCloudflareJWT, "https://gw.example:443", v1.Config{}},
	}
	for _, tt := range tests {
		t.Run(string(tt.mode)+" "+tt.endpoint, func(t *testing.T) {
			got, err := clientConfig(&sdkgw.Config{Name: "gw", Endpoint: tt.endpoint, AuthMode: tt.mode, Dir: "/gw"})
			var unsupported *UnsupportedAuthModeError
			if tt.want == (v1.Config{}) {
				if !errors.As(err, &unsupported) || unsupported.Mode != string(tt.mode) {
					t.Errorf("err = %v, want unsupported auth mode %s", err, tt.mode)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("config = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// writeConfig writes a CLI config with one active gateway into a temp dir
// and points XDG_CONFIG_HOME at it.
func writeConfig(t *testing.T, metadata string) {
	t.Helper()
	xdg := t.TempDir()
	dir := filepath.Join(xdg, "openshell")
	gw := filepath.Join(dir, "gateways", "gw")
	if err := os.MkdirAll(gw, 0o700); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(dir, "active_gateway"): "gw",
		filepath.Join(gw, "metadata.json"):   metadata,
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)
}

func TestPollErrors(t *testing.T) {
	t.Run("no CLI config", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		if s := poll(context.Background()); s.Err == nil || s.Name != "" {
			t.Errorf("poll = %+v", s)
		}
	})
	t.Run("auth mode not supported", func(t *testing.T) {
		writeConfig(t, `{"gateway_endpoint":"https://gw.example:443","auth_mode":"oidc"}`)
		s := poll(context.Background())
		var unsupported *UnsupportedAuthModeError
		if !errors.As(s.Err, &unsupported) || s.Name != "gw" {
			t.Errorf("poll = %+v", s)
		}
	})
	t.Run("mTLS bundle missing", func(t *testing.T) {
		writeConfig(t, `{"gateway_endpoint":"https://127.0.0.1:17690","auth_mode":"mtls"}`)
		if s := poll(context.Background()); s.Err == nil || s.Name != "gw" {
			t.Errorf("poll = %+v", s)
		}
	})
}

// TestLive polls the mTLS test gateway (scripts/test-gateway.sh up):
//
//	SHELLIGHT_LIVE=1 go test ./internal/gateway -run Live -v
func TestLive(t *testing.T) {
	s := livePoll(t, "test-gateway", "shellight-test")
	if !hasProposal(s.Info.Sandboxes, "fwtest", "ports.ubuntu.com:80") {
		t.Error("sandbox fwtest has no pending policy proposal for ports.ubuntu.com:80")
	}
}

// TestLivePlaintext polls the test gateway without TLS (auth mode
// plaintext), which scripts/test-gateway.sh up starts next to the mTLS one.
func TestLivePlaintext(t *testing.T) {
	livePoll(t, "test-gateway-plaintext", "shellight-test-plaintext")
}

// livePoll runs one poll round with the CLI config of the test gateway in
// .dev/<dir> and checks that it reached the gateway named name.
func livePoll(t *testing.T, dir, name string) Status {
	t.Helper()
	if os.Getenv("SHELLIGHT_LIVE") == "" {
		t.Skip("set SHELLIGHT_LIVE=1 to run against the test gateway")
	}
	xdg, err := filepath.Abs(filepath.Join("../../.dev", dir, "cli-config"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)

	s := poll(context.Background())
	if s.Err != nil {
		t.Fatal(s.Err)
	}
	if s.Name != name || s.Info.Version == "" || len(s.Info.Drivers) == 0 {
		t.Errorf("status = %+v", s)
	}
	if s.ForwardErr != nil {
		t.Error(s.ForwardErr)
	}
	if s.ProposalErr != nil {
		t.Error(s.ProposalErr)
	}
	t.Logf("%+v", s)
	return s
}

// hasProposal reports whether the sandbox named sandbox has a policy
// proposal for endpoint.
func hasProposal(sbs []Sandbox, sandbox, endpoint string) bool {
	for _, sb := range sbs {
		if sb.Name != sandbox {
			continue
		}
		for _, p := range sb.Proposals {
			if slices.Contains(p.Endpoints, endpoint) {
				return true
			}
		}
	}
	return false
}
