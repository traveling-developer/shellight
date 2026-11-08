package gateway

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/NVIDIA/OpenShell/sdk/go/openshell/v1/fake"
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
	denialErr := same
	denialErr.DenialErr = errors.New("policy draft of default/a: down")
	if base.equal(denialErr) {
		t.Error("denial error not seen")
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
	t.Run("auth mode not mtls", func(t *testing.T) {
		writeConfig(t, `{"gateway_endpoint":"http://127.0.0.1:17690","auth_mode":"plaintext"}`)
		if s := poll(context.Background()); s.Err == nil {
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

// TestLive polls the test gateway (scripts/test-gateway.sh up):
//
//	SHELLIGHT_LIVE=1 go test ./internal/gateway -run Live -v
func TestLive(t *testing.T) {
	if os.Getenv("SHELLIGHT_LIVE") == "" {
		t.Skip("set SHELLIGHT_LIVE=1 to run against the test gateway")
	}
	xdg, err := filepath.Abs("../../.dev/test-gateway/cli-config")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)

	s := poll(context.Background())
	if s.Err != nil {
		t.Fatal(s.Err)
	}
	if s.Name != "shellight-test" || s.Info.Version == "" || len(s.Info.Drivers) == 0 {
		t.Errorf("status = %+v", s)
	}
	if s.ForwardErr != nil {
		t.Error(s.ForwardErr)
	}
	if s.DenialErr != nil {
		t.Error(s.DenialErr)
	}
	t.Logf("%+v", s)
}
