package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/traveling-developer/shellight/internal/gateway"
	"github.com/traveling-developer/shellight/internal/trayicon"
)

// Status dot colours for the white glyph (docs/design/README.md); also the
// status shape colours of a dark menu.
var (
	unknownDot = trayicon.RGB{R: 0x9a, G: 0x9e, B: 0x95} // no result yet
	okDot      = trayicon.RGB{R: 0xb8, G: 0xe6, B: 0x2a}
	busyDot    = trayicon.RGB{R: 0xf2, G: 0xb8, B: 0x4b}
	errorDot   = trayicon.RGB{R: 0xff, G: 0x7a, B: 0x66}
)

// Phase states, which pick the shape and colour of a sandbox's status.
const (
	phaseOK    = "ok"    // settled
	phaseBusy  = "busy"  // between phases
	phaseError = "error" // phase Error
)

// phaseState sorts a sandbox phase into a phase state.
func phaseState(phase string) string {
	switch phase {
	case "Error":
		return phaseError
	case "Ready", "Stopped", "Completed", "Suspended":
		return phaseOK
	default: // Provisioning, Starting, Stopping, Deleting, Suspending, Unknown
		return phaseBusy
	}
}

// dot picks the status dot: error if the gateway is unreachable or a
// sandbox is in phase Error, busy if a sandbox is between phases, else ok.
func dot(s gateway.Status) trayicon.RGB {
	if s.Err != nil {
		return errorDot
	}
	busy := false
	for _, sb := range s.Info.Sandboxes {
		switch phaseState(sb.Phase) {
		case phaseError:
			return errorDot
		case phaseBusy:
			busy = true
		}
	}
	if busy {
		return busyDot
	}
	return okDot
}

// menuShapes are the status shapes of the menu's sandbox lines by phase
// state, for a light (false) and a dark (true) menu, in the colours of the
// design mockups. A menu item image does not follow the menu's appearance,
// so the menu is rebuilt when the system appearance changes.
var menuShapes = map[bool]map[string][]byte{
	false: statusShapes(
		trayicon.RGB{R: 0x3e, G: 0x6b, B: 0x00},
		trayicon.RGB{R: 0xb4, G: 0x56, B: 0x0b},
		trayicon.RGB{R: 0xc4, G: 0x32, B: 0x1f}),
	true: statusShapes(okDot, busyDot, errorDot),
}

// statusShapes renders the status shape of each phase state.
func statusShapes(ok, busy, bad trayicon.RGB) map[string][]byte {
	return map[string][]byte{
		phaseOK:    trayicon.StatusShape(trayicon.Circle, ok),
		phaseBusy:  trayicon.StatusShape(trayicon.Ring, busy),
		phaseError: trayicon.StatusShape(trayicon.Triangle, bad),
	}
}

// connecting is the gateway line before the first poll result.
const connecting = "Gateway: connecting…"

// gatewayLine is the dashboard's gateway line of a reachable gateway.
func gatewayLine(s gateway.Status) string {
	return fmt.Sprintf("Gateway: %s · %s · %s", s.Name, s.Info.Version, strings.Join(s.Info.Drivers, ", "))
}

// menuGatewayLine is the menu's gateway line of a reachable gateway.
func menuGatewayLine(s gateway.Status) string {
	mode := s.AuthMode
	if mode == "mtls" {
		mode = "mTLS"
	}
	return "Gateway: " + s.Name + " · " + mode
}

// versionLine names the shellight build: its version, or the build ID of a
// dev build.
func versionLine(version, buildID string) string {
	if version == "dev" {
		return "shellight dev · " + buildID
	}
	return "shellight " + version
}

// edgeLines are the two lines that say what is wrong with the gateway if
// s.Err is set, and what the user can do, at now.
func edgeLines(s gateway.Status, now time.Time) (head, hint string) {
	var unsupported *gateway.UnsupportedAuthModeError
	switch {
	case s.Name == "":
		return "Gateway: not configured", "Set one up with the openshell CLI"
	case errors.As(s.Err, &unsupported):
		return "Gateway: " + s.Name + " — auth mode " + unsupported.Mode + " not supported",
			"Switch gateways with the openshell CLI"
	}
	head = "Gateway: " + s.Name + " unreachable"
	retry := fmt.Sprintf("every %d s", int(s.Retry.Round(time.Second)/time.Second))
	if s.LastSeen.IsZero() {
		return head, "Retrying " + retry
	}
	return head, "Last seen " + seenLabel(s.LastSeen, now) + " · retrying " + retry
}

