package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/traveling-developer/shellight/internal/gateway"
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
		want string
	}{
		{"unreachable", gateway.Status{Name: "gw", Err: errors.New("down")}, "red"},
		{"no sandboxes", status(), "green"},
		{"all settled", status("Ready", "Stopped", "Completed", "Suspended"), "green"},
		{"one between phases", status("Ready", "Provisioning"), "yellow"},
		{"unknown phase", status("Unknown"), "yellow"},
		{"error wins", status("Provisioning", "Error"), "red"},
	}
	names := map[any]string{okDot: "green", busyDot: "yellow", errorDot: "red"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := names[dot(tt.s)]; got != tt.want {
				t.Errorf("dot = %s, want %s", got, tt.want)
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

func TestAddStatus(t *testing.T) {
	created := time.Date(2026, 11, 8, 15, 20, 0, 0, time.Local)
	s := gateway.Status{Name: "gw", Info: gateway.Info{
		Version: "0.1.7",
		Drivers: []string{"podman"},
		Sandboxes: []gateway.Sandbox{{
			Name: "probe", Workspace: "default", Phase: "Error", Created: created,
			PolicyVersion: 1, Reason: "Canonical main process exited with status 3",
		}},
	}}
	m := application.NewMenu()
	addStatus(m, s)
	want := []string{"Gateway: gw · 0.1.7 · podman", "-", "Sandboxes", "probe — Error"}
	if got := labels(m); !reflect.DeepEqual(got, want) {
		t.Errorf("menu = %q, want %q", got, want)
	}
	want = []string{"Workspace: default", "Created: 2026-11-08 15:20", "Policy: v1",
		"Canonical main process exited with status 3"}
	if got := labels(m.FindByLabel("probe — Error").GetSubmenu()); !reflect.DeepEqual(got, want) {
		t.Errorf("submenu = %q, want %q", got, want)
	}
}

func TestAddStatusError(t *testing.T) {
	m := application.NewMenu()
	addStatus(m, gateway.Status{Name: "gw", Endpoint: "https://127.0.0.1:17690", Err: errors.New("connection refused")})
	if got := labels(m); !reflect.DeepEqual(got, []string{"Gateway: gw — unreachable"}) {
		t.Errorf("menu = %q", got)
	}
	want := []string{"https://127.0.0.1:17690", "connection refused"}
	if got := labels(m.ItemAt(0).GetSubmenu()); !reflect.DeepEqual(got, want) {
		t.Errorf("submenu = %q, want %q", got, want)
	}

	m = application.NewMenu()
	addStatus(m, gateway.Status{Err: errors.New("no active gateway")})
	if got := labels(m); !reflect.DeepEqual(got, []string{"Gateway: not configured"}) {
		t.Errorf("menu = %q", got)
	}
}

func TestAddStatusForwards(t *testing.T) {
	s := gateway.Status{Name: "gw", Info: gateway.Info{
		Version: "0.1.7",
		Drivers: []string{"podman"},
		Sandboxes: []gateway.Sandbox{{
			Name: "web", Workspace: "default", Phase: "Ready", PolicyVersion: 1,
			Created: time.Date(2026, 11, 8, 16, 20, 0, 0, time.Local),
			Forwards: []gateway.Forward{
				{Bind: "127.0.0.1", Port: 8080},
				{Bind: "::1", Port: 9000},
			},
		}},
	}, ForwardErr: errors.New("openshell forward list: exit status 1")}
	m := application.NewMenu()
	addStatus(m, s)
	want := []string{"Gateway: gw · 0.1.7 · podman", "-", "Sandboxes", "web — Ready",
		"Port forwards: unavailable"}
	if got := labels(m); !reflect.DeepEqual(got, want) {
		t.Errorf("menu = %q, want %q", got, want)
	}
	sub := m.FindByLabel("web — Ready").GetSubmenu()
	want = []string{"Workspace: default", "Created: 2026-11-08 16:20", "Policy: v1", "-",
		"Port forwards", "127.0.0.1:8080", "[::1]:9000"}
	if got := labels(sub); !reflect.DeepEqual(got, want) {
		t.Errorf("submenu = %q, want %q", got, want)
	}
}

func TestBrowseHost(t *testing.T) {
	for bind, want := range map[string]string{
		"127.0.0.1": "127.0.0.1", "::1": "::1", "0.0.0.0": "localhost", "::": "localhost",
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

func TestAddStatusErrors(t *testing.T) {
	tests := []struct {
		name string
		s    gateway.Status
		want string
	}{
		{"not configured", gateway.Status{Err: errors.New("no active gateway")}, "Gateway: not configured"},
		{"unreachable", gateway.Status{Name: "gw", Err: errors.New("down")}, "Gateway: gw — unreachable"},
		{"auth mode not supported",
			gateway.Status{Name: "gw", Err: &gateway.UnsupportedAuthModeError{Mode: "oidc"}},
			"Gateway: gw — auth mode oidc not supported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := application.NewMenu()
			addStatus(m, tt.s)
			if got := labels(m); !reflect.DeepEqual(got, []string{tt.want}) {
				t.Errorf("menu = %q, want %q", got, []string{tt.want})
			}
		})
	}
}
