package plugin

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// fetchDashboardTitle interroge Grafana pour récupérer le titre du dashboard.
func fetchDashboardTitle(ctx context.Context, s Settings, uid string) (string, error) {
	if s.GrafanaSAToken == "" {
		return uid, nil // pas d'auth configurée, retourne l'UID comme titre
	}
	u := fmt.Sprintf("%s/api/dashboards/uid/%s", strings.TrimRight(s.GrafanaURL, "/"), uid)
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+s.GrafanaSAToken)
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return uid, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return uid, fmt.Errorf("grafana HTTP %d: %s", resp.StatusCode, string(body)[:min(200, len(body))])
	}
	var payload struct {
		Dashboard struct {
			Title string `json:"title"`
		} `json:"dashboard"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return uid, err
	}
	if payload.Dashboard.Title == "" {
		return uid, nil
	}
	return payload.Dashboard.Title, nil
}

// renderDashboardPNG appelle image-renderer pour produire un PNG du dashboard.
func renderDashboardPNG(ctx context.Context, s Settings, uid, from, to, theme string) ([]byte, error) {
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

	v := url.Values{
		"url":               {dashURL},
		"width":             {fmt.Sprintf("%d", s.ViewportWidth)},
		"height":            {fmt.Sprintf("%d", s.ViewportHeight)},
		"encoding":          {"png"},
		"deviceScaleFactor": {"2.0"},
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
