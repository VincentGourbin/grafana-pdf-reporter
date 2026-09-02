package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

// Settings est la conf injectée via l'API Plugin Settings (POST /api/plugins/
// <id>/settings). Les valeurs sensibles vont dans secureJsonData (chiffré DB),
// les autres dans jsonData.
type Settings struct {
	GrafanaURL     string
	GrafanaSAToken string // depuis secureJsonData
	MinRole        string // rôle Grafana minimal autorisé à exporter
	TLSSkipVerify  bool   // uniquement pour les certificats self-signed explicitement assumés
	TLSCACert      string // certificat(s) CA PEM non secret
	ViewportWidth  int    // override optionnel de la largeur de rendu (0 = largeur de la stratégie)
	MemLimitMiB    int    // limite mémoire Go par instance (0 = défaut du runtime)
	RenderTimeout  time.Duration
	// DeviceScaleFactor : facteur de résolution passé à image-renderer.
	// Quadratique en mémoire (2.0 = 4× pixels vs 1.0). Défaut 1.5 pour borner
	// l'empreinte du decode RGBA côté plugin (cf. issue OOM).
	DeviceScaleFactor float64

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
// le renderer partagent le même host. Sont aussi les défauts UI
// visibles dans la cover quand l'utilisateur n'a rien personnalisé.
func DefaultSettings() Settings {
	return Settings{
		GrafanaURL:         "http://localhost:3000",
		MinRole:            "Viewer",
		RenderTimeout:      60 * time.Second,
		DeviceScaleFactor:  1.5,
		CoverBrandTitle:    "Grafana",
		CoverBrandSubtitle: "Dashboard report",
		CoverFooterLeft:    "Confidential — do not redistribute",
		CoverFooterRight:   "grafana-pdf-reporter",
		CoverAccentHex:     "#10B981",
	}
}

type settingsJSON struct {
	GrafanaURL             string  `json:"grafanaURL"`
	MinRole                string  `json:"minRole"`
	TLSSkipVerify          bool    `json:"tlsSkipVerify"`
	TLSCACert              string  `json:"tlsCACert"`
	ViewportWidth          int     `json:"viewportWidth"`
	MemLimitMiB            int     `json:"memLimitMiB"`
	DeviceScaleFactor      float64 `json:"deviceScaleFactor"`
	RenderTimeoutSec       int     `json:"renderTimeoutSec"`
	CoverBrandTitle        string  `json:"coverBrandTitle"`
	CoverBrandSubtitle     string  `json:"coverBrandSubtitle"`
	CoverFooterLeft        string  `json:"coverFooterLeft"`
	CoverFooterRight       string  `json:"coverFooterRight"`
	CoverAccentHex         string  `json:"coverAccentHex"`
	CoverLogoDataURL       string  `json:"coverLogoDataURL"`
	CoverBackgroundDataURL string  `json:"coverBackgroundDataURL"`
}

func settingsFromJSON(raw []byte) (Settings, error) {
	s := DefaultSettings()
	if len(raw) == 0 {
		return s, nil
	}

	var j settingsJSON
	if err := json.Unmarshal(raw, &j); err != nil {
		return s, fmt.Errorf("decode JSONData: %w", err)
	}
	ifSetStr(&s.GrafanaURL, j.GrafanaURL)
	if j.MinRole == "Viewer" || j.MinRole == "Editor" || j.MinRole == "Admin" {
		s.MinRole = j.MinRole
	}
	s.TLSSkipVerify = j.TLSSkipVerify
	s.TLSCACert = j.TLSCACert
	ifSetInt(&s.ViewportWidth, j.ViewportWidth)
	ifSetInt(&s.MemLimitMiB, j.MemLimitMiB)
	if j.DeviceScaleFactor > 0 {
		s.DeviceScaleFactor = j.DeviceScaleFactor
	}
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
	return s, nil
}

func settingsFromInstance(is backend.AppInstanceSettings) (Settings, error) {
	s, err := settingsFromJSON(is.JSONData)
	if err != nil {
		return s, err
	}
	if secure := is.DecryptedSecureJSONData; secure != nil {
		s.GrafanaSAToken = secure["grafanaSAToken"]
	}
	return s, nil
}

// settingsFromContext lit les Settings depuis le PluginContext de la request.
// Cf. handler.go : on appelle ça AU MOMENT du traitement de chaque request.
func settingsFromContext(ctx context.Context) (Settings, error) {
	pCtx := httpadapter.PluginConfigFromContext(ctx)
	// AppInstanceSettings est nil tant que le plugin n'a pas été enabled+settings push.
	if pCtx.AppInstanceSettings == nil {
		return DefaultSettings(), nil
	}
	return settingsFromInstance(*pCtx.AppInstanceSettings)
}
