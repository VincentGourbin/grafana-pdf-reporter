package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var uidRE = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

// sectionResult agrège ce qu'on extrait d'un dashboard pour produire 1 section.
type sectionResult struct {
	UID     string
	Title   string
	PNG     []byte
	Strat   Strategy
	Aspect  float64
	RenderMs int64
}

// renderOneSection : meta → strategy → render PNG.
// Pure : ne touche pas au PDF.
func renderOneSection(ctx context.Context, settings Settings, uid, from, to, theme, strategyOverride string) (*sectionResult, error) {
	meta, err := fetchDashboardMeta(ctx, settings, uid)
	if err != nil {
		return nil, fmt.Errorf("grafana meta %s: %w", uid, err)
	}
	strat := resolveStrategy(strategyOverride, meta.Aspect)
	t0 := time.Now()
	png, err := renderDashboardPNG(ctx, settings, uid, from, to, theme,
		strat.ViewportWidth, strat.ViewportHeight)
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", uid, err)
	}
	return &sectionResult{
		UID:      uid,
		Title:    meta.Title,
		PNG:      png,
		Strat:    strat,
		Aspect:   meta.Aspect,
		RenderMs: time.Since(t0).Milliseconds(),
	}, nil
}

// handleGenerate : 1 dashboard → PDF (cover + page).
func (a *App) handleGenerate(rw http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	uid := q.Get("dashboard")
	if !uidRE.MatchString(uid) {
		http.Error(rw, "invalid dashboard uid", http.StatusBadRequest)
		return
	}
	from := q.Get("from")
	if from == "" {
		from = "now-6h"
	}
	to := q.Get("to")
	if to == "" {
		to = "now"
	}
	theme := q.Get("theme")
	if theme != "dark" && theme != "light" {
		theme = "dark"
	}
	strategyOverride := q.Get("strategy")

	user := req.Header.Get("X-Grafana-User")
	a.logger.Info("generate request",
		"uid", uid, "from", from, "to", to, "theme", theme,
		"strategy", strategyOverride, "user", user)

	settings, err := settingsFromContext(req.Context())
	if err != nil {
		http.Error(rw, fmt.Sprintf("settings: %v", err), http.StatusInternalServerError)
		return
	}
	if settings.GrafanaSAToken == "" || settings.RendererAuthTok == "" {
		http.Error(rw, "plugin not fully configured", http.StatusServiceUnavailable)
		return
	}

	t0 := time.Now()
	sec, err := renderOneSection(req.Context(), settings, uid, from, to, theme, strategyOverride)
	if err != nil {
		a.logger.Error("render failed", "err", err.Error())
		http.Error(rw, err.Error(), http.StatusBadGateway)
		return
	}
	a.logger.Info("strategy",
		"name", sec.Strat.Name, "aspect", sec.Aspect,
		"orient", sec.Strat.PageOrient)

	pdfBytes, err := buildReportPDF(sec.PNG, sec.Title, from, to, user, theme, sec.Strat.PageOrient, coverFromSettings(settings))
	if err != nil {
		a.logger.Error("pdf build failed", "err", err.Error())
		http.Error(rw, fmt.Sprintf("pdf: %v", err), http.StatusInternalServerError)
		return
	}
	total := time.Since(t0).Milliseconds()

	fname := fmt.Sprintf("%s_%s.pdf", uid, time.Now().UTC().Format("20060102-1504"))
	rw.Header().Set("Content-Type", "application/pdf")
	rw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fname))
	rw.Header().Set("X-Render-Time-Ms", fmt.Sprintf("%d", sec.RenderMs))
	rw.Header().Set("X-Total-Time-Ms", fmt.Sprintf("%d", total))
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write(pdfBytes)

	a.logger.Info("generated",
		"uid", uid, "bytes", len(pdfBytes),
		"render_ms", sec.RenderMs, "total_ms", total)
}

