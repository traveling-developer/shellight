// Command shellight is the shellight menu bar app for NVIDIA OpenShell.
package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/traveling-developer/shellight/internal/gateway"
	"github.com/traveling-developer/shellight/internal/trayicon"
)

// buildID is set at link time (scripts/build-darwin.sh) to tell builds apart.
var buildID = "dev"

// whiteGlyph is the menu bar glyph colour.
var whiteGlyph = trayicon.RGB{R: 0xff, G: 0xff, B: 0xff}

// unknownDot is Apple's system grey: no status known yet.
var unknownDot = trayicon.RGB{R: 0x8e, G: 0x8e, B: 0x93}

// gatewayLabel names the active gateway of the openshell CLI, or says why it
// cannot be used. Until the tray polls the gateway it only shows that
// shellight finds the CLI config.
func gatewayLabel() string {
	c, err := gateway.Dial()
	if err != nil {
		return "Gateway: " + err.Error()
	}
	defer c.Close()
	return "Gateway: " + c.Name + " (" + c.Endpoint + ")"
}

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
	menu.Add("Status: unknown").SetEnabled(false)
	menu.Add(gatewayLabel()).SetEnabled(false)
	menu.AddSeparator()
	menu.Add("Build " + buildID).SetEnabled(false)
	menu.Add("Quit").OnClick(func(*application.Context) {
		app.Quit()
	})

	// No window is attached, so a click on the icon opens the menu.
	tray := app.SystemTray.New()
	tray.SetTooltip("shellight")
	tray.SetMenu(menu)

	// The icon is not a template image (the dot keeps its colour). The glyph
	// is always white; a setting to switch it is in docs/backlog.md.
	tray.SetIcon(trayicon.Badged(whiteGlyph, unknownDot))

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
