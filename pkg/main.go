// Entry point pour le backend du plugin Grafana pdf-reporter.
package main

import (
	"os"
	"runtime/debug"
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/backend/app"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"

	"github.com/VincentGourbin/grafana-pdf-reporter/pkg/plugin"
)

// applyMemoryLimit borne le heap Go pour rester sous la limite mémoire du
// container. Grafana lance le plugin comme sous-processus ; SANS GOMEMLIMIT le
// runtime ignore le cgroup, laisse croître le heap et rend la mémoire au
// système paresseusement → la RSS monte par paliers report après report
// jusqu'à l'OOM. Avec une limite, le GC devient agressif et la RSS reste bornée.
//
// Priorité : GOMEMLIMIT (honoré nativement par le runtime) > variable
// PDFREPORTER_MEM_LIMIT_MIB > défaut prudent 384 MiB.
func applyMemoryLimit() {
	if os.Getenv("GOMEMLIMIT") != "" {
		return // déjà fixé via l'env : le runtime l'applique seul.
	}
	limitMiB := 384
	if v := os.Getenv("PDFREPORTER_MEM_LIMIT_MIB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limitMiB = n
		}
	}
	debug.SetMemoryLimit(int64(limitMiB) << 20)
	log.DefaultLogger.Info("Go memory limit set", "mib", limitMiB)
}

func main() {
	applyMemoryLimit()
	if err := app.Manage(
		"vincentgourbin-pdfreporter-app",
		plugin.NewApp,
		app.ManageOpts{},
	); err != nil {
		log.DefaultLogger.Error("plugin manage failed", "error", err.Error())
		os.Exit(1)
	}
}
