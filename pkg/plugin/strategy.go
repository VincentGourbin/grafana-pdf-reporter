package plugin

// Strategy regroupe les choix de rendu : largeur de viewport renderer +
// orientation de la page A4 finale. La hauteur de capture est toujours
// automatique côté Grafana (cf. render.go) — elle s'adapte au contenu réel
// du dashboard, donc seule la largeur varie par stratégie. On en a 3,
// sélectionnées selon le ratio largeur/hauteur du dashboard ("aspect"), avec
// override possible côté caller (query param).
type Strategy struct {
	Name          string
	ViewportWidth int
	PageOrient    string // "L" ou "P"
}

func strategyLandscape() Strategy {
	return Strategy{Name: "landscape", ViewportWidth: 1920, PageOrient: "L"}
}
func strategySquare() Strategy {
	return Strategy{Name: "square", ViewportWidth: 1600, PageOrient: "L"}
}
func strategyPortrait() Strategy {
	return Strategy{Name: "portrait", ViewportWidth: 1280, PageOrient: "P"}
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
