package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/traveling-developer/shellight/internal/gateway"
)

func TestDashboardStatus(t *testing.T) {
	now := time.Date(2026, 11, 11, 10, 0, 0, 0, time.Local)
	reachable := gateway.Status{Name: "gw", ForwardErr: errors.New("openshell forward list: exit status 1"), Info: gateway.Info{
		Version: "0.1.7",
		Drivers: []string{"podman"},
		Sandboxes: []gateway.Sandbox{ // sorted by workspace, then name
			{Name: "a", Workspace: "default", Phase: "Ready", PolicyVersion: 7,
				Created:  time.Date(2026, 11, 11, 9, 12, 0, 0, time.Local),
				Forwards: []gateway.Forward{{Bind: "127.0.0.1", Port: 5173}}},
			{Name: "b", Workspace: "default", Phase: "Provisioning", PolicyVersion: 1,
				Created: time.Date(2026, 11, 3, 9, 12, 0, 0, time.Local)},
			{Name: "c", Workspace: "team", Phase: "Error", PolicyVersion: 2,
				Reason:  "Canonical main process exited with status 3",
				Created: time.Date(2026, 11, 10, 18, 40, 0, 0, time.Local)},
		},
	}}

	tests := []struct {
		name string
		s    *gateway.Status
		want DashboardStatus
	}{
		{"before the first poll", nil,
			DashboardStatus{Problem: "Gateway: connecting…", Workspaces: []DashboardWorkspace{}}},
		{"unreachable", &gateway.Status{Name: "gw", Err: errors.New("connection refused")},
			DashboardStatus{Problem: "Gateway: gw — unreachable", Detail: "connection refused", Workspaces: []DashboardWorkspace{}}},
		{"no sandboxes", &gateway.Status{Name: "gw", Info: gateway.Info{Version: "0.1.7", Drivers: []string{"podman"}}},
			DashboardStatus{Gateway: "Gateway: gw · 0.1.7 · podman", Workspaces: []DashboardWorkspace{}}},
		{"grouped by workspace", &reachable, DashboardStatus{
			Gateway:    "Gateway: gw · 0.1.7 · podman",
			ForwardErr: "openshell forward list: exit status 1",
			Workspaces: []DashboardWorkspace{
				{Name: "default", Sandboxes: []DashboardSandbox{
					{Name: "a", Workspace: "default", Phase: "Ready", State: "ok", Created: "Today, 09:12", PolicyVersion: 7,
						Forwards: []DashboardForward{{HostPort: "localhost:5173", URL: "http://localhost:5173", Port: 5173}}},
					{Name: "b", Workspace: "default", Phase: "Provisioning", State: "busy", Created: "3 Nov, 09:12", PolicyVersion: 1},
				}},
				{Name: "team", Sandboxes: []DashboardSandbox{
					{Name: "c", Workspace: "team", Phase: "Error", State: "error", Created: "Yesterday, 18:40", PolicyVersion: 2,
						Reason: "Canonical main process exited with status 3"},
				}},
			},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dashboardStatus(tt.s, now); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dashboardStatus =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}

func TestCreatedLabel(t *testing.T) {
	at := func(y int, m time.Month, d, h, min int) time.Time {
		return time.Date(y, m, d, h, min, 0, 0, time.Local)
	}
	now, newYear := at(2026, 11, 11, 10, 42), at(2027, 1, 1, 8, 0)
	tests := []struct {
		now, t time.Time
		want   string
	}{
		{now, at(2026, 11, 11, 0, 0), "Today, 00:00"},
		{now, at(2026, 11, 11, 9, 12), "Today, 09:12"},
		{now, at(2026, 11, 10, 23, 59), "Yesterday, 23:59"},
		{now, at(2026, 11, 10, 0, 0), "Yesterday, 00:00"},
		{now, at(2026, 11, 9, 23, 59), "9 Nov, 23:59"},
		{now, at(2026, 1, 3, 9, 12), "3 Jan, 09:12"},
		{now, at(2025, 12, 31, 18, 40), "31 Dec 2025, 18:40"},
		{now, at(2026, 11, 12, 8, 0), "12 Nov, 08:00"},          // clock skew: in the future
		{now, at(2026, 11, 11, 9, 12).UTC(), "Today, 09:12"},    // shown in local time
		{newYear, at(2026, 12, 31, 18, 40), "Yesterday, 18:40"}, // across new year
	}
	for _, tt := range tests {
		if got := createdLabel(tt.t, tt.now); got != tt.want {
			t.Errorf("createdLabel(%v, now %v) = %q, want %q", tt.t, tt.now, got, tt.want)
		}
	}
}
