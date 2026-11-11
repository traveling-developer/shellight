package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/traveling-developer/shellight/internal/gateway"
	"github.com/traveling-developer/shellight/internal/trayicon"
)

func status(phases ...string) gateway.Status {
	s := gateway.Status{Name: "gw"}
	for _, p := range phases {
		s.Info.Sandboxes = append(s.Info.Sandboxes, gateway.Sandbox{Name: p, Phase: p})
	}
	return s
}

func TestDot(t *testing.T) {
	tests := []struct {
		name string
		s    gateway.Status
		want trayicon.RGB
	}{
		{"unreachable", gateway.Status{Name: "gw", Err: errors.New("down")}, errorDot},
		{"no sandboxes", status(), okDot},
		{"all settled", status("Ready", "Stopped", "Completed", "Suspended"), okDot},
		{"one between phases", status("Ready", "Provisioning"), busyDot},
		{"unknown phase", status("Unknown"), busyDot},
		{"error wins", status("Provisioning", "Error"), errorDot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dot(tt.s); got != tt.want {
				t.Errorf("dot = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// labels returns the labels of m's items; "-" for a separator.
func labels(m *application.Menu) []string {
	var l []string
	for i := 0; m.ItemAt(i) != nil; i++ {
		if it := m.ItemAt(i); it.IsSeparator() {
			l = append(l, "-")
		} else {
			l = append(l, it.Label())
		}
	}
	return l
}

// noOpen is the sandbox click handler of the menu tests.
func noOpen(string, string) {}

func TestAddStatus(t *testing.T) {
	s := gateway.Status{Name: "gw", AuthMode: "mtls", Info: gateway.Info{
		Version: "0.1.7",
		Drivers: []string{"podman"},
		Sandboxes: []gateway.Sandbox{
			{Name: "probe", Workspace: "default", Phase: "Error", PolicyVersion: 1,
				Reason: "Canonical main process exited with status 3"},
			{Name: "web", Workspace: "default", Phase: "Ready", Forwards: []gateway.Forward{
				{Workspace: "default", Sandbox: "web", Bind: "127.0.0.1", Port: 8080},
				{Workspace: "default", Sandbox: "web", Bind: "192.168.1.20", Port: 9000},
			}},
			{Name: "api", Workspace: "team", Phase: "Provisioning", Forwards: []gateway.Forward{
				{Workspace: "team", Sandbox: "api", Bind: "::", Port: 3000},
			}},
		},
	}, OtherForwards: []gateway.Forward{{Workspace: "default", Sandbox: "elsewhere", Bind: "::1", Port: 4096}}}
	m := application.NewMenu()
	addStatus(m, s, time.Now(), false, noOpen)
	want := []string{
		"Gateway: gw · mTLS", "-",
		"Sandboxes", "probe — Error", "web — Ready", "api — Provisioning", "-",
		"Port forwards", "localhost:8080 — web", "192.168.1.20:9000 — web", "localhost:3000 — api",
		"localhost:4096 — elsewhere",
	}
	if got := labels(m); !reflect.DeepEqual(got, want) {
		t.Errorf("menu = %q, want %q", got, want)
	}
	if it := m.FindByLabel("probe — Error"); it.IsSubmenu() || !it.Enabled() {
		t.Error("sandbox line is a submenu or disabled")
	}
}

func TestAddStatusNoForwards(t *testing.T) {
	s := gateway.Status{Name: "gw", AuthMode: "plaintext", Info: gateway.Info{
		Sandboxes: []gateway.Sandbox{{Name: "web", Workspace: "default", Phase: "Ready"}},
	}}
	m := application.NewMenu()
	addStatus(m, s, time.Now(), false, noOpen)
	want := []string{"Gateway: gw · plaintext", "-", "Sandboxes", "web — Ready"}
	if got := labels(m); !reflect.DeepEqual(got, want) {
		t.Errorf("menu = %q, want %q", got, want)
	}
}

func TestAddStatusNoSandboxes(t *testing.T) {
	s := gateway.Status{Name: "gw", AuthMode: "none",
		OtherForwards: []gateway.Forward{{Sandbox: "elsewhere", Bind: "127.0.0.1", Port: 4096}}}
	m := application.NewMenu()
	addStatus(m, s, time.Now(), false, noOpen)
	want := []string{"Gateway: gw · none", "-", "No sandboxes", "-",
		"Port forwards", "localhost:4096 — elsewhere"}
	if got := labels(m); !reflect.DeepEqual(got, want) {
		t.Errorf("menu = %q, want %q", got, want)
	}
}

func TestAddStatusError(t *testing.T) {
	now := time.Date(2026, 11, 11, 10, 40, 0, 0, time.Local)
	s := gateway.Status{Name: "local", Endpoint: "https://127.0.0.1:17690", Err: errors.New("connection refused"),
		LastSeen: time.Date(2026, 11, 11, 10, 38, 0, 0, time.Local), Retry: 5 * time.Second}
	m := application.NewMenu()
	addStatus(m, s, now, false, noOpen)
	want := []string{"Gateway: local unreachable", "Last seen 10:38 · retrying every 5 s"}
	if got := labels(m); !reflect.DeepEqual(got, want) {
		t.Errorf("menu = %q, want %q", got, want)
	}
	for i := range want {
		if item := m.ItemAt(i); item.Enabled() || item.GetSubmenu() != nil {
			t.Errorf("item %d is enabled or has a submenu", i)
		}
	}
}

func TestAddStatusForwardError(t *testing.T) {
	s := gateway.Status{Name: "gw", AuthMode: "mtls", Info: gateway.Info{
		Sandboxes: []gateway.Sandbox{{Name: "web", Workspace: "default", Phase: "Ready"}},
	}, ForwardErr: errors.New("openshell forward list: exit status 1")}
	m := application.NewMenu()
	addStatus(m, s, time.Now(), false, noOpen)
	want := []string{"Gateway: gw · mTLS", "-", "Sandboxes", "web — Ready", "-",
		"Port forwards: unavailable"}
	if got := labels(m); !reflect.DeepEqual(got, want) {
		t.Errorf("menu = %q, want %q", got, want)
	}
	sub := m.FindByLabel("Port forwards: unavailable").GetSubmenu()
	if got := labels(sub); !reflect.DeepEqual(got, []string{"openshell forward list: exit status 1"}) {
		t.Errorf("submenu = %q", got)
	}
}

func TestVersionLine(t *testing.T) {
	if got := versionLine("dev", "20261111-133000"); got != "shellight dev · 20261111-133000" {
		t.Errorf("dev build: %q", got)
	}
	if got := versionLine("0.1.0", "20261111-133000"); got != "shellight 0.1.0" {
		t.Errorf("release: %q", got)
	}
}

func TestBrowseHost(t *testing.T) {
	for bind, want := range map[string]string{
		"127.0.0.1": "localhost", "::1": "localhost", "0.0.0.0": "localhost", "::": "localhost",
		"127.0.0.2": "127.0.0.2", "192.168.1.20": "192.168.1.20", "fe80::1": "fe80::1",
	} {
		if got := browseHost(bind); got != want {
			t.Errorf("browseHost(%q) = %q, want %q", bind, got, want)
		}
	}
}

func TestWrap(t *testing.T) {
	got := wrap("aaa bbb  ccc dddddddddd e", 7)
	want := []string{"aaa bbb", "ccc", "dddddddddd", "e"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrap = %q, want %q", got, want)
	}
}

func TestEdgeLines(t *testing.T) {
	now := time.Date(2026, 11, 11, 10, 40, 0, 0, time.Local)
	down := errors.New("connection refused")
	tests := []struct {
		name       string
		s          gateway.Status
		head, hint string
	}{
		{"not configured", gateway.Status{Err: errors.New("no active gateway")},
			"Gateway: not configured", "Set one up with the openshell CLI"},
		{"auth mode not supported",
			gateway.Status{Name: "corp", Err: &gateway.UnsupportedAuthModeError{Mode: "oidc"},
				LastSeen: now.Add(-time.Minute), Retry: 5 * time.Second},
			"Gateway: corp — auth mode oidc not supported", "Switch gateways with the openshell CLI"},
		{"unreachable since start", gateway.Status{Name: "local", Err: down, Retry: 5 * time.Second},
			"Gateway: local unreachable", "Retrying every 5 s"},
		{"unreachable, last seen today",
			gateway.Status{Name: "local", Err: down, Retry: 5 * time.Second,
				LastSeen: time.Date(2026, 11, 11, 10, 38, 0, 0, time.Local)},
			"Gateway: local unreachable", "Last seen 10:38 · retrying every 5 s"},
		{"backing off, last seen yesterday",
			gateway.Status{Name: "local", Err: down, Retry: 30 * time.Second,
				LastSeen: time.Date(2026, 11, 10, 18, 40, 0, 0, time.Local)},
			"Gateway: local unreachable", "Last seen Yesterday, 18:40 · retrying every 30 s"},
		{"backing off since start", gateway.Status{Name: "local", Err: down, Retry: 30 * time.Second},
			"Gateway: local unreachable", "Retrying every 30 s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			head, hint := edgeLines(tt.s, now)
			if head != tt.head || hint != tt.hint {
				t.Errorf("edgeLines = %q, %q; want %q, %q", head, hint, tt.head, tt.hint)
			}
		})
	}
}
