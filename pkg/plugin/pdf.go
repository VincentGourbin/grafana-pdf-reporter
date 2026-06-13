package plugin

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// CoverConfig regroupe les champs texte/visuels de la cover, alimentés
// depuis Settings (page Settings du plugin).
type CoverConfig struct {
	BrandTitle        string
	BrandSubtitle     string
	FooterLeft        string
	FooterRight       string
	AccentHex         string
	LogoDataURL       string
	BackgroundDataURL string // image pleine page, optionnelle
}

func coverFromSettings(s Settings) CoverConfig {
	return CoverConfig{
		BrandTitle:        s.CoverBrandTitle,
		BrandSubtitle:     s.CoverBrandSubtitle,
		FooterLeft:        s.CoverFooterLeft,
		FooterRight:       s.CoverFooterRight,
		AccentHex:         s.CoverAccentHex,
		LogoDataURL:       s.CoverLogoDataURL,
		BackgroundDataURL: s.CoverBackgroundDataURL,
	}
}

// hexToRGB parse "#RRGGBB" → [3]int{r,g,b}. Retourne fallback en cas d'échec.
func hexToRGB(hex string, fallback [3]int) [3]int {
	h := strings.TrimSpace(hex)
	h = strings.TrimPrefix(h, "#")
	if len(h) != 6 {
		return fallback
	}
	var r, g, b int
	if _, err := fmt.Sscanf(h, "%02x%02x%02x", &r, &g, &b); err != nil {
		return fallback
	}
	return [3]int{r, g, b}
}

// dataURLToImage parse "data:image/png;base64,..." → (bytes, mimeShortType).
// mimeShortType ∈ {"PNG", "JPG"} (compatible gofpdf ImageOptions). Retourne
// (nil, "") si la dataURL est vide / invalide.
func dataURLToImage(data string) ([]byte, string) {
	if data == "" {
		return nil, ""
	}
	if !strings.HasPrefix(data, "data:") {
		return nil, ""
	}
	comma := strings.Index(data, ",")
	if comma < 0 {
		return nil, ""
	}
	header := data[5:comma] // ex: "image/png;base64"
	payload := data[comma+1:]
	if !strings.Contains(header, "base64") {
		return nil, ""
	}
	mime := strings.SplitN(header, ";", 2)[0]
	var kind string
	switch mime {
	case "image/png":
		kind = "PNG"
	case "image/jpeg", "image/jpg":
		kind = "JPG"
	default:
		return nil, ""
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, ""
	}
	return raw, kind
}

// drawImage : helper centralisé pour Register + Place une image. `name` doit
// être unique sur l'instance Fpdf (sinon gofpdf retombe sur la première
// enregistrée sous ce nom). Erreur de Register est non-fatale : si l'image
// échoue à s'enregistrer, ImageOptions ne dessinera rien — mais on n'arrête
// pas la génération pour autant.
func drawImage(pdf *gofpdf.Fpdf, name, kind string, raw []byte, x, y, w, h float64) {
	opts := gofpdf.ImageOptions{ImageType: kind, ReadDpi: false}
	_ = pdf.RegisterImageOptionsReader(name, opts, bytes.NewReader(raw))
	pdf.ImageOptions(name, x, y, w, h, false, opts, 0, "")
}

// centeredLine écrit une ligne de texte centrée horizontalement, à la
// position y donnée. La fonte et la couleur doivent être réglées avant
// l'appel.
func centeredLine(pdf *gofpdf.Fpdf, y, lineH float64, text string) {
	pw, _ := pdf.GetPageSize()
	w := pdf.GetStringWidth(text)
	pdf.SetXY((pw-w)/2, y)
	pdf.Cell(w, lineH, text)
}

// newReportPDF construit un PDF vide, prêt à recevoir N sections via
// addReportSection. L'orientation par défaut n'a aucune importance puisque
// chaque AddPageFormat ré-impose la sienne.
func newReportPDF() *gofpdf.Fpdf {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: "L",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	pdf.SetAutoPageBreak(false, 0)
	return pdf
}

// addReportSection ajoute une cover + une page dashboard pour UN dashboard.
// pageOrient ∈ {"L", "P"}. `sectionIdx` doit être unique pour chaque appel
// sur le même *Fpdf — il sert à garantir un nom d'image unique côté gofpdf
// (sinon les sections 2+ recyclent l'image de la première car gofpdf cache
// les images par nom).
func addReportSection(pdf *gofpdf.Fpdf, sectionIdx int, pngBytes []byte, title, from, to, user, theme, pageOrient string, cov CoverConfig) error {
	cropped, imgW, imgH, err := cropToContent(pngBytes, theme)
	if err != nil {
		return fmt.Errorf("crop: %w", err)
	}

	if pageOrient != "P" && pageOrient != "L" {
		pageOrient = "L"
	}
	a4 := gofpdf.SizeType{Wd: 210, Ht: 297}

	// === COVER ===
	pdf.AddPageFormat(pageOrient, a4)
	drawCover(pdf, title, from, to, user, theme, cov)

	// === DASHBOARD ===
	pdf.AddPageFormat(pageOrient, a4)
	pw, ph := pdf.GetPageSize()
	if theme == "dark" {
		pdf.SetFillColor(17, 18, 23)
		pdf.Rect(0, 0, pw, ph, "F")
	}
	imgRatio := float64(imgW) / float64(imgH)
	pageRatio := pw / ph
	var drawW, drawH float64
	if imgRatio > pageRatio {
		drawW = pw
		drawH = pw / imgRatio
	} else {
		drawH = ph
		drawW = ph * imgRatio
	}
	x := (pw - drawW) / 2
	y := (ph - drawH) / 2

	drawImage(pdf, fmt.Sprintf("dash-%d.png", sectionIdx), "PNG", cropped,
		x, y, drawW, drawH)

	return pdf.Error()
}

