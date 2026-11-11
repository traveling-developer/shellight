package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/traveling-developer/shellight/internal/gateway"
	"github.com/traveling-developer/shellight/internal/trayicon"
)

// Status dot colours for the white glyph (docs/design/README.md).
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

// connecting is the gateway line before the first poll result.
const connecting = "Gateway: connecting…"

// gatewayLine is the gateway line of a reachable gateway.
func gatewayLine(s gateway.Status) string {
	return fmt.Sprintf("Gateway: %s · %s · %s", s.Name, s.Info.Version, strings.Join(s.Info.Drivers, ", "))
}

// problem says what is wrong with the gateway if s.Err is set.
func problem(s gateway.Status) string {
	var unsupported *gateway.UnsupportedAuthModeError
	switch {
	case s.Name == "":
		return "Gateway: not configured"
	case errors.As(s.Err, &unsupported):
		return "Gateway: " + s.Name + " — auth mode " + unsupported.Mode + " not supported"
	default:
		return "Gateway: " + s.Name + " — unreachable"
	}
}

// addStatus adds the gateway line and the sandboxes of s to m.
func addStatus(m *application.Menu, s gateway.Status) {
	if s.Err != nil {
		sub := m.AddSubmenu(problem(s))
		if s.Endpoint != "" {
			sub.Add(s.Endpoint).SetEnabled(false)
		}
		addLines(sub, s.Err.Error())
		return
	}

	m.Add(gatewayLine(s)).SetEnabled(false)
	m.AddSeparator()
	if len(s.Info.Sandboxes) == 0 {
		m.Add("No sandboxes").SetEnabled(false)
		return
	}
	m.Add("Sandboxes").SetEnabled(false)
	for _, sb := range s.Info.Sandboxes {
		sub := m.AddSubmenu(sb.Name + " — " + sb.Phase)
		sub.Add("Workspace: " + sb.Workspace).SetEnabled(false)
		sub.Add("Created: " + sb.Created.Local().Format("2006-01-02 15:04")).SetEnabled(false)
		sub.Add(fmt.Sprintf("Policy: v%d", sb.PolicyVersion)).SetEnabled(false)
		if sb.Reason != "" {
			addLines(sub, sb.Reason)
		}
		if len(sb.Forwards) > 0 {
			sub.AddSeparator()
			sub.Add("Port forwards").SetEnabled(false)
			for _, f := range sb.Forwards {
				addForward(sub, f)
			}
		}
	}
	// Forwards only show under their sandbox, so this matters only with
	// sandboxes.
	if s.ForwardErr != nil {
		addLines(m.AddSubmenu("Port forwards: unavailable"), s.ForwardErr.Error())
	}
}

// addForward adds a port forward as "bind:port", which opens it in the
// browser.
func addForward(m *application.Menu, f gateway.Forward) {
	_, url := forwardTarget(f)
	m.Add(net.JoinHostPort(f.Bind, strconv.Itoa(f.Port))).OnClick(func(*application.Context) {
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
