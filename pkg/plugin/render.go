package plugin

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

// La config (tokens, URLs) est désormais portée par Settings (cf config.go)
// et lue depuis le PluginContext de chaque request. Plus de vars de package.

const (
	// maxRenderBytes bounds a rendered PNG when Grafana sets no response limit.
	maxRenderBytes = 256 << 20
	// maxErrorBody bounds what is kept of a failed Grafana response.
	maxErrorBody = 512
)

// newHTTPClient builds the SDK client, which applies Grafana's configured
// response limit and proxy settings. Timeouts and TLS are set here because the
// SDK default of 30s is shorter than a render, and because it does not read
// the CA certificate from jsonData, where this plugin keeps it.
func newHTTPClient(ctx context.Context, s Settings, logger log.Logger) (*http.Client, error) {
	var opts httpclient.Options
	if app := httpadapter.PluginConfigFromContext(ctx).AppInstanceSettings; app != nil {
		var err error
		if opts, err = app.HTTPClientOptions(ctx); err != nil {
			return nil, fmt.Errorf("http client options: %w", err)
		}
	}
	if s.TLSSkipVerify {
		logger.Warn("TLS verification disabled (tlsSkipVerify=true)")
	}
	timeouts := httpclient.DefaultTimeoutOptions
	timeouts.Timeout = s.RenderTimeout + 15*time.Second
	opts.Timeouts = &timeouts
	opts.TLS = &httpclient.TLSOptions{
		CACertificate:      s.TLSCACert,
		InsecureSkipVerify: s.TLSSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}
	return httpclient.New(opts)
}

// grafanaStatusError is a non-200 answer from Grafana.
type grafanaStatusError struct {
	Code int
	Body string
}

func (e *grafanaStatusError) Error() string {
	return fmt.Sprintf("grafana HTTP %d: %s", e.Code, e.Body)
}

func statusError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	return &grafanaStatusError{Code: resp.StatusCode, Body: string(body)}
}

