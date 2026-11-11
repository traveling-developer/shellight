// Command shellight is the shellight menu bar app for NVIDIA OpenShell.
package main

import (
	"context"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/traveling-developer/shellight/frontend"
	"github.com/traveling-developer/shellight/internal/gateway"
	"github.com/traveling-developer/shellight/internal/trayicon"
)

// buildID is set at link time (scripts/build-darwin.sh) to tell builds apart.
var buildID = "dev"

// version is the release version, set at link time for a release; "dev"
// otherwise.
var version = "dev"

// whiteGlyph is the menu bar glyph colour.
var whiteGlyph = trayicon.RGB{R: 0xff, G: 0xff, B: 0xff}

func main() {
	// The dashboard is bound as a service, so it exists before the app.
	dash := newDashboard()
	app := application.New(application.Options{
		Name:        "shellight",
		Description: "Menu bar app for NVIDIA OpenShell",
		Services:    []application.Service{application.NewService(dash)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(frontend.Dist())},
		Mac: application.MacOptions{
			// Menu bar app: no Dock icon (only while the dashboard is
			// open), keep running without windows.
			ActivationPolicy: application.ActivationPolicyAccessory,
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})
	dash.app = app

	menu := app.NewMenu()
	// No window is attached, so a click on the icon opens the menu.
	tray := app.SystemTray.New()
	tray.SetTooltip("shellight")

	// render rebuilds the menu and the status dot; s is nil before the first
	// poll result. last is what it rendered, to rebuild the menu when the
	// system appearance changes.
	var last *gateway.Status
	render := func(s *gateway.Status) {
		last = s
		menu.Clear()
		dotColour := unknownDot
		if s == nil {
			menu.Add(connecting).SetEnabled(false)
		} else {
			addStatus(menu, *s, app.Env.IsDarkMode(), dash.openAt)
			dotColour = dot(*s)
		}
		menu.AddSeparator()
		menu.Add("Open dashboard…").OnClick(func(*application.Context) {
			dash.open()
		})
		menu.Add(versionLine(version, buildID)).SetEnabled(false)
		menu.Add("Quit shellight").SetAccelerator("CmdOrCtrl+Q").OnClick(func(*application.Context) {
			app.Quit()
		})
		menu.Update()

		// The icon is not a template image (the dot keeps its colour). The
		// glyph is always white until a setting can switch it to black.
		tray.SetIcon(trayicon.Badged(whiteGlyph, dotColour))
	}
	render(nil)
	tray.SetMenu(menu)

	ctx, cancel := context.WithCancel(context.Background())
	app.OnShutdown(cancel)
	// Menu updates only take effect once the app runs.
	var tracker proposalTracker
	var notes notifier
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go gateway.Poll(ctx, func(s gateway.Status) {
			if ns := tracker.update(s); len(ns) > 0 {
				go notes.send(ns)
			}
			dash.update(dashboardStatus(&s, time.Now()))
			application.InvokeSync(func() { render(&s) })
		})
	})

	app.Event.OnApplicationEvent(events.Common.ThemeChanged, func(*application.ApplicationEvent) {
		application.InvokeSync(func() { render(last) })
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
