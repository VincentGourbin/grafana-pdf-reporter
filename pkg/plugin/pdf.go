package plugin

import (
	"bytes"
	"fmt"
	"image/png"
	"time"

	"github.com/jung-kurt/gofpdf"
)

const (
	a4LandscapeW = 297.0 // mm
	a4LandscapeH = 210.0 // mm
)

// buildReportPDF orchestre : crop PNG → assemble cover + page dashboard.
func buildReportPDF(pngBytes []byte, title, from, to, user, theme string) ([]byte, error) {
	// Étape 1 : crop le PNG pour retirer la zone vide bottom.
	cropped, err := cropToContent(pngBytes, theme)
	if err != nil {
		return nil, fmt.Errorf("crop: %w", err)
	}

	// Étape 2 : déterminer les dimensions natives du PNG croppé
	img, err := png.Decode(bytes.NewReader(cropped))
	if err != nil {
		return nil, fmt.Errorf("decode cropped: %w", err)
	}
	imgW := img.Bounds().Dx()
	imgH := img.Bounds().Dy()

	// Étape 3 : initialiser le PDF avec cover A4 paysage standard.
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: "L",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})

	// === COVER PAGE ===
	addCoverPage(pdf, title, from, to, user, theme)

	// === DASHBOARD PAGE ===
	// Page custom à la taille du PNG croppé pour zéro letterboxing.
	// 1 CSS px @ deviceScaleFactor=2 → 0.5 mm/sqrt physical (≈ scale_pt 0.375 in pts)
	// En mm : 1 phys px = 0.5 * 0.265 ≈ 0.132 mm (approx)
	// scale_mm = (1/96) * 25.4 / 2 = 0.132 (px-to-mm for dPR=2)
	const scaleMM = 25.4 / 96.0 / 2.0
	pageW := float64(imgW) * scaleMM
	pageH := float64(imgH) * scaleMM
	pdf.AddPageFormat("L", gofpdf.SizeType{Wd: pageW, Ht: pageH})

	// Background dark si nécessaire (au cas où le PNG aurait de la transparence)
	if theme == "dark" {
		pdf.SetFillColor(17, 18, 23) // #111217
		pdf.Rect(0, 0, pageW, pageH, "F")
	}

	// Embed PNG à la taille exacte de la page
	if err := pdf.RegisterImageOptionsReader(
		"dashboard.png",
		gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: false},
		bytes.NewReader(cropped),
	); err != nil && pdf.Err() {
		// gofpdf retourne un err noop si déjà enregistrée — on tolère
	}
	pdf.ImageOptions("dashboard.png", 0, 0, pageW, pageH, false,
		gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, 0, "")

	if pdf.Err() {
		return nil, pdf.Error()
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// addCoverPage génère la page de couverture sur le PDF en cours.
func addCoverPage(pdf *gofpdf.Fpdf, title, from, to, user, theme string) {
	pdf.AddPage()
	pw, ph := pdf.GetPageSize()

	// Palette thème
	var (
		bg, panel, accent              [3]int
		textMain, textDim              [3]int
	)
	if theme == "dark" {
		bg = [3]int{15, 17, 21}        // #0F1115
		panel = [3]int{27, 30, 37}     // #1B1E25
		accent = [3]int{16, 185, 129}  // #10B981
		textMain = [3]int{243, 244, 246} // #F3F4F6
		textDim = [3]int{156, 163, 175}  // #9CA3AF
	} else {
		bg = [3]int{255, 255, 255}
		panel = [3]int{243, 244, 246}
		accent = [3]int{5, 150, 105}
		textMain = [3]int{17, 24, 39}
		textDim = [3]int{107, 114, 128}
	}

	// Fond
	pdf.SetFillColor(bg[0], bg[1], bg[2])
	pdf.Rect(0, 0, pw, ph, "F")

	// Bande latérale gauche couleur accent
	pdf.SetFillColor(accent[0], accent[1], accent[2])
	pdf.Rect(0, 0, 12, ph, "F")

	// Brand header
	pdf.SetTextColor(textMain[0], textMain[1], textMain[2])
	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetXY(30, 25)
	pdf.Cell(0, 8, "Reachy Jardin")
	pdf.SetTextColor(textDim[0], textDim[1], textDim[2])
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(30, 33)
	pdf.Cell(0, 5, "Rapport d'observabilite Jetson Orin Nano")

	// Card centrale
	cardX, cardY := 30.0, 60.0
	cardW, cardH := pw-60, ph-100
	pdf.SetFillColor(panel[0], panel[1], panel[2])
	pdf.RoundedRect(cardX, cardY, cardW, cardH, 6, "1234", "F")

	// Titre du dashboard
	pdf.SetTextColor(textMain[0], textMain[1], textMain[2])
	pdf.SetFont("Helvetica", "B", 32)
	titleW := pdf.GetStringWidth(title)
	pdf.SetXY((pw-titleW)/2, cardY+cardH/2-20)
	pdf.Cell(titleW, 12, title)

	// Période
	pdf.SetFont("Helvetica", "", 14)
	pdf.SetTextColor(textDim[0], textDim[1], textDim[2])
	period := fmt.Sprintf("Periode  -  %s  >  %s", from, to)
	periodW := pdf.GetStringWidth(period)
	pdf.SetXY((pw-periodW)/2, cardY+cardH/2-2)
	pdf.Cell(periodW, 6, period)

	// Bloc info bas de card
	pdf.SetTextColor(textMain[0], textMain[1], textMain[2])
	pdf.SetFont("Helvetica", "", 11)
	now := time.Now().UTC().Format("2006-01-02 15:04 UTC")
	gen := fmt.Sprintf("Genere le %s", now)
	genW := pdf.GetStringWidth(gen)
	pdf.SetXY((pw-genW)/2, cardY+cardH/2+15)
	pdf.Cell(genW, 5, gen)
	if user != "" && user != "-" {
		byUser := fmt.Sprintf("par %s (via Grafana)", user)
		byW := pdf.GetStringWidth(byUser)
		pdf.SetXY((pw-byW)/2, cardY+cardH/2+22)
		pdf.Cell(byW, 5, byUser)
	}

	// Footer
	pdf.SetTextColor(textDim[0], textDim[1], textDim[2])
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetXY(30, ph-15)
	pdf.Cell(0, 4, "Confidentiel - ne pas redistribuer")
	pdf.SetXY(pw-90, ph-15)
	pdf.Cell(80, 4, "grafana-pdf-reporter")
}
