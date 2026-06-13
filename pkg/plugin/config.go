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

	// Cover branding (1 template global éditable via la page Settings).
	CoverBrandTitle        string
	CoverBrandSubtitle     string
	CoverFooterLeft        string
	CoverFooterRight       string
	CoverAccentHex         string // "#10B981" par défaut
	CoverLogoDataURL       string // "data:image/png;base64,..." ou vide
	CoverBackgroundDataURL string // image pleine page (généralement A4 paysage)
}

// ifSetStr écrit src dans *dst seulement si src ≠ "". Pratique pour "écrase
// le défaut uniquement si l'utilisateur a fourni une valeur".
func ifSetStr(dst *string, src string) {
	if src != "" {
		*dst = src
	}
}

// ifSetInt : idem pour les ints (zero = "non fourni").
func ifSetInt(dst *int, src int) {
	if src > 0 {
		*dst = src
	}
}

// DefaultSettings : valeurs par défaut applicables sur un setup où Grafana et
// le renderer partagent le même host (cas Jetson). Sont aussi les défauts UI
// visibles dans la cover quand l'utilisateur n'a rien personnalisé.
func DefaultSettings() Settings {
	return Settings{
		GrafanaURL:         "https://127.0.0.1:3000",
		ImageRendererURL:   "http://127.0.0.1:8181",
		ViewportWidth:      1280,
		ViewportHeight:     3000,
		RenderTimeout:      60 * time.Second,
		CoverBrandTitle:    "Grafana",
		CoverBrandSubtitle: "Dashboard report",
		CoverFooterLeft:    "Confidential — do not redistribute",
		CoverFooterRight:   "grafana-pdf-reporter",
		CoverAccentHex:     "#10B981",
	}
}

// settingsFromContext lit les Settings depuis le PluginContext de la request.
// Cf. handler.go : on appelle ça AU MOMENT du traitement de chaque request.
func settingsFromContext(ctx context.Context) (Settings, error) {
	s := DefaultSettings()
	pCtx := httpadapter.PluginConfigFromContext(ctx)
	// AppInstanceSettings est nil tant que le plugin n'a pas été enabled+settings push.
	if pCtx.AppInstanceSettings == nil {
		return s, nil
	}

	if raw := pCtx.AppInstanceSettings.JSONData; len(raw) > 0 {
		var j struct {
			GrafanaURL             string `json:"grafanaURL"`
			ImageRendererURL       string `json:"imageRendererURL"`
			ViewportWidth          int    `json:"viewportWidth"`
			ViewportHeight         int    `json:"viewportHeight"`
			RenderTimeoutSec       int    `json:"renderTimeoutSec"`
			CoverBrandTitle        string `json:"coverBrandTitle"`
			CoverBrandSubtitle     string `json:"coverBrandSubtitle"`
			CoverFooterLeft        string `json:"coverFooterLeft"`
			CoverFooterRight       string `json:"coverFooterRight"`
			CoverAccentHex         string `json:"coverAccentHex"`
			CoverLogoDataURL       string `json:"coverLogoDataURL"`
			CoverBackgroundDataURL string `json:"coverBackgroundDataURL"`
		}
		if err := json.Unmarshal(raw, &j); err != nil {
			return s, fmt.Errorf("decode JSONData: %w", err)
		}
		ifSetStr(&s.GrafanaURL, j.GrafanaURL)
		ifSetStr(&s.ImageRendererURL, j.ImageRendererURL)
		ifSetInt(&s.ViewportWidth, j.ViewportWidth)
		ifSetInt(&s.ViewportHeight, j.ViewportHeight)
		if j.RenderTimeoutSec > 0 {
			s.RenderTimeout = time.Duration(j.RenderTimeoutSec) * time.Second
		}
		ifSetStr(&s.CoverBrandTitle, j.CoverBrandTitle)
		ifSetStr(&s.CoverBrandSubtitle, j.CoverBrandSubtitle)
		ifSetStr(&s.CoverFooterLeft, j.CoverFooterLeft)
		ifSetStr(&s.CoverFooterRight, j.CoverFooterRight)
		ifSetStr(&s.CoverAccentHex, j.CoverAccentHex)
		// Logo et background acceptent "" comme "rien" (défaut vide), pas
		// besoin de garde — affectation directe.
		s.CoverLogoDataURL = j.CoverLogoDataURL
		s.CoverBackgroundDataURL = j.CoverBackgroundDataURL
	}

	if secure := pCtx.AppInstanceSettings.DecryptedSecureJSONData; secure != nil {
		s.GrafanaSAToken = secure["grafanaSAToken"]
		s.RendererAuthTok = secure["rendererAuthToken"]
	}
	return s, nil
}
