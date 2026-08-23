package plugin

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// La config (tokens, URLs) est désormais portée par Settings (cf config.go)
// et lue depuis le PluginContext de chaque request. Plus de vars de package.

// newHTTPClient construit le client HTTP selon la configuration TLS.
// La vérification est stricte par défaut. Le contournement self-signed est
// explicite, borné à cette connexion et signalé dans les logs.
func newHTTPClient(s Settings, logger log.Logger) *http.Client {
	tlsCfg := &tls.Config{}
	if s.TLSSkipVerify {
		tlsCfg.InsecureSkipVerify = true // #nosec G402 -- option explicite d'administration
		logger.Warn("TLS verification disabled (tlsSkipVerify=true)")
	}
	if s.TLSCACert != "" {
		pool := x509.NewCertPool()
		if pool.AppendCertsFromPEM([]byte(s.TLSCACert)) {
			tlsCfg.RootCAs = pool
		} else {
			logger.Error("tlsCACert provided but not parseable, ignoring")
		}
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
			Proxy:           http.ProxyFromEnvironment,
		},
		Timeout: 90 * time.Second,
	}
}

// DashboardMeta = title + aspect calculé à partir du layout des panels.
// aspect = width / height en grid units (Grafana utilise 24 colonnes,
// chaque "row unit" = 30px par défaut). aspect <= 0 si inconnu.
type DashboardMeta struct {
	Title  string
	Aspect float64
}

// fetchDashboardMeta interroge Grafana pour récupérer titre + aspect.
// Calcule l'aspect en sommant max(gridPos.y+h) et max(gridPos.x+w) sur les
// panels, y compris ceux contenus dans des rows collapsées (panels[].panels[]).
func fetchDashboardMeta(ctx context.Context, client *http.Client, s Settings, uid string) (DashboardMeta, error) {
	meta := DashboardMeta{Title: uid}
	if s.GrafanaSAToken == "" {
		return meta, nil
	}
	u := fmt.Sprintf("%s/api/dashboards/uid/%s", strings.TrimRight(s.GrafanaURL, "/"), uid)
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+s.GrafanaSAToken)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return meta, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return meta, fmt.Errorf("grafana HTTP %d: %s", resp.StatusCode, string(body)[:min(200, len(body))])
	}
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
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return meta, err
	}
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
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("grafana render HTTP %d: %s",
			resp.StatusCode, string(body)[:min(200, len(body))])
	}
	if len(body) < 8 || string(body[:4]) != "\x89PNG" {
		return nil, fmt.Errorf("grafana render returned non-PNG (first 30 bytes: %q)",
			string(body[:min(30, len(body))]))
	}
	return body, nil
}