// buildReportPDF : 1 dashboard = 1 section. Pratique pour /generate.
func buildReportPDF(pngBytes []byte, title, from, to, user, theme, pageOrient string, cov CoverConfig) ([]byte, error) {
	pdf := newReportPDF()
	if err := addReportSection(pdf, 0, pngBytes, title, from, to, user, theme, pageOrient, cov); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// drawCover dessine la cover : page déjà créée par l'appelant.
func drawCover(pdf *gofpdf.Fpdf, title, from, to, user, theme string, cov CoverConfig) {
	pw, ph := pdf.GetPageSize()
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	var (
		bg, panel         [3]int
		textMain, textDim [3]int
		defaultAccent     [3]int
	)
	if theme == "dark" {
		bg = [3]int{15, 17, 21}
		panel = [3]int{27, 30, 37}
		defaultAccent = [3]int{16, 185, 129}
		textMain = [3]int{243, 244, 246}
		textDim = [3]int{156, 163, 175}
	} else {
		bg = [3]int{255, 255, 255}
		panel = [3]int{243, 244, 246}
		defaultAccent = [3]int{5, 150, 105}
		textMain = [3]int{17, 24, 39}
		textDim = [3]int{107, 114, 128}
	}
	accent := hexToRGB(cov.AccentHex, defaultAccent)

	// Background : couleur unie, puis l'image custom par-dessus si fournie.
	pdf.SetFillColor(bg[0], bg[1], bg[2])
	pdf.Rect(0, 0, pw, ph, "F")
	if raw, kind := dataURLToImage(cov.BackgroundDataURL); raw != nil {
		// L'image qui ne respecte pas le ratio sera étirée — au user de
		// fournir un fichier au bon ratio.
		drawImage(pdf, "cover-bg", kind, raw, 0, 0, pw, ph)
	}

	pdf.SetFillColor(accent[0], accent[1], accent[2])
	pdf.Rect(0, 0, 12, ph, "F")

	// Logo (optionnel) — placé à gauche du titre de brand. Limite 18mm carré.
	logoRight := 30.0
	if raw, kind := dataURLToImage(cov.LogoDataURL); raw != nil {
		drawImage(pdf, "cover-logo", kind, raw, 30, 22, 18, 18)
		logoRight = 54.0
	}

	pdf.SetTextColor(textMain[0], textMain[1], textMain[2])
	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetXY(logoRight, 25)
	pdf.Cell(0, 8, tr(cov.BrandTitle))
	pdf.SetTextColor(textDim[0], textDim[1], textDim[2])
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(logoRight, 33)
	pdf.Cell(0, 5, tr(cov.BrandSubtitle))

	cardX, cardY := 30.0, 60.0
	cardW, cardH := pw-60, ph-100
	// Card semi-transparente : laisse voir le background custom dessous,
	// tout en gardant le texte lisible. Alpha 0.7 = compromis lisibilité/
	// transparence sympa pour des fonds peu contrastés.
	pdf.SetAlpha(0.7, "Normal")
	pdf.SetFillColor(panel[0], panel[1], panel[2])
	pdf.RoundedRect(cardX, cardY, cardW, cardH, 6, "1234", "F")
	pdf.SetAlpha(1.0, "Normal")

	// Lignes centrées dans la card.
	pdf.SetTextColor(textMain[0], textMain[1], textMain[2])
	pdf.SetFont("Helvetica", "B", 32)
	centeredLine(pdf, cardY+cardH/2-20, 12, tr(title))

	pdf.SetTextColor(textDim[0], textDim[1], textDim[2])
	pdf.SetFont("Helvetica", "", 14)
	centeredLine(pdf, cardY+cardH/2-2, 6,
		tr(fmt.Sprintf("Période  ·  %s  →  %s", from, to)))

	pdf.SetTextColor(textMain[0], textMain[1], textMain[2])
	pdf.SetFont("Helvetica", "", 11)
	now := time.Now().UTC().Format("2006-01-02 15:04 UTC")
	centeredLine(pdf, cardY+cardH/2+15, 5,
		tr(fmt.Sprintf("Généré le %s", now)))
	if user != "" && user != "-" {
		centeredLine(pdf, cardY+cardH/2+22, 5,
			tr(fmt.Sprintf("par %s (via Grafana)", user)))
	}

	pdf.SetTextColor(textDim[0], textDim[1], textDim[2])
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetXY(30, ph-15)
	pdf.Cell(0, 4, tr(cov.FooterLeft))
	pdf.SetXY(pw-90, ph-15)
	pdf.Cell(80, 4, tr(cov.FooterRight))
}
