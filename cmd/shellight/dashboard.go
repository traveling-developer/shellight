package main

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/dock"

	"github.com/traveling-developer/shellight/internal/gateway"
)

// statusEvent carries a DashboardStatus to the dashboard after every poll
// round that changed something.
const statusEvent = "dashboard:status"

func init() {
	application.RegisterEvent[DashboardStatus](statusEvent)
}

// DashboardStatus is what the dashboard shows of a poll round.
type DashboardStatus struct {
	Gateway    string               `json:"gateway"` // gateway line if the gateway is reachable
	Problem    string               `json:"problem"` // what is wrong with the gateway, else empty
	Detail     string               `json:"detail"`  // the error behind Problem
	Workspaces []DashboardWorkspace `json:"workspaces"`
}

// DashboardWorkspace is a workspace with its sandboxes, sorted by name.
type DashboardWorkspace struct {
	Name      string             `json:"name"`
	Sandboxes []DashboardSandbox `json:"sandboxes"`
}

// DashboardSandbox is a sandbox in the dashboard sidebar.
type DashboardSandbox struct {
	Name  string `json:"name"`
	Phase string `json:"phase"`
	State string `json:"state"` // ok, busy or error: its status shape
}

// dashboardStatus turns a poll result into what the dashboard shows; s is
// nil before the first poll result.
func dashboardStatus(s *gateway.Status) DashboardStatus {
	ds := DashboardStatus{Workspaces: []DashboardWorkspace{}}
	switch {
	case s == nil:
		ds.Problem = connecting
		return ds
	case s.Err != nil:
		ds.Problem, ds.Detail = problem(*s), s.Err.Error()
		return ds
	}
	ds.Gateway = gatewayLine(*s)
	for _, sb := range s.Info.Sandboxes { // sorted by workspace, then name
		if n := len(ds.Workspaces); n == 0 || ds.Workspaces[n-1].Name != sb.Workspace {
			ds.Workspaces = append(ds.Workspaces, DashboardWorkspace{Name: sb.Workspace})
		}
		ws := &ds.Workspaces[len(ds.Workspaces)-1]
		ws.Sandboxes = append(ws.Sandboxes, DashboardSandbox{Name: sb.Name, Phase: sb.Phase, State: phaseState(sb.Phase)})
	}
	return ds
}

// Dashboard owns the dashboard window. Its exported methods are bound to
// the frontend.
type Dashboard struct {
	app  *application.App // set once the app exists
	dock *dock.DockService

	mu     sync.Mutex
	status DashboardStatus
	window *application.WebviewWindow // nil while closed
}

// newDashboard returns a dashboard showing "connecting"; set its app before
// the app runs.
func newDashboard() *Dashboard {
	return &Dashboard{dock: dock.New(), status: dashboardStatus(nil)}
}

// Status returns what the dashboard shows now; the frontend calls it when
// it loads and then follows statusEvent.
func (d *Dashboard) Status() DashboardStatus {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.status
}

// update stores s and passes it to an open dashboard.
func (d *Dashboard) update(s DashboardStatus) {
	d.mu.Lock()
	d.status = s
	d.mu.Unlock()
	d.app.Event.Emit(statusEvent, s)
}

// open opens the dashboard, or brings it to front if it is open. The Dock
// icon shows while the dashboard is open. Closing the window destroys it,
// so a closed dashboard holds no web view.
func (d *Dashboard) open() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.window == nil {
		w := d.app.Window.NewWithOptions(application.WebviewWindowOptions{
			Title:     "shellight",
			Width:     900,
			Height:    600,
			MinWidth:  600,
			MinHeight: 400,
			URL:       "/",
			// No visible title bar: the sidebar runs up under the
			// traffic lights, its top drags the window.
			Mac: application.MacWindow{TitleBar: application.MacTitleBarHiddenInset},
		})
		w.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
			d.mu.Lock()
			if d.window == w {
				d.window = nil
			}
			d.mu.Unlock()
			d.dock.HideAppIcon()
		})
		d.window = w
		d.dock.ShowAppIcon()
	}
	d.window.Show()
	d.window.Focus()
}
