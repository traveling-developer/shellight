package main

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/dock"

	"github.com/traveling-developer/shellight/internal/gateway"
)

// statusEvent carries a DashboardStatus to the dashboard after every poll
// round that changed something.
const statusEvent = "dashboard:status"

// selectEvent carries a SandboxKey to an open dashboard to select that
// sandbox.
const selectEvent = "dashboard:select"

func init() {
	application.RegisterEvent[DashboardStatus](statusEvent)
	application.RegisterEvent[SandboxKey](selectEvent)
}

// SandboxKey identifies a sandbox.
type SandboxKey struct {
	Workspace string `json:"workspace"`
	Name      string `json:"name"`
}

// DashboardStatus is what the dashboard shows of a poll round.
type DashboardStatus struct {
	Gateway    string               `json:"gateway"` // gateway line if the gateway is reachable
	Problem    string               `json:"problem"` // what is wrong with the gateway, else empty
	Detail     string               `json:"detail"`  // the error behind Problem
	Workspaces []DashboardWorkspace `json:"workspaces"`
	// ForwardErr is set if listing the port forwards failed; then no
	// sandbox has forwards.
	ForwardErr string `json:"forwardErr"`
}

// DashboardWorkspace is a workspace with its sandboxes, sorted by name.
type DashboardWorkspace struct {
	Name      string             `json:"name"`
	Sandboxes []DashboardSandbox `json:"sandboxes"`
}

// DashboardSandbox is a sandbox in the dashboard sidebar and its details.
type DashboardSandbox struct {
	Name          string             `json:"name"`
	Workspace     string             `json:"workspace"`
	Phase         string             `json:"phase"`
	State         string             `json:"state"`   // ok, busy or error: its status shape
	Reason        string             `json:"reason"`  // why it is in phase Error, else empty
	Created       string             `json:"created"` // e.g. "Today, 09:12"
	PolicyVersion uint32             `json:"policyVersion"`
	Forwards      []DashboardForward `json:"forwards"` // running ones, sorted by port
}

// DashboardForward is a running port forward to a sandbox. The CLI
// forwards a local port to the same port in the sandbox.
type DashboardForward struct {
	HostPort string `json:"hostPort"` // what to open, e.g. localhost:5173
	URL      string `json:"url"`
	Port     int    `json:"port"`
}

// dashboardStatus turns a poll result into what the dashboard shows at
// now; s is nil before the first poll result.
func dashboardStatus(s *gateway.Status, now time.Time) DashboardStatus {
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
	if s.ForwardErr != nil {
		ds.ForwardErr = s.ForwardErr.Error()
	}
	for _, sb := range s.Info.Sandboxes { // sorted by workspace, then name
		if n := len(ds.Workspaces); n == 0 || ds.Workspaces[n-1].Name != sb.Workspace {
			ds.Workspaces = append(ds.Workspaces, DashboardWorkspace{Name: sb.Workspace})
		}
		ws := &ds.Workspaces[len(ds.Workspaces)-1]
		ws.Sandboxes = append(ws.Sandboxes, dashboardSandbox(sb, now))
	}
	return ds
}

func dashboardSandbox(sb gateway.Sandbox, now time.Time) DashboardSandbox {
	d := DashboardSandbox{
		Name:          sb.Name,
		Workspace:     sb.Workspace,
		Phase:         sb.Phase,
		State:         phaseState(sb.Phase),
		Reason:        sb.Reason,
		Created:       createdLabel(sb.Created, now),
		PolicyVersion: sb.PolicyVersion,
	}
	for _, f := range sb.Forwards {
		hostPort, url := forwardTarget(f)
		d.Forwards = append(d.Forwards, DashboardForward{HostPort: hostPort, URL: url, Port: f.Port})
	}
	return d
}

// createdLabel formats a created time in now's time zone: "Today, 09:12",
// "Yesterday, 18:40", "3 Nov, 09:12", or "3 Nov 2025, 09:12" if not in
// now's year.
func createdLabel(t, now time.Time) string {
	t = t.In(now.Location())
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	tomorrow, yesterday := today.AddDate(0, 0, 1), today.AddDate(0, 0, -1)
	switch {
	case !t.Before(today) && t.Before(tomorrow):
		return "Today, " + t.Format("15:04")
	case !t.Before(yesterday) && t.Before(today):
		return "Yesterday, " + t.Format("15:04")
	case t.Year() == y:
		return t.Format("2 Jan, 15:04")
	default:
		return t.Format("2 Jan 2006, 15:04")
	}
}

// Dashboard owns the dashboard window. Its exported methods are bound to
// the frontend.
type Dashboard struct {
	app  *application.App // set once the app exists
	dock *dock.DockService

	mu     sync.Mutex
	status DashboardStatus
	window *application.WebviewWindow // nil while closed
	// selection is the sandbox to select once a new dashboard has loaded;
	// nil to select the first sandbox.
	selection *SandboxKey
}

// newDashboard returns a dashboard showing "connecting"; set its app before
// the app runs.
func newDashboard() *Dashboard {
	return &Dashboard{dock: dock.New(), status: dashboardStatus(nil, time.Now())}
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

// Selection returns the sandbox to select when the dashboard loads, or nil
// for the first sandbox; the frontend calls it once it follows selectEvent.
func (d *Dashboard) Selection() *SandboxKey {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.selection
}

// openAt opens the dashboard with the sandbox workspace/name selected.
func (d *Dashboard) openAt(workspace, name string) {
	key := SandboxKey{Workspace: workspace, Name: name}
	d.mu.Lock()
	d.selection = &key
	d.mu.Unlock()
	// An open dashboard follows the event; a new one asks for Selection.
	d.app.Event.Emit(selectEvent, key)
	d.show()
}

// open opens the dashboard with its selection as it was, or with the first
// sandbox selected if it is new.
func (d *Dashboard) open() {
	d.mu.Lock()
	d.selection = nil
	d.mu.Unlock()
	d.show()
}

// show opens the dashboard, or brings it to front if it is open. The Dock
// icon shows while the dashboard is open. Closing the window destroys it,
// so a closed dashboard holds no web view.
func (d *Dashboard) show() {
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
