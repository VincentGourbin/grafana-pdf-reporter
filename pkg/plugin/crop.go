package plugin

import (
	"bytes"
	"image"
	"image/png"
)

// cropToContent détecte la bounding box du contenu non-background et crop
// le PNG sur les 4 côtés. Évite tout letterboxing (bandes vides à droite,
// en bas, ou n'importe où) quand on a rendu avec une viewport plus large
// que le dashboard naturel.
func cropToContent(pngBytes []byte, theme string) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Luminance background : dark ≈ 18, light ≈ 250.
	bgLum := 18
	if theme == "light" {
		bgLum = 250
	}
	// Un pixel "content" doit être à > delta niveaux du background.
	const delta = 30

	isContent := func(x, y int) bool {
		r, g, b, _ := img.At(x, y).RGBA()
		gray := (int(r>>8) + int(g>>8) + int(b>>8)) / 3
		var diff int
		if theme == "dark" {
			diff = gray - bgLum
		} else {
			diff = bgLum - gray
		}
		return diff > delta
	}

	// Pour qu'une row/col compte comme "contenu", on demande au moins
	// 5 pixels significativement différents du bg.
	const minContentPixels = 5

	rowHasContent := func(y int) bool {
		n := 0
		for x := 0; x < w; x++ {
			if isContent(x, y) {
				n++
				if n >= minContentPixels {
					return true
				}
			}
		}
		return false
	}
	colHasContent := func(x int) bool {
		n := 0
		for y := 0; y < h; y++ {
			if isContent(x, y) {
				n++
				if n >= minContentPixels {
					return true
				}
			}
		}
		return false
	}

	// Top : scan vers le bas, première row avec contenu.
	top := 0
	for y := 0; y < h; y++ {
		if rowHasContent(y) {
			top = y
			break
		}
	}
	// Bottom : scan vers le haut, dernière row avec contenu.
	bottom := h
	for y := h - 1; y >= 0; y-- {
		if rowHasContent(y) {
			bottom = y + 1
			break
		}
	}
	// Left : scan vers la droite, première col avec contenu.
	left := 0
	for x := 0; x < w; x++ {
		if colHasContent(x) {
			left = x
			break
		}
	}
	// Right : scan vers la gauche, dernière col avec contenu.
	right := w
	for x := w - 1; x >= 0; x-- {
		if colHasContent(x) {
			right = x + 1
			break
		}
	}

	// Si le contenu remplit déjà toute l'image, retourne tel quel.
	if top == 0 && left == 0 && right == w && bottom == h {
		return pngBytes, nil
	}
	// Sanity : si on détecte 0 contenu, retourne tel quel.
	if right <= left || bottom <= top {
		return pngBytes, nil
	}

	// Padding 8 px de chaque côté pour respirer.
	const pad = 8
	top = maxInt(0, top-pad)
	left = maxInt(0, left-pad)
	bottom = minInt(h, bottom+pad)
	right = minInt(w, right+pad)

	cropped := image.NewRGBA(image.Rect(0, 0, right-left, bottom-top))
	for y := top; y < bottom; y++ {
		for x := left; x < right; x++ {
			cropped.Set(x-left, y-top, img.At(x, y))
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, cropped); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
