package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

// Settings est la conf injectée via l'API Plugin Settings (POST /api/plugins/
// <id>/settings). Les valeurs sensibles vont dans secureJsonData (chiffré DB),
// les autres dans jsonData.
type Settings struct {
	GrafanaURL       string
	GrafanaSAToken   string // depuis secureJsonData
	ImageRendererURL string
	RendererAuthTok  string // depuis secureJsonData
	ViewportWidth    int
	ViewportHeight   int
	RenderTimeout    time.Duration
}

// settingsFromContext lit les Settings depuis le PluginContext de la request.
// Cf. handler.go : on appelle ça AU MOMENT du traitement de chaque request.
func settingsFromContext(ctx context.Context) (Settings, error) {
	s := Settings{
		// Valeurs par défaut applicables sur un setup où Grafana+renderer
		// partagent le même host (cas target platform).
		GrafanaURL:       "https://127.0.0.1:3000",
		ImageRendererURL: "http://127.0.0.1:8181",
		ViewportWidth:    1280,
		ViewportHeight:   3000,
		RenderTimeout:    60 * time.Second,
	}
	pCtx := httpadapter.PluginConfigFromContext(ctx)
	// AppInstanceSettings est nil tant que le plugin n'a pas été enabled+settings push.
	if pCtx.AppInstanceSettings == nil {
		return s, nil
	}

	if raw := pCtx.AppInstanceSettings.JSONData; len(raw) > 0 {
		var j struct {
			GrafanaURL       string `json:"grafanaURL"`
			ImageRendererURL string `json:"imageRendererURL"`
			ViewportWidth    int    `json:"viewportWidth"`
			ViewportHeight   int    `json:"viewportHeight"`
			RenderTimeoutSec int    `json:"renderTimeoutSec"`
		}
		if err := json.Unmarshal(raw, &j); err != nil {
			return s, fmt.Errorf("decode JSONData: %w", err)
		}
		if j.GrafanaURL != "" {
			s.GrafanaURL = j.GrafanaURL
		}
		if j.ImageRendererURL != "" {
			s.ImageRendererURL = j.ImageRendererURL
		}
		if j.ViewportWidth > 0 {
			s.ViewportWidth = j.ViewportWidth
		}
		if j.ViewportHeight > 0 {
			s.ViewportHeight = j.ViewportHeight
		}
		if j.RenderTimeoutSec > 0 {
			s.RenderTimeout = time.Duration(j.RenderTimeoutSec) * time.Second
		}
	}

	if secure := pCtx.AppInstanceSettings.DecryptedSecureJSONData; secure != nil {
		s.GrafanaSAToken = secure["grafanaSAToken"]
		s.RendererAuthTok = secure["rendererAuthToken"]
	}
	return s, nil
}
