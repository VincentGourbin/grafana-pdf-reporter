package plugin

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
)

// cropToContent détecte la bounding box du contenu non-background et crop
// le PNG sur les 4 côtés. Retourne aussi les dimensions finales pour éviter
// un png.Decode supplémentaire côté appelant.
//
// Implémentation : un seul passage O(W·H) qui suit min/max simultanément.
// `image/draw` fait la copie en mémoire optimisée (vs. setpixel loop Go).
func cropToContent(pngBytes []byte, theme string) (data []byte, dx, dy int, err error) {
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, 0, 0, err
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
	// minContentPixels pixels significativement différents du bg.
	const minContentPixels = 5

	rowCount := make([]int, h)
	colCount := make([]int, w)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if isContent(x, y) {
				rowCount[y]++
				colCount[x]++
			}
		}
	}

	top, bottom := 0, h
	for y := 0; y < h; y++ {
		if rowCount[y] >= minContentPixels {
			top = y
			break
		}
	}
	for y := h - 1; y >= 0; y-- {
		if rowCount[y] >= minContentPixels {
			bottom = y + 1
			break
		}
	}
	left, right := 0, w
	for x := 0; x < w; x++ {
		if colCount[x] >= minContentPixels {
			left = x
			break
		}
	}
	for x := w - 1; x >= 0; x-- {
		if colCount[x] >= minContentPixels {
			right = x + 1
			break
		}
	}

	// Si le contenu remplit déjà toute l'image, ou aucun contenu détecté,
	// retourne tel quel.
	if right <= left || bottom <= top {
		return pngBytes, w, h, nil
	}
	if top == 0 && left == 0 && right == w && bottom == h {
		return pngBytes, w, h, nil
	}

	// Padding pour respirer.
	const pad = 8
	top = max(0, top-pad)
	left = max(0, left-pad)
	bottom = min(h, bottom+pad)
	right = min(w, right+pad)

	cw, ch := right-left, bottom-top
	cropped := image.NewRGBA(image.Rect(0, 0, cw, ch))
	draw.Draw(cropped, cropped.Bounds(), img, image.Pt(left, top), draw.Src)

	var out bytes.Buffer
	if err := png.Encode(&out, cropped); err != nil {
		return nil, 0, 0, err
	}
	return out.Bytes(), cw, ch, nil
}
