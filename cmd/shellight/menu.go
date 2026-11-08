package main

import (
	"fmt"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/traveling-developer/shellight/internal/gateway"
	"github.com/traveling-developer/shellight/internal/trayicon"
)

// Status dot colours, Apple system colours.
var (
	unknownDot = trayicon.RGB{R: 0x8e, G: 0x8e, B: 0x93} // grey: no result yet
	okDot      = trayicon.RGB{R: 0x34, G: 0xc7, B: 0x59} // green
	busyDot    = trayicon.RGB{R: 0xff, G: 0xcc, B: 0x00} // yellow
	errorDot   = trayicon.RGB{R: 0xff, G: 0x3b, B: 0x30} // red
)

// dot picks the status dot: red if the gateway is unreachable or a sandbox
// is in phase Error, yellow if a sandbox is between phases, else green.
func dot(s gateway.Status) trayicon.RGB {
	if s.Err != nil {
		return errorDot
	}
	busy := false
	for _, sb := range s.Info.Sandboxes {
		switch sb.Phase {
		case "Error":
			return errorDot
		case "Ready", "Stopped", "Completed", "Suspended":
		default: // Provisioning, Starting, Stopping, Deleting, Suspending, Unknown
			busy = true
		}
	}
	if busy {
		return busyDot
	}
	return okDot
}

// addStatus adds the gateway line and the sandboxes of s to m.
func addStatus(m *application.Menu, s gateway.Status) {
	if s.Err != nil {
		label := "Gateway: not configured"
		if s.Name != "" {
			label = "Gateway: " + s.Name + " — unreachable"
		}
		sub := m.AddSubmenu(label)
		if s.Endpoint != "" {
			sub.Add(s.Endpoint).SetEnabled(false)
		}
		addLines(sub, s.Err.Error())
		return
	}

	m.Add(fmt.Sprintf("Gateway: %s · %s · %s",
		s.Name, s.Info.Version, strings.Join(s.Info.Drivers, ", "))).SetEnabled(false)
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
	}
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
