# Build pipeline pour grafana-pdf-reporter.
# Cible : produire `dist/` qui peut être copié dans /var/lib/grafana/plugins.

PLUGIN_ID := vincentgourbin-pdfreporter-app
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
TODAY     := $(shell date -u +"%Y-%m-%d")

# Tout passe par Docker pour ne pas exiger Go/Node sur le host.
GO_IMAGE  := golang:1.23
NODE_IMAGE := node:20-slim
GO_RUN    := docker run --rm -v "$$PWD:/work" -w /work -u "$$(id -u):$$(id -g)" -e HOME=/tmp -e GOCACHE=/tmp/.gocache -e GOMODCACHE=/tmp/.gomodcache $(GO_IMAGE)
NODE_RUN  := docker run --rm -v "$$PWD:/work" -w /work -u "$$(id -u):$$(id -g)" -e HOME=/tmp $(NODE_IMAGE)

.PHONY: all backend frontend clean help

help:
	@echo "Cibles : backend frontend all clean"

all: backend frontend
	@echo "OK dist/ prêt"

# Cross-compile pour Jetson (arm64) + dev local (amd64/arm64 Mac).
backend:
	@mkdir -p dist
	$(GO_RUN) sh -c "\
	  CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags '-s -w -X main.version=$(VERSION)' -o dist/gpx_pdfreporter_linux_arm64   ./pkg && \
	  CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags '-s -w -X main.version=$(VERSION)' -o dist/gpx_pdfreporter_linux_amd64   ./pkg && \
	  CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags '-s -w -X main.version=$(VERSION)' -o dist/gpx_pdfreporter_darwin_arm64  ./pkg \
	"
	# Le binaire utilisé par Grafana au runtime est résolu selon `executable` du plugin.json
	# Grafana ajoute le suffixe _<os>_<arch> automatiquement.

frontend:
	@mkdir -p dist/img
	# 1. plugin.json (avec placeholders remplacés)
	cp src/plugin.json dist/plugin.json
	sed -i.bak "s|%VERSION%|$(VERSION)|g; s|%TODAY%|$(TODAY)|g" dist/plugin.json && rm dist/plugin.json.bak
	# 2. Logo (placeholder SVG simple si pas de logo officiel)
	[ -f src/img/logo.svg ] && cp src/img/logo.svg dist/img/logo.svg || \
	  printf '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48"><rect width="48" height="48" rx="6" fill="#10B981"/><text x="24" y="32" text-anchor="middle" fill="white" font-family="sans-serif" font-weight="bold" font-size="20">PDF</text></svg>\n' > dist/img/logo.svg
	# 3. Bundle module.ts via esbuild (image Docker, user non-root).
	# Grafana 11+ accepte les ES modules pour les plugins.
	$(NODE_RUN) sh -c "\
	  npx --yes esbuild src/module.ts \
	    --bundle --format=esm --platform=browser --target=es2020 \
	    --external:@grafana/data --external:@grafana/runtime --external:@grafana/ui --external:react --external:react-dom \
	    --outfile=dist/module.js \
	"

clean:
	rm -rf dist/
