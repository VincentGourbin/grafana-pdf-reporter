# Standard Grafana plugin build pipeline for grafana-pdf-reporter.
# Produces `dist/`, ready to copy into /var/lib/grafana/plugins.

PLUGIN_ID := vincentgourbin-pdfreporter-app

# URLs où le plugin sera installé. La signature privée Grafana est bindée à
# cette liste — accéder à Grafana via une URL absente refusera de charger le
# plugin. Ne pas mettre de valeur d'infrastructure par défaut dans le dépôt.
ROOT_URLS ?=

# Tout passe par Docker pour ne pas exiger Go/Node sur le host.
# Bootstrap image already available on constrained build hosts. GOTOOLCHAIN
# fetches and runs the pinned current toolchain inside the container.
GO_IMAGE  := golang:1.26
NODE_IMAGE := node:22
GO_RUN    := docker run --rm -v "$$PWD:/work" -w /work -u "$$(id -u):$$(id -g)" -e HOME=/tmp -e GOCACHE=/tmp/.gocache -e GOMODCACHE=/tmp/.gomodcache -e GOTOOLCHAIN=go1.26.7+auto $(GO_IMAGE)
NODE_RUN  := docker run --rm -v "$$PWD:/work" -w /work -u "$$(id -u):$$(id -g)" -e HOME=/tmp $(NODE_IMAGE)
# Le signing tool a besoin du token ; on le passe à Docker explicitement
# (pas via env_file car ça reste un secret local).
NODE_SIGN := docker run --rm -v "$$PWD:/work" -w /work -u "$$(id -u):$$(id -g)" -e HOME=/tmp -e GRAFANA_ACCESS_POLICY_TOKEN $(NODE_IMAGE)

.PHONY: all backend frontend sign validate clean help

help:
	@echo "Targets: backend frontend all validate sign clean"
	@echo
	@echo "Signature (after backend+frontend):"
	@echo "  export GRAFANA_ACCESS_POLICY_TOKEN=glc_xxx  # cf README"
	@echo "  make sign"
	@echo "  # ou : make sign ROOT_URLS=https://your-host/"

all: backend frontend
	@echo "OK dist/ prêt"

# Cross-compile pour les plateformes supportées (Linux arm64/amd64,
# macOS arm64 et Windows amd64).
backend:
	@mkdir -p dist
	$(GO_RUN) sh -c "go run github.com/magefile/mage@v1.15.0 -v buildAll"

frontend:
	$(NODE_RUN) sh -c "npm ci && npm run build"

validate:
	rm -rf release && mkdir -p release/staging/$(PLUGIN_ID)
	cp -a dist/. release/staging/$(PLUGIN_ID)/
	cd release/staging && zip -qr ../$(PLUGIN_ID)-$$(node -p "require('../../package.json').version").zip $(PLUGIN_ID)
	mkdir -p release/source
	rsync -a --exclude '.git/' --exclude 'node_modules/' --exclude 'dist/' --exclude 'release/' --exclude '.eslintcache' --exclude 'PLAN-*.md' ./ release/source/
	docker run --rm --pull=always --platform linux/amd64 -e GOTOOLCHAIN=auto -v "$$PWD:/work" -w /work grafana/plugin-validator-cli \
	  -sourceCodeUri file:///work/release/source/ /work/release/$(PLUGIN_ID)-$$(node -p "require('./package.json').version").zip

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
	@if [ -z "$(ROOT_URLS)" ]; then \
		echo "ERROR: ROOT_URLS n'est pas défini pour la signature."; \
		echo "Exemple : make sign ROOT_URLS=https://grafana.example.com/"; \
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
	rm -rf dist/ node_modules/.cache .eslintcache