// handleBundle accepte 2 formes :
//   - POST JSON {dashboards:[uid,...], from, to, theme, strategy}
//   - GET  ?dashboards=uid1,uid2&from=...&to=...&theme=...&strategy=...
//
// Le GET permet aux clients (typiquement nos liens UI) d'utiliser un simple
// window.open sans fetch async — robuste sur Safari iOS où l'enchaînement
// window.open("about:blank") + fetch + location.replace est filtré.
func (a *App) handleBundle(rw http.ResponseWriter, req *http.Request) {
	var payload struct {
		Dashboards []string `json:"dashboards"`
		From       string   `json:"from"`
		To         string   `json:"to"`
		Theme      string   `json:"theme"`
		Strategy   string   `json:"strategy"`
	}
	switch req.Method {
	case http.MethodPost:
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			http.Error(rw, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
	case http.MethodGet:
		q := req.URL.Query()
		raw := q.Get("dashboards")
		if raw != "" {
			for _, uid := range strings.Split(raw, ",") {
				uid = strings.TrimSpace(uid)
				if uid != "" {
					payload.Dashboards = append(payload.Dashboards, uid)
				}
			}
		}
		payload.From = q.Get("from")
		payload.To = q.Get("to")
		payload.Theme = q.Get("theme")
		payload.Strategy = q.Get("strategy")
	default:
		http.Error(rw, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	if len(payload.Dashboards) == 0 {
		http.Error(rw, "no dashboards selected", http.StatusBadRequest)
		return
	}
	if len(payload.Dashboards) > 25 {
		http.Error(rw, "too many dashboards (max 25)", http.StatusBadRequest)
		return
	}
	for _, uid := range payload.Dashboards {
		if !uidRE.MatchString(uid) {
			http.Error(rw, "invalid uid: "+uid, http.StatusBadRequest)
			return
		}
	}
	if payload.From == "" {
		payload.From = "now-6h"
	}
	if payload.To == "" {
		payload.To = "now"
	}
	if payload.Theme != "dark" && payload.Theme != "light" {
		payload.Theme = "dark"
	}

	user := req.Header.Get("X-Grafana-User")
	a.logger.Info("bundle request",
		"count", len(payload.Dashboards), "from", payload.From, "to", payload.To,
		"theme", payload.Theme, "strategy", payload.Strategy, "user", user)

	settings, err := settingsFromContext(req.Context())
	if err != nil {
		http.Error(rw, fmt.Sprintf("settings: %v", err), http.StatusInternalServerError)
		return
	}
	if settings.GrafanaSAToken == "" || settings.RendererAuthTok == "" {
		http.Error(rw, "plugin not fully configured", http.StatusServiceUnavailable)
		return
	}

	t0 := time.Now()
	pdf := newReportPDF()
	cov := coverFromSettings(settings)
	for idx, uid := range payload.Dashboards {
		sec, err := renderOneSection(req.Context(), settings, uid,
			payload.From, payload.To, payload.Theme, payload.Strategy)
		if err != nil {
			a.logger.Error("bundle section failed", "uid", uid, "err", err.Error())
			http.Error(rw, err.Error(), http.StatusBadGateway)
			return
		}
		if err := addReportSection(pdf, idx, sec.PNG, sec.Title, payload.From, payload.To,
			user, payload.Theme, sec.Strat.PageOrient, cov); err != nil {
			a.logger.Error("bundle add section failed", "uid", uid, "err", err.Error())
			http.Error(rw, fmt.Sprintf("pdf: %v", err), http.StatusInternalServerError)
			return
		}
		a.logger.Info("bundle section ok",
			"uid", uid, "title", sec.Title, "orient", sec.Strat.PageOrient,
			"render_ms", sec.RenderMs)
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		http.Error(rw, fmt.Sprintf("pdf output: %v", err), http.StatusInternalServerError)
		return
	}
	total := time.Since(t0).Milliseconds()

	fname := fmt.Sprintf("bundle_%s.pdf", time.Now().UTC().Format("20060102-1504"))
	rw.Header().Set("Content-Type", "application/pdf")
	rw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fname))
	rw.Header().Set("X-Total-Time-Ms", fmt.Sprintf("%d", total))
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write(out.Bytes())

	a.logger.Info("bundle generated",
		"count", len(payload.Dashboards), "bytes", out.Len(), "total_ms", total)
}
