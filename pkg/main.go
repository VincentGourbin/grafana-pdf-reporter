// Entry point pour le backend du plugin Grafana pdf-reporter.
package main

import (
	"math"
	"os"
	"runtime/debug"

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
// GOMEMLIMIT est honoré nativement par le runtime et reste prioritaire sur
// le défaut prudent de 384 MiB. Une valeur jsonData.memLimitMiB peut ensuite
// remplacer ce défaut au niveau d'une instance du plugin.
func applyMemoryLimit() {
	if debug.SetMemoryLimit(-1) != math.MaxInt64 {
		return // limite déjà fixée (GOMEMLIMIT) : on ne touche à rien.
	}
	const defaultMemLimitMiB = 384
	debug.SetMemoryLimit(int64(defaultMemLimitMiB) << 20)
	log.DefaultLogger.Info("Go memory limit set", "mib", defaultMemLimitMiB)
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
