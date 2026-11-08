// Command shellight is the shellight menu bar app for NVIDIA OpenShell.
package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

// buildID is set at link time (scripts/build-darwin.sh) to tell builds apart.
var buildID = "dev"

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
	menu.AddSeparator()
	menu.Add("Build " + buildID).SetEnabled(false)
	menu.Add("Quit").OnClick(func(*application.Context) {
		app.Quit()
	})

	// No window is attached, so a click on the icon opens the menu.
	tray := app.SystemTray.New()
	tray.SetTooltip("shellight")
	tray.SetTemplateIcon(icons.SystrayMacTemplate)
	tray.SetMenu(menu)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
