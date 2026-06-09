package plugin

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Config par env vars — injectable depuis Grafana plugin config plus tard.
var (
	grafanaURL       = getenv("GF_PDFREPORTER_GRAFANA_URL", "https://127.0.0.1:3000")
	grafanaSAToken   = getenv("GF_PDFREPORTER_SA_TOKEN", "")
	imageRendererURL = getenv("GF_PDFREPORTER_IMAGE_RENDERER_URL", "http://127.0.0.1:8181")
	rendererAuthTok  = getenv("GF_PDFREPORTER_RENDERER_AUTH_TOKEN", "")
	viewportWidth    = getenvInt("GF_PDFREPORTER_VIEWPORT_WIDTH", 1280)
	viewportHeight   = getenvInt("GF_PDFREPORTER_VIEWPORT_HEIGHT", 3000)
	renderTimeout    = time.Duration(getenvInt("GF_PDFREPORTER_RENDER_TIMEOUT_SEC", 60)) * time.Second
)

// httpClient ignore TLS errors (Grafana self-signed sur 127.0.0.1).
var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
	Timeout: 90 * time.Second,
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
		return n
	}
	return def
}

// fetchDashboardTitle interroge Grafana pour récupérer le titre du dashboard.
func fetchDashboardTitle(ctx context.Context, uid string) (string, error) {
	if grafanaSAToken == "" {
		return uid, nil // pas d'auth configurée, retourne l'UID comme titre
	}
	u := fmt.Sprintf("%s/api/dashboards/uid/%s", strings.TrimRight(grafanaURL, "/"), uid)
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+grafanaSAToken)
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
func renderDashboardPNG(ctx context.Context, uid, from, to, theme string) ([]byte, error) {
	dashURL := fmt.Sprintf("%s/d/%s/?%s",
		strings.TrimRight(grafanaURL, "/"), uid,
		url.Values{
			"from":  {from},
			"to":    {to},
			"kiosk": {"1"},
			"theme": {theme},
			"orgId": {"1"},
		}.Encode(),
	)

	v := url.Values{
		"url":                {dashURL},
		"width":              {fmt.Sprintf("%d", viewportWidth)},
		"height":             {fmt.Sprintf("%d", viewportHeight)},
		"encoding":           {"png"},
		"deviceScaleFactor":  {"2.0"},
		"timeout":            {fmt.Sprintf("%d", int(renderTimeout.Seconds()))},
		"timezone":           {"Europe/Paris"},
	}
	renderURL := fmt.Sprintf("%s/render?%s",
		strings.TrimRight(imageRendererURL, "/"), v.Encode())

	rctx, cancel := context.WithTimeout(ctx, renderTimeout+15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(rctx, "GET", renderURL, nil)
	req.Header.Set("X-Auth-Token", rendererAuthTok)
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
