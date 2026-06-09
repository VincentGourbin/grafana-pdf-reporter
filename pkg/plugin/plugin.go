// Package plugin implémente l'app Grafana pdf-reporter.
//
// Architecture :
//   - Resource handler HTTP exposé via httpadapter sous
//     /api/plugins/<plugin-id>/resources/
//   - Route GET /generate?dashboard=<uid>[&from=<ts>&to=<ts>&theme=<dark|light>]
//     renvoie un PDF complet (cover + dashboard cropped).
//
// L'auth est gérée par Grafana avant que la requête n'atteigne le plugin :
// le user doit être connecté et avoir accès au dashboard.
package plugin

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

// App implémente backend.AppHandler.
type App struct {
	resourceHandler backend.CallResourceHandler
	logger          log.Logger
}

// NewApp est appelé par le SDK pour instancier l'app.
func NewApp(_ context.Context, _ backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	a := &App{logger: log.DefaultLogger}
	mux := http.NewServeMux()
	mux.HandleFunc("/generate", a.handleGenerate)
	mux.HandleFunc("/healthz", a.handleHealthz)
	a.resourceHandler = httpadapter.New(mux)
	return a, nil
}

// Dispose libère les ressources de l'instance (rien à faire ici).
func (a *App) Dispose() {}

// CheckHealth implémente backend.CheckHealthHandler.
func (a *App) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "OK",
	}, nil
}

// CallResource route les requêtes HTTP vers le mux interne.
func (a *App) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	return a.resourceHandler.CallResource(ctx, req, sender)
}

func (a *App) handleHealthz(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Set("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("ok\n"))
}