// seenLabel formats a last seen time like createdLabel, but without
// "Today, ": "10:38", "Yesterday, 18:40", "3 Nov, 09:12".
func seenLabel(t, now time.Time) string {
	return strings.TrimPrefix(createdLabel(t, now), "Today, ")
}

// addStatus adds the gateway line, the sandboxes and the port forwards of s
// to m, with status shapes for a dark menu if dark is set, or the edge
// lines at now if the gateway has a problem. A click on a sandbox calls
// openSandbox with its workspace and name.
func addStatus(m *application.Menu, s gateway.Status, now time.Time, dark bool, openSandbox func(workspace, name string)) {
	if s.Err != nil {
		head, hint := edgeLines(s, now)
		m.Add(head).SetEnabled(false)
		m.Add(hint).SetEnabled(false)
		return
	}

	m.Add(menuGatewayLine(s)).SetEnabled(false)
	m.AddSeparator()
	if len(s.Info.Sandboxes) == 0 {
		m.Add("No sandboxes").SetEnabled(false)
	} else {
		m.Add("Sandboxes").SetEnabled(false)
	}
	var fwds []gateway.Forward
	for _, sb := range s.Info.Sandboxes {
		m.Add(sb.Name + " — " + sb.Phase).
			SetBitmap(menuShapes[dark][phaseState(sb.Phase)]).
			OnClick(func(*application.Context) { openSandbox(sb.Workspace, sb.Name) })
		fwds = append(fwds, sb.Forwards...)
	}
	fwds = append(fwds, s.OtherForwards...)

	switch {
	case s.ForwardErr != nil:
		m.AddSeparator()
		addLines(m.AddSubmenu("Port forwards: unavailable"), s.ForwardErr.Error())
	case len(fwds) > 0:
		m.AddSeparator()
		m.Add("Port forwards").SetEnabled(false)
		for _, f := range fwds {
			addForward(m, f)
		}
	}
}

// addForward adds a port forward as "host:port — sandbox", which opens it
// in the browser.
func addForward(m *application.Menu, f gateway.Forward) {
	hostPort, url := forwardTarget(f)
	m.Add(hostPort + " — " + f.Sandbox).OnClick(func(*application.Context) {
		if err := application.Get().Browser.OpenURL(url); err != nil {
			log.Printf("open %s: %v", url, err)
		}
	})
}

// forwardTarget returns what to open for a forward: its "host:port" and its
// http URL.
func forwardTarget(f gateway.Forward) (hostPort, url string) {
	hostPort = net.JoinHostPort(browseHost(f.Bind), strconv.Itoa(f.Port))
	return hostPort, "http://" + hostPort
}

// browseHost is the host to open a forward bound to bind: localhost for the
// loopback address, and for a wildcard address, which is not reachable as
// such. Other loopback addresses such as 127.0.0.2 stay, as localhost does
// not reach them.
func browseHost(bind string) string {
	switch bind {
	case "127.0.0.1", "::1", "0.0.0.0", "::":
		return "localhost"
	}
	return bind
}

// addLines adds text as disabled items, wrapped at word boundaries, so a
// long error does not make the menu very wide.
func addLines(m *application.Menu, text string) {
	for _, line := range wrap(text, 60) {
		m.Add(line).SetEnabled(false)
	}
}

// wrap splits text into lines of at most width runes where possible; a
// single longer word stays on its own line.
func wrap(text string, width int) []string {
	var lines []string
	line := ""
	for _, w := range strings.Fields(text) {
		switch {
		case line == "":
			line = w
		case len([]rune(line))+1+len([]rune(w)) <= width:
			line += " " + w
		default:
			lines = append(lines, line)
			line = w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
