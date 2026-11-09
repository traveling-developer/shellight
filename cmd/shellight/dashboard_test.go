package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/traveling-developer/shellight/internal/gateway"
)

func TestDashboardStatus(t *testing.T) {
	reachable := gateway.Status{Name: "gw", Info: gateway.Info{
		Version: "0.1.7",
		Drivers: []string{"podman"},
		Sandboxes: []gateway.Sandbox{ // sorted by workspace, then name
			{Name: "a", Workspace: "default", Phase: "Ready"},
			{Name: "b", Workspace: "default", Phase: "Provisioning"},
			{Name: "c", Workspace: "team", Phase: "Error"},
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
			Gateway: "Gateway: gw · 0.1.7 · podman",
			Workspaces: []DashboardWorkspace{
				{Name: "default", Sandboxes: []DashboardSandbox{
					{Name: "a", Phase: "Ready", State: "ok"},
					{Name: "b", Phase: "Provisioning", State: "busy"},
				}},
				{Name: "team", Sandboxes: []DashboardSandbox{{Name: "c", Phase: "Error", State: "error"}}},
			},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dashboardStatus(tt.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dashboardStatus =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}
