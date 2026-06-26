package plugin

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// La config (tokens, URLs) est désormais portée par Settings (cf config.go)
// et lue depuis le PluginContext de chaque request. Plus de vars de package.

// httpClient ignore TLS errors (Grafana self-signed sur 127.0.0.1).
var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
	Timeout: 90 * time.Second,
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
func fetchDashboardMeta(ctx context.Context, s Settings, uid string) (DashboardMeta, error) {
	meta := DashboardMeta{Title: uid}
	if s.GrafanaSAToken == "" {
		return meta, nil
	}
	u := fmt.Sprintf("%s/api/dashboards/uid/%s", strings.TrimRight(s.GrafanaURL, "/"), uid)
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+s.GrafanaSAToken)
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
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

// renderDashboardPNG appelle image-renderer pour produire un PNG du dashboard.
// La viewport (width, height) est celle de la stratégie choisie ; les autres
// paramètres viennent des Settings.
func renderDashboardPNG(ctx context.Context, s Settings, uid, from, to, theme string, viewportW, viewportH int) ([]byte, error) {
	dashURL := fmt.Sprintf("%s/d/%s/?%s",
		strings.TrimRight(s.GrafanaURL, "/"), uid,
		url.Values{
			"from":  {from},
			"to":    {to},
			"kiosk": {"1"},
			"theme": {theme},
			"orgId": {"1"},
		}.Encode(),
	)

	// deviceScaleFactor est quadratique en mémoire (decode RGBA côté plugin) :
	// 2.0 = 4× pixels vs 1.0. On borne au défaut 1.5 si non/mal configuré.
	dsf := s.DeviceScaleFactor
	if dsf <= 0 {
		dsf = 1.5
	}
	v := url.Values{
		"url":               {dashURL},
		"width":             {fmt.Sprintf("%d", viewportW)},
		"height":            {fmt.Sprintf("%d", viewportH)},
		"encoding":          {"png"},
		"deviceScaleFactor": {strconv.FormatFloat(dsf, 'f', -1, 64)},
		"timeout":           {fmt.Sprintf("%d", int(s.RenderTimeout.Seconds()))},
		"timezone":          {"Europe/Paris"},
	}
	renderURL := fmt.Sprintf("%s/render?%s",
		strings.TrimRight(s.ImageRendererURL, "/"), v.Encode())

	rctx, cancel := context.WithTimeout(ctx, s.RenderTimeout+15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(rctx, "GET", renderURL, nil)
	req.Header.Set("X-Auth-Token", s.RendererAuthTok)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("image-renderer HTTP %d: %s",
			resp.StatusCode, string(body)[:min(200, len(body))])
	}
	if len(body) < 8 || string(body[:4]) != "\x89PNG" {
		return nil, fmt.Errorf("image-renderer returned non-PNG (first 30 bytes: %q)",
			string(body[:min(30, len(body))]))
	}
	return body, nil
}
