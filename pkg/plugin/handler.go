package plugin

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

var uidRE = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

// sectionResult agrège ce qu'on extrait d'un dashboard pour produire 1 section.
type sectionResult struct {
	UID      string
	Title    string
	PNG      []byte
	Strat    Strategy
	Aspect   float64
	RenderMs int64
}

// renderOneSection : meta → strategy → render PNG. Pure.
func renderOneSection(ctx context.Context, client *http.Client, settings Settings, uid, from, to, theme, tz, strategyOverride string) (*sectionResult, error) {
	meta, err := fetchDashboardMeta(ctx, client, settings, uid)
	if err != nil {
		return nil, fmt.Errorf("grafana meta %s: %w", uid, err)
	}
	strat := resolveStrategy(strategyOverride, meta.Aspect)
	viewportW := strat.ViewportWidth
	if settings.ViewportWidth > 0 {
		viewportW = settings.ViewportWidth
	}
	t0 := time.Now()
	png, err := renderDashboardPNG(ctx, client, settings, uid, from, to, theme, tz, viewportW)
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

// reportParams = du normalisé pour les deux endpoints. Toute la validation /
// défauts vit ici (1 source de vérité, pas un copier-coller par handler).
type reportParams struct {
	UIDs     []string
	From     string
	To       string
	Theme    string
	TZ       string
	Strategy string
}

func roleRank(role string) int {
	switch role {
	case "Admin":
		return 3
	case "Editor":
		return 2
	case "Viewer":
		return 1
	default:
		return 0
	}
}

func callerFromContext(req *http.Request) (login, role string) {
	pCtx := httpadapter.PluginConfigFromContext(req.Context())
	if pCtx.User == nil {
		return "", ""
	}
	return pCtx.User.Login, pCtx.User.Role
}

// parseDashboardsParam : "uid1,uid2,uid3" → []string (trim + skip vide).
func parseDashboardsParam(raw string) []string {
	var out []string
	for _, uid := range strings.Split(raw, ",") {
		uid = strings.TrimSpace(uid)
		if uid != "" {
			out = append(out, uid)
		}
	}
	return out
}

// normalizeParams parse + valide la query string commune /generate et /bundle.
// Champs query : dashboards (CSV) ou dashboard (single), from, to, theme,
// strategy. Renvoie une erreur HTTP (status + message) si invalide.
func normalizeParams(q map[string][]string) (reportParams, int, error) {
	p := reportParams{
		From:     "now-6h",
		To:       "now",
		Theme:    "dark",
		Strategy: "",
	}
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}

	if csv := get("dashboards"); csv != "" {
		p.UIDs = parseDashboardsParam(csv)
	} else if single := get("dashboard"); single != "" {
		p.UIDs = []string{single}
	}
	if len(p.UIDs) == 0 {
		return p, http.StatusBadRequest, fmt.Errorf("no dashboards provided")
	}
	if len(p.UIDs) > 25 {
		return p, http.StatusBadRequest, fmt.Errorf("too many dashboards (max 25)")
	}
	for _, uid := range p.UIDs {
		if !uidRE.MatchString(uid) {
			return p, http.StatusBadRequest, fmt.Errorf("invalid uid: %s", uid)
		}
	}

	if v := get("from"); v != "" {
		p.From = v
	}
	if v := get("to"); v != "" {
		p.To = v
	}
	if v := get("theme"); v == "dark" || v == "light" {
		p.Theme = v
	}
	if v := get("tz"); len(v) <= 64 {
		p.TZ = v
	}
	p.Strategy = get("strategy")
	return p, 0, nil
}

