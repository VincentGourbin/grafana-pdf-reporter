package plugin

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

var uidRE = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

// handleGenerate orchestre : render PNG → crop → assemble PDF avec cover.
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

	user := req.Header.Get("X-Grafana-User")
	a.logger.Info("generate request",
		"uid", uid, "from", from, "to", to, "theme", theme, "user", user)

	// Étape 1 : récupérer le titre du dashboard (et plus tard ses dimensions).
	title, err := fetchDashboardTitle(req.Context(), uid)
	if err != nil {
		a.logger.Error("fetch dashboard meta failed", "err", err.Error())
		http.Error(rw, fmt.Sprintf("grafana meta: %v", err), http.StatusBadGateway)
		return
	}

	// Étape 2 : render PNG via image-renderer (en passant par Grafana /render/d).
	t0 := time.Now()
	pngBytes, err := renderDashboardPNG(req.Context(), uid, from, to, theme)
	if err != nil {
		a.logger.Error("render failed", "err", err.Error())
		http.Error(rw, fmt.Sprintf("render: %v", err), http.StatusBadGateway)
		return
	}
	renderDt := time.Since(t0)

	// Étape 3 : crop bottom + wrap PDF avec cover.
	pdfBytes, err := buildReportPDF(pngBytes, title, from, to, user, theme)
	if err != nil {
		a.logger.Error("pdf build failed", "err", err.Error())
		http.Error(rw, fmt.Sprintf("pdf: %v", err), http.StatusInternalServerError)
		return
	}
	totalDt := time.Since(t0)

	fname := fmt.Sprintf("%s_%s.pdf", uid, time.Now().UTC().Format("20060102-1504"))
	rw.Header().Set("Content-Type", "application/pdf")
	rw.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, fname))
	rw.Header().Set("X-Render-Time-Ms", fmt.Sprintf("%d", renderDt.Milliseconds()))
	rw.Header().Set("X-Total-Time-Ms", fmt.Sprintf("%d", totalDt.Milliseconds()))
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write(pdfBytes)

	a.logger.Info("generated",
		"uid", uid, "bytes", len(pdfBytes),
		"render_ms", renderDt.Milliseconds(), "total_ms", totalDt.Milliseconds())

	_ = bytes.NewReader(nil) // silence unused import si on retire les usages
}
