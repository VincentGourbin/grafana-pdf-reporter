// Entry point pour le backend du plugin Grafana pdf-reporter.
package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/app"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"

	"github.com/VincentGourbin/grafana-pdf-reporter/pkg/plugin"
)

func main() {
	if err := app.Manage(
		"vincentgourbin-pdfreporter-app",
		plugin.NewApp,
		app.ManageOpts{},
	); err != nil {
		log.DefaultLogger.Error("plugin manage failed", "error", err.Error())
		os.Exit(1)
	}
}
