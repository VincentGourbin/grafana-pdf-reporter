package plugin

import (
	"bytes"
	"image"
	"image/png"
)

// cropToContent détecte la dernière ligne pixel non-background et crop le PNG
// à ce bottom. Évite la zone vide sous le dernier panel quand on a rendu
// avec une viewport beaucoup plus haute que le dashboard naturel.
//
// Heuristique : on scanne de bas en haut et on s'arrête à la 1ère ligne qui a
// > seuil % de pixels significativement plus clairs que le background dark.
func cropToContent(pngBytes []byte, theme string) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Luminance background : dark theme ≈ 18, light theme ≈ 250.
	bgLum := uint8(18)
	if theme == "light" {
		bgLum = uint8(250)
	}
	threshold := int(bgLum) + 8  // marge bordures

	lastContentY := 0
	// Sample 1 pixel sur 4 horizontalement pour aller vite.
	for y := h - 1; y >= 0; y-- {
		lightCount := 0
		for x := 0; x < w; x += 4 {
			r, g, b, _ := img.At(x, y).RGBA()
			// RGBA returns 16-bit values, on prend les 8 bits hauts
			gray := (int(r>>8) + int(g>>8) + int(b>>8)) / 3
			diff := gray - threshold
			if theme == "light" {
				diff = threshold - gray
			}
			if diff > 0 {
				lightCount++
			}
		}
		if lightCount > w/40 {
			lastContentY = y + 1
			break
		}
	}

	if lastContentY == 0 || lastContentY >= h-2 {
		return pngBytes, nil
	}
	// Padding bottom = 8 px
	cropY := lastContentY + 8
	if cropY > h {
		cropY = h
	}

	subImg := image.NewRGBA(image.Rect(0, 0, w, cropY))
	// Copie pixel-par-pixel
	for y := 0; y < cropY; y++ {
		for x := 0; x < w; x++ {
			subImg.Set(x, y, img.At(x, y))
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, subImg); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
