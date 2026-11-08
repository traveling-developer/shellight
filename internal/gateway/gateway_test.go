package gateway

import (
	"context"
	"os"
	"path/filepath"
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
	c := &Client{sdk: f}

	info, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "0.1.7" || len(info.Drivers) != 1 || info.Drivers[0] != "podman" {
		t.Errorf("gateway = %q %q", info.Version, info.Drivers)
	}

	want := map[string]Sandbox{
		"ok": {Name: "ok", Workspace: "default", Phase: "Ready", Created: created, PolicyVersion: 2},
		"broken": {Name: "broken", Workspace: "team", Phase: "Error", Created: created, PolicyVersion: 1,
			Reason: "Canonical main process exited with status 3"},
	}
	if len(info.Sandboxes) != len(want) {
		t.Fatalf("sandboxes = %+v", info.Sandboxes)
	}
	for _, s := range info.Sandboxes { // the fake lists in random order
		if s != want[s.Name] {
			t.Errorf("sandbox = %+v, want %+v", s, want[s.Name])
		}
	}
}

func TestFetchError(t *testing.T) {
	f := fake.NewClient()
	f.Close() // a closed fake fails every call as unavailable
	c := &Client{sdk: f}
	if _, err := c.Fetch(context.Background()); err == nil {
		t.Error("Fetch: no error")
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

// TestLive fetches from the test gateway (scripts/test-gateway.sh up):
//
//	SHELLIGHT_LIVE=1 go test ./internal/gateway -run Live -v
func TestLive(t *testing.T) {
	if os.Getenv("SHELLIGHT_LIVE") == "" {
		t.Skip("set SHELLIGHT_LIVE=1 to run against the test gateway")
	}
	cfg, err := filepath.Abs("../../.dev/test-gateway/cli-config")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", cfg)

	c, err := Dial()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.Name != "shellight-test" {
		t.Errorf("Name = %q", c.Name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, err := c.Fetch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.Version == "" || len(info.Drivers) == 0 {
		t.Errorf("info = %+v", info)
	}
	t.Logf("%+v", info)
}
