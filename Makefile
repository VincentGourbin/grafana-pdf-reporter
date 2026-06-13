# Build pipeline pour grafana-pdf-reporter.
# Cible : produire `dist/` qui peut être copié dans /var/lib/grafana/plugins.

PLUGIN_ID := vincentgourbin-pdfreporter-app
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
TODAY     := $(shell date -u +"%Y-%m-%d")

# URLs où le plugin sera installé. La signature privée Grafana est bindée à
# cette liste — accéder à Grafana via une URL absente refusera de charger le
# plugin. Ajouter ici toutes les façons d'atteindre Grafana (LAN, FQDN, etc).
ROOT_URLS ?= https://grafana.example.com:8443/,https://grafana.example.com:3000/

# Tout passe par Docker pour ne pas exiger Go/Node sur le host.
GO_IMAGE  := golang:1.23
NODE_IMAGE := node:20-slim
GO_RUN    := docker run --rm -v "$$PWD:/work" -w /work -u "$$(id -u):$$(id -g)" -e HOME=/tmp -e GOCACHE=/tmp/.gocache -e GOMODCACHE=/tmp/.gomodcache $(GO_IMAGE)
NODE_RUN  := docker run --rm -v "$$PWD:/work" -w /work -u "$$(id -u):$$(id -g)" -e HOME=/tmp $(NODE_IMAGE)
# Le signing tool a besoin du token ; on le passe à Docker explicitement
# (pas via env_file car ça reste un secret local).
NODE_SIGN := docker run --rm -v "$$PWD:/work" -w /work -u "$$(id -u):$$(id -g)" -e HOME=/tmp -e GRAFANA_ACCESS_POLICY_TOKEN $(NODE_IMAGE)

.PHONY: all backend frontend sign clean help

help:
	@echo "Cibles : backend frontend all sign clean"
	@echo
	@echo "Signature (after backend+frontend):"
	@echo "  export GRAFANA_ACCESS_POLICY_TOKEN=glc_xxx  # cf README"
	@echo "  make sign"
	@echo "  # ou : make sign ROOT_URLS=https://your-host/"

all: backend frontend
	@echo "OK dist/ prêt"

# Cross-compile pour target platform (arm64) + dev local (amd64/arm64 Mac).
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
	# 3. Module frontend : on copie src/module.amd.js (écrit à la main en AMD).
	#    Grafana exige du AMD côté plugin loader ; esbuild ne sait pas faire
	#    de l'AMD, et webpack est lourd → AMD vanilla, 60 lignes.
	cp src/module.amd.js dist/module.js

# Signe le contenu de dist/ avec une signature PRIVATE bindée à $(ROOT_URLS).
# Produit dist/MANIFEST.txt — Grafana le vérifie au chargement et accepte
# alors de charger le plugin sans GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS.
sign:
	@if [ -z "$$GRAFANA_ACCESS_POLICY_TOKEN" ]; then \
		echo "ERROR: GRAFANA_ACCESS_POLICY_TOKEN n'est pas exporté."; \
		echo "Crée un token sur https://grafana.com/orgs/-/access-policies"; \
		echo "  scope: plugins:write, realm: <your org>"; \
		echo "Puis : export GRAFANA_ACCESS_POLICY_TOKEN=glc_..."; \
		exit 1; \
	fi
	@if [ ! -f dist/plugin.json ]; then \
		echo "ERROR: dist/ vide. Lance 'make all' d'abord."; \
		exit 1; \
	fi
	@echo "Signing for rootUrls: $(ROOT_URLS)"
	$(NODE_SIGN) npx --yes @grafana/sign-plugin@latest \
	  --rootUrls $(ROOT_URLS) \
	  --distDir dist
	@echo "OK signature dist/MANIFEST.txt généré."

clean:
	rm -rf dist/
