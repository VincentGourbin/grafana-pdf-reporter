package plugin

// Strategy regroupe les choix de rendu : taille viewport renderer + orientation
// de la page A4 finale. On en a 3, sélectionnés selon le ratio largeur/hauteur
// du dashboard ("aspect"), avec override possible côté caller (query param).
type Strategy struct {
	Name           string
	ViewportWidth  int
	ViewportHeight int
	PageOrient     string // "L" ou "P"
}

func strategyLandscape() Strategy {
	return Strategy{Name: "landscape", ViewportWidth: 1920, ViewportHeight: 1080, PageOrient: "L"}
}
func strategySquare() Strategy {
	return Strategy{Name: "square", ViewportWidth: 1600, ViewportHeight: 1200, PageOrient: "L"}
}
func strategyPortrait() Strategy {
	return Strategy{Name: "portrait", ViewportWidth: 1280, ViewportHeight: 2400, PageOrient: "P"}
}

// resolveStrategy choisit la stratégie à utiliser.
// override ∈ {"", "auto", "landscape", "square", "portrait"} (vide = auto).
// aspect = width / height "naturel" du dashboard (depuis Grafana gridPos).
// aspect <= 0 signifie inconnu → fallback landscape (le cas le plus fréquent).
func resolveStrategy(override string, aspect float64) Strategy {
	switch override {
	case "landscape":
		return strategyLandscape()
	case "square":
		return strategySquare()
	case "portrait":
		return strategyPortrait()
	}
	if aspect <= 0 {
		return strategyLandscape()
	}
	switch {
	case aspect >= 1.5:
		return strategyLandscape()
	case aspect <= 0.7:
		return strategyPortrait()
	default:
		return strategySquare()
	}
}
