// Command shellight is the shellight menu bar app for NVIDIA OpenShell.
package main

import (
	"context"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/traveling-developer/shellight/internal/gateway"
	"github.com/traveling-developer/shellight/internal/trayicon"
)

// buildID is set at link time (scripts/build-darwin.sh) to tell builds apart.
var buildID = "dev"

// whiteGlyph is the menu bar glyph colour.
var whiteGlyph = trayicon.RGB{R: 0xff, G: 0xff, B: 0xff}

func main() {
	app := application.New(application.Options{
		Name:        "shellight",
		Description: "Menu bar app for NVIDIA OpenShell",
		Mac: application.MacOptions{
			// Menu bar app: no Dock icon, keep running without windows.
			ActivationPolicy: application.ActivationPolicyAccessory,
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	menu := app.NewMenu()
	// No window is attached, so a click on the icon opens the menu.
	tray := app.SystemTray.New()
	tray.SetTooltip("shellight")

	// render rebuilds the menu and the status dot; s is nil before the first
	// poll result.
	render := func(s *gateway.Status) {
		menu.Clear()
		dotColour := unknownDot
		if s == nil {
			menu.Add("Gateway: connecting…").SetEnabled(false)
		} else {
			addStatus(menu, *s)
			dotColour = dot(*s)
		}
		menu.AddSeparator()
		menu.Add("Build " + buildID).SetEnabled(false)
		menu.Add("Quit").OnClick(func(*application.Context) {
			app.Quit()
		})
		menu.Update()

		// The icon is not a template image (the dot keeps its colour). The
		// glyph is always white; a setting to switch it is in docs/backlog.md.
		tray.SetIcon(trayicon.Badged(whiteGlyph, dotColour))
	}
	render(nil)
	tray.SetMenu(menu)

	ctx, cancel := context.WithCancel(context.Background())
	app.OnShutdown(cancel)
	// Menu updates only take effect once the app runs.
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go gateway.Poll(ctx, func(s gateway.Status) {
			application.InvokeSync(func() { render(&s) })
		})
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