// generateAndWrite : pipeline commun render N sections → PDF → écriture HTTP.
// Utilisé par /generate (1 uid) ET /bundle (N uids).
func (a *App) generateAndWrite(rw http.ResponseWriter, req *http.Request, p reportParams) {
	settings, err := settingsFromContext(req.Context())
	if err != nil {
		http.Error(rw, fmt.Sprintf("settings: %v", err), http.StatusInternalServerError)
		return
	}
	if settings.GrafanaSAToken == "" {
		http.Error(rw, "plugin not configured: set grafanaSAToken (secureJsonData)", http.StatusServiceUnavailable)
		return
	}
	select {
	case a.renderSem <- struct{}{}:
		defer func() { <-a.renderSem }()
	case <-time.After(30 * time.Second):
		http.Error(rw, "too many concurrent exports, retry later", http.StatusTooManyRequests)
		return
	case <-req.Context().Done():
		return
	}

	user, role := callerFromContext(req)
	if roleRank(role) < roleRank(settings.MinRole) {
		a.logger.Warn("export denied by minimum role", "user", user, "role", role, "required", settings.MinRole)
		http.Error(rw, "access denied: insufficient Grafana role", http.StatusForbidden)
		return
	}
	client := newHTTPClient(settings, a.logger)
	t0 := time.Now()
	pdf := newReportPDF()
	cov := coverFromSettings(settings)

	for idx, uid := range p.UIDs {
		sec, err := renderOneSection(req.Context(), client, settings, uid,
			p.From, p.To, p.Theme, p.TZ, p.Strategy)
		if err != nil {
			a.logger.Error("section failed", "uid", uid, "err", err.Error())
			http.Error(rw, err.Error(), http.StatusBadGateway)
			return
		}
		a.logger.Info("section ok",
			"uid", uid, "title", sec.Title, "orient", sec.Strat.PageOrient,
			"aspect", sec.Aspect, "strategy", sec.Strat.Name,
			"render_ms", sec.RenderMs)
		if err := addReportSection(pdf, idx, sec.PNG, sec.Title, p.From, p.To,
			user, p.Theme, sec.Strat.PageOrient, cov); err != nil {
			a.logger.Error("add section failed", "uid", uid, "err", err.Error())
			http.Error(rw, fmt.Sprintf("pdf: %v", err), http.StatusInternalServerError)
			return
		}
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		http.Error(rw, fmt.Sprintf("pdf output: %v", err), http.StatusInternalServerError)
		return
	}
	total := time.Since(t0).Milliseconds()

	// Nom de fichier : <uid> seul si 1 dashboard, sinon "bundle".
	stem := "bundle"
	if len(p.UIDs) == 1 {
		stem = p.UIDs[0]
	}
	fname := fmt.Sprintf("%s_%s.pdf", stem, time.Now().UTC().Format("20060102-1504"))
	rw.Header().Set("Content-Type", "application/pdf")
	rw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fname))
	rw.Header().Set("X-Total-Time-Ms", fmt.Sprintf("%d", total))
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write(out.Bytes())

	a.logger.Info("report generated",
		"count", len(p.UIDs), "bytes", out.Len(), "total_ms", total, "user", user)
}

// handleGenerate : 1 dashboard via query (?dashboard=<uid>...).
// Conservé pour rétro-compat avec d'éventuels liens directs ; le frontend
// route tout via /bundle.
func (a *App) handleGenerate(rw http.ResponseWriter, req *http.Request) {
	p, status, err := normalizeParams(req.URL.Query())
	if err != nil {
		http.Error(rw, err.Error(), status)
		return
	}
	if len(p.UIDs) != 1 {
		http.Error(rw, "use /bundle for multi-dashboard exports", http.StatusBadRequest)
		return
	}
	a.generateAndWrite(rw, req, p)
}

// handleBundle : N dashboards via query (?dashboards=uid1,uid2,...).
// GET-only car le frontend utilise un simple window.open (= compatible avec
// les bloqueurs de popup Safari iOS qui filtrent about:blank+fetch async).
func (a *App) handleBundle(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, "GET only", http.StatusMethodNotAllowed)
		return
	}
	p, status, err := normalizeParams(req.URL.Query())
	if err != nil {
		http.Error(rw, err.Error(), status)
		return
	}
	a.generateAndWrite(rw, req, p)
}