// grafanaGet decodes a JSON answer from Grafana. An empty token sends the
// request unauthenticated.
func grafanaGet(ctx context.Context, client *http.Client, s Settings, path, token string, out any) error {
	u := strings.TrimRight(s.GrafanaURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return statusError(resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// DashboardMeta = title + aspect calculé à partir du layout des panels.
// aspect = width / height en grid units (Grafana utilise 24 colonnes,
// chaque "row unit" = 30px par défaut). aspect <= 0 si inconnu.
type DashboardMeta struct {
	Title     string
	Aspect    float64
	FolderUID string // vide pour un dashboard à la racine
}

// fetchDashboardMeta interroge Grafana pour récupérer titre + aspect.
// Calcule l'aspect en sommant max(gridPos.y+h) et max(gridPos.x+w) sur les
// panels, y compris ceux contenus dans des rows collapsées (panels[].panels[]).
func fetchDashboardMeta(ctx context.Context, client *http.Client, s Settings, uid string) (DashboardMeta, error) {
	meta := DashboardMeta{Title: uid}
	type panel struct {
		GridPos struct {
			X int `json:"x"`
			Y int `json:"y"`
			W int `json:"w"`
			H int `json:"h"`
		} `json:"gridPos"`
		Panels []panel `json:"panels"`
	}
	var payload struct {
		Dashboard struct {
			Title  string  `json:"title"`
			Panels []panel `json:"panels"`
		} `json:"dashboard"`
		Meta struct {
			FolderUID string `json:"folderUid"`
		} `json:"meta"`
	}
	if err := grafanaGet(ctx, client, s, "/api/dashboards/uid/"+url.PathEscape(uid), s.GrafanaSAToken, &payload); err != nil {
		return meta, err
	}
	meta.FolderUID = payload.Meta.FolderUID
	if payload.Dashboard.Title != "" {
		meta.Title = payload.Dashboard.Title
	}

	maxX, maxY := 0, 0
	var walk func(ps []panel)
	walk = func(ps []panel) {
		for _, p := range ps {
			if rx := p.GridPos.X + p.GridPos.W; rx > maxX {
				maxX = rx
			}
			if ry := p.GridPos.Y + p.GridPos.H; ry > maxY {
				maxY = ry
			}
			if len(p.Panels) > 0 {
				walk(p.Panels)
			}
		}
	}
	walk(payload.Dashboard.Panels)
	if maxX > 0 && maxY > 0 {
		// Grafana grid : 24 colonnes (largeur fixe), unit h ≈ 30 CSS px,
		// unit w = container/24. Pour l'aspect, on compare directement les
		// quantités grid : aspect = colsUtilisées / rowsUtilisées.
		// On utilise 24 (largeur logique) plutôt que maxX, car les dashboards
		// "wide" remplissent rarement TOUTES les colonnes mais leur viewport
		// reste 1920 ; l'aspect doit refléter le ratio canvas.
		const cols = 24
		const rowPx = 30
		// width naturelle en CSS px ≈ cols * (1920/24) = 1920
		// height naturelle en CSS px ≈ maxY * rowPx
		meta.Aspect = (float64(cols) * (1920.0 / 24.0)) / (float64(maxY) * rowPx)
	}
	return meta, nil
}

// renderDashboardPNG demande à Grafana lui-même de rendre le dashboard via
// son endpoint natif /render/d/<uid>. Grafana gère l'authentification du
// renderer distant configuré dans [rendering] ; le plugin n'a donc pas à
// connaître l'URL ni le secret du renderer.
//
// height=-1 + fullPageImage=true (les mêmes paramètres que le bouton natif
// "Export as image" de Grafana) font capturer la hauteur RÉELLE du contenu
// du dashboard, quel que soit son nombre de panels. Une hauteur de viewport
// fixe par stratégie tronquait silencieusement tout dashboard dont le
// contenu dépassait cette hauteur (vérifié : un dashboard "square" de 8
// panels empilés était coupé aux 4 premiers avec une hauteur fixe de 1200px,
// capturé en entier avec l'auto-hauteur).
func renderDashboardPNG(ctx context.Context, client *http.Client, s Settings, uid, from, to, theme, tz string, viewportW int) ([]byte, error) {
	dsf := s.DeviceScaleFactor
	if dsf <= 0 {
		dsf = 1.5
	}
	v := url.Values{
		"from":                 {from},
		"to":                   {to},
		"theme":                {theme},
		"kiosk":                {"true"},
		"hideNav":              {"true"},
		"_dash.hideTimePicker": {"true"},
		"_dash.hideVariables":  {"true"},
		"width":                {strconv.Itoa(viewportW)},
		"height":               {"-1"},
		"fullPageImage":        {"true"},
		"scale":                {strconv.FormatFloat(dsf, 'f', -1, 64)},
		"timeout":              {strconv.Itoa(int(s.RenderTimeout.Seconds()))},
	}
	if tz != "" {
		// `timezone` is Grafana's native dashboard URL parameter. The frontend
		// calls the plugin-level value `tz` to keep its query compact.
		v.Set("timezone", tz)
	}
	renderURL := fmt.Sprintf("%s/render/d/%s/?%s",
		strings.TrimRight(s.GrafanaURL, "/"), uid, v.Encode())

	rctx, cancel := context.WithTimeout(ctx, s.RenderTimeout+15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(rctx, "GET", renderURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.GrafanaSAToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grafana render: %w", statusError(resp))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRenderBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read render: %w", err)
	}
	if len(body) > maxRenderBytes {
		return nil, errors.New("grafana render exceeds the maximum image size")
	}
	if len(body) < 8 || string(body[:4]) != "\x89PNG" {
		return nil, fmt.Errorf("grafana render returned non-PNG (first 30 bytes: %q)",
			string(body[:min(30, len(body))]))
	}
	return body, nil
}
