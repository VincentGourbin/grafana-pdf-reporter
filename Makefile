# Build pipeline pour grafana-pdf-reporter.
# Cible : produire `dist/` qui peut être copié dans /var/lib/grafana/plugins.

PLUGIN_ID := vincentgourbin-pdfreporter-app
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
TODAY     := $(shell date -u +"%Y-%m-%d")

.PHONY: all backend frontend clean help

help:
	@echo "Cibles : backend frontend all clean"

all: backend frontend
	@echo "OK dist/ prêt"

# Cross-compile pour Jetson (arm64) + dev local (amd64/arm64 Mac).
backend:
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/gpx_pdfreporter_linux_arm64   ./pkg
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/gpx_pdfreporter_linux_amd64   ./pkg
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/gpx_pdfreporter_darwin_arm64  ./pkg
	# Le binaire utilisé par Grafana au runtime est résolu selon `executable` du plugin.json
	# Grafana ajoute le suffixe _<os>_<arch> automatiquement.

frontend:
	@mkdir -p dist
	# Frontend = juste le module.ts compilé + plugin.json + assets.
	# En attendant le webpack stack, on copie tel quel + on remplace les placeholders.
	cp src/plugin.json dist/plugin.json
	sed -i.bak "s|%VERSION%|$(VERSION)|g; s|%TODAY%|$(TODAY)|g" dist/plugin.json && rm dist/plugin.json.bak
	# TODO: bundler le src/module.ts via webpack ou esbuild.

clean:
	rm -rf dist/
