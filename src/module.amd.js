// Module frontend en AMD pur. Utilise les composants @grafana/ui pour
// hériter du thème courant (clair/sombre) et du look natif Grafana.
//
//   - Page d'app accessible via le menu de gauche.
//   - Lien Cmd+K "Export PDF" : si on est sur un dashboard, exporte
//     directement ; sinon redirige vers la page de l'app.
//
// Toutes les chaînes UI sont externalisées dans STRINGS (fr/en),
// langue auto-détectée via navigator.language.

define([
  "react",
  "@grafana/data",
  "@grafana/ui",
  "@grafana/runtime",
], function (React, gData, gUI, gRT) {
  "use strict";

  var AppPlugin = gData.AppPlugin;
  var dateTime = gData.dateTime;
  var h = React.createElement;
  var PLUGIN_ID = "vincentgourbin-pdfreporter-app";

  // Composants Grafana — feature-detect au cas où une version manque.
  var PluginPage = gUI.PluginPage || gRT.PluginPage || null;
  var Button = gUI.Button;
  var LinkButton = gUI.LinkButton || gUI.Button;
  var Field = gUI.Field;
  var Input = gUI.Input;
  var Checkbox = gUI.Checkbox;
  var RadioButtonGroup = gUI.RadioButtonGroup;
  var Alert = gUI.Alert;
  var Spinner = gUI.Spinner;
  var TimeRangePicker = gUI.TimeRangePicker;
  var Stack = gUI.Stack;
  var Icon = gUI.Icon;

  // ---------- i18n ----------

  var SUPPORTED_LANGS = ["fr", "en"];
  var STRINGS = {
    fr: {
      page_title: "Export PDF",
      page_subtitle:
        "Sélectionne un ou plusieurs dashboards. Multi-sélection = un seul PDF concaténé, dans l'ordre de la liste.",
      // Settings (cover branding)
      settings_btn: "Paramètres",
      settings_back: "Retour à la liste",
      settings_title: "Personnalisation de la page de garde",
      settings_subtitle:
        "Ces valeurs sont utilisées sur la page de couverture de chaque rapport.",
      brand_title: "Titre principal",
      brand_subtitle: "Sous-titre",
      footer_left: "Pied de page gauche",
      footer_right: "Pied de page droit",
      accent_hex: "Couleur d'accent",
      logo: "Logo",
      logo_pick: "Choisir un fichier (PNG/JPEG, max 200 Ko)",
      logo_remove: "Retirer",
      background: "Fond de la page de garde",
      background_pick: "Choisir un fichier (PNG/JPEG, max 2 Mo)",
      background_remove: "Retirer le fond",
      background_too_large: "Image de fond trop volumineuse (max 2 Mo).",
      prompt_title: "Prompt pour générer un fond",
      prompt_help:
        "Remplace [YOUR CONCEPT HERE] par ton idée (ex. \"abstract topographic terrain\", \"sparse circuit board lines\"), puis copie tout dans DALL-E, FLUX, Imagen… La couleur d'accent et le thème sont déjà intégrés.",
      copy_prompt: "Copier le prompt",
      copied: "Prompt copié !",
      preview: "Aperçu de la couverture",
      save: "Enregistrer",
      saving: "Enregistrement…",
      saved: "Paramètres enregistrés.",
      save_error: "Erreur d'enregistrement : ",
      logo_too_large: "Logo trop volumineux (max 200 Ko).",
      forbidden:
        "Tu n'as pas le droit de modifier la configuration du plugin (rôle Admin requis).",
      period: "Période",
      from: "Depuis",
      to: "Jusqu'à",
      theme: "Thème",
      theme_dark: "Sombre",
      theme_light: "Clair",
      strategy: "Mise en page",
      strategy_help:
        "Auto choisit selon la forme du dashboard. Forcer une orientation si tu veux uniformiser.",
      strategy_auto: "Auto",
      strategy_landscape: "Paysage",
      strategy_square: "Carré",
      strategy_portrait: "Portrait",
      select_all: "Tout sélectionner",
      clear_all: "Tout désélectionner",
      selected_count_zero: "Aucun dashboard sélectionné",
      selected_count_one: "1 dashboard sélectionné",
      selected_count_many: "{n} dashboards sélectionnés",
      export_btn: "Exporter en PDF",
      exporting: "Génération…",
      open_link: "Ouvrir",
      loading: "Chargement…",
      empty: "Aucun dashboard accessible.",
    },
    en: {
      page_title: "Export PDF",
      page_subtitle:
        "Select one or more dashboards. Multi-select = a single concatenated PDF, in list order.",
      settings_btn: "Settings",
      settings_back: "Back to list",
      settings_title: "Cover page branding",
      settings_subtitle: "These values are used on every report cover page.",
      brand_title: "Brand title",
      brand_subtitle: "Subtitle",
      footer_left: "Footer (left)",
      footer_right: "Footer (right)",
      accent_hex: "Accent color",
      logo: "Logo",
      logo_pick: "Choose a file (PNG/JPEG, max 200 KB)",
      logo_remove: "Remove",
      background: "Cover background",
      background_pick: "Choose a file (PNG/JPEG, max 2 MB)",
      background_remove: "Remove background",
      background_too_large: "Background image too large (max 2 MB).",
      prompt_title: "Image-gen prompt",
      prompt_help:
        "Replace [YOUR CONCEPT HERE] with your idea (e.g. \"abstract topographic terrain\", \"sparse circuit board lines\"), then paste into DALL-E, FLUX, Imagen… Your accent color and theme are already baked in.",
      copy_prompt: "Copy prompt",
      copied: "Prompt copied!",
      preview: "Cover preview",
      save: "Save",
      saving: "Saving…",
      saved: "Settings saved.",
      save_error: "Save failed: ",
      logo_too_large: "Logo too large (max 200 KB).",
      forbidden:
        "You don't have permission to edit plugin settings (Admin role required).",
      period: "Time range",
      from: "From",
      to: "To",
      theme: "Theme",
      theme_dark: "Dark",
      theme_light: "Light",
      strategy: "Layout",
      strategy_help:
        "Auto picks based on each dashboard shape. Force one to unify all sections.",
      strategy_auto: "Auto",
      strategy_landscape: "Landscape",
      strategy_square: "Square",
      strategy_portrait: "Portrait",
      select_all: "Select all",
      clear_all: "Clear",
      selected_count_zero: "No dashboard selected",
      selected_count_one: "1 dashboard selected",
      selected_count_many: "{n} dashboards selected",
      export_btn: "Export PDF",
      exporting: "Generating…",
      open_link: "Open",
      loading: "Loading…",
      empty: "No dashboards available.",
    },
  };

  function pickLang() {
    var nav =
      (typeof navigator !== "undefined" &&
        (navigator.language || navigator.userLanguage)) ||
      "en";
    var code = String(nav).toLowerCase().slice(0, 2);
    return SUPPORTED_LANGS.indexOf(code) >= 0 ? code : "en";
  }
  var LANG = pickLang();
  function t(key, vars) {
    var table = STRINGS[LANG] || STRINGS.en;
    var s =
      table[key] != null
        ? table[key]
        : STRINGS.en[key] != null
          ? STRINGS.en[key]
          : key;
    if (vars) {
      Object.keys(vars).forEach(function (k) {
        s = s.replace("{" + k + "}", vars[k]);
      });
    }
    return s;
  }

  // ---------- Thème suivi sur Grafana ----------
  // On lit le thème courant du user via @grafana/runtime.config et on s'en
  // sert directement pour image-renderer. Pas d'UI à exposer.

  function currentGrafanaTheme() {
    try {
      if (gRT && gRT.config) {
        var c = gRT.config;
        if (c.theme2 && typeof c.theme2.isDark === "boolean") {
          return c.theme2.isDark ? "dark" : "light";
        }
        if (c.bootData && c.bootData.user && c.bootData.user.lightTheme) {
          return "light";
        }
        if (c.theme && c.theme.type) {
          return c.theme.type === "light" ? "light" : "dark";
        }
      }
    } catch (e) {}
    return "dark";
  }

  // ---------- TimeRange helpers ----------

  function makeTimeRange(fromRaw, toRaw) {
    // TimeRangePicker attend un TimeRange complet (from/to DateTime + raw).
    // Pour les expressions relatives ("now-6h", "now/d"), on résout via
    // rangeUtil si dispo ; sinon on met dateTime() comme placeholder.
    var resolved = null;
    try {
      if (gData.rangeUtil && gData.rangeUtil.convertRawToRange) {
        resolved = gData.rangeUtil.convertRawToRange({
          from: fromRaw,
          to: toRaw,
        });
      }
    } catch (e) {}
    if (resolved && resolved.from && resolved.to) return resolved;
    var fromDt = dateTime ? dateTime(fromRaw) : new Date();
    var toDt = dateTime ? dateTime(toRaw) : new Date();
    return {
      from: fromDt,
      to: toDt,
      raw: { from: fromRaw, to: toRaw },
    };
  }

  // Décale la fenêtre en gardant sa durée. Direction = -1 (back) ou +1 (fwd).
  // Sortie : raw devient un couple de chaînes ISO (absolu), pas dateTime ;
  // c'est ce que Grafana fait après un shift et c'est ce que les viewers
  // affichent proprement.
  function shiftTimeRange(tr, direction) {
    try {
      var from = tr.from && tr.from.valueOf ? tr.from.valueOf() : Date.now();
      var to = tr.to && tr.to.valueOf ? tr.to.valueOf() : Date.now();
      var span = to - from;
      var newFrom = from + direction * span;
      var newTo = to + direction * span;
      return {
        from: dateTime(newFrom),
        to: dateTime(newTo),
        raw: {
          from: new Date(newFrom).toISOString(),
          to: new Date(newTo).toISOString(),
        },
      };
    } catch (e) {
      return tr;
    }
  }
  // Élargit la fenêtre en doublant sa durée autour du centre.
  function zoomTimeRange(tr) {
    try {
      var from = tr.from && tr.from.valueOf ? tr.from.valueOf() : Date.now();
      var to = tr.to && tr.to.valueOf ? tr.to.valueOf() : Date.now();
      var span = to - from;
      var center = (from + to) / 2;
      var newFrom = center - span;
      var newTo = center + span;
      return {
        from: dateTime(newFrom),
        to: dateTime(newTo),
        raw: {
          from: new Date(newFrom).toISOString(),
          to: new Date(newTo).toISOString(),
        },
      };
    } catch (e) {
      return tr;
    }
  }
  // Pour la sortie HTTP : si raw est encore une chaîne ("now-6h"), on la
  // garde ; sinon on sérialise l'ISO.
  function rangeToRawStrings(tr) {
    var r = (tr && tr.raw) || {};
    function ser(v, fallbackDt) {
      if (typeof v === "string") return v;
      if (v && typeof v.toISOString === "function") return v.toISOString();
      if (fallbackDt && typeof fallbackDt.toISOString === "function")
        return fallbackDt.toISOString();
      return "now";
    }
    return { from: ser(r.from, tr.from), to: ser(r.to, tr.to) };
  }

  // ---------- Export : ouvre dans un nouvel onglet ----------
  //
  // Stratégie SIMPLE et qui survit aux bloqueurs de popup iOS Safari /
  // private mode : un seul window.open(url, "_blank") synchrone, vers
  // un endpoint GET. Le serveur répond avec Content-Disposition:
  // attachment → le browser affiche le PDF / lance le download.

  function openUrlNewTab(url) {
    // window.open dans un click handler synchrone = autorisé par tous
    // les bloqueurs de popup (c'est une intention user explicite).
    var w = window.open(url, "_blank");
    if (!w) {
      // Si quand même bloqué, fallback : crée un <a target="_blank">
      // et click programmatique (sera ouvert comme un lien click).
      var a = document.createElement("a");
      a.href = url;
      a.target = "_blank";
      a.rel = "noopener";
      a.style.display = "none";
      document.body.appendChild(a);
      a.click();
      a.remove();
    }
  }

  function exportSingle(uid) {
    var q = new URLSearchParams();
    q.set("dashboard", uid);
    q.set("theme", currentGrafanaTheme());
    var url =
      "/api/plugins/" + PLUGIN_ID + "/resources/generate?" + q.toString();
    openUrlNewTab(url);
  }
  function uidFromContext(ctx) {
    if (ctx && ctx.dashboard && ctx.dashboard.uid) return ctx.dashboard.uid;
    var m = window.location.pathname.match(/\/d\/([^/]+)/);
    return m ? m[1] : null;
  }

  // ---------- Page ----------

  // ---------- Root (view dispatcher : list ↔ settings) ----------

  function RootPage() {
    var v = React.useState("list");
    var view = v[0];
    var setView = v[1];
    if (view === "settings") {
      return h(SettingsView, {
        onBack: function () {
          setView("list");
        },
      });
    }
    return h(ListView, {
      onOpenSettings: function () {
        setView("settings");
      },
    });
  }

  // ---------- Liste des dashboards ----------

  function ListView(props) {
    var s = React.useState({
      loading: true,
      error: null,
      dashboards: [],
      selected: {},
      timeRange: makeTimeRange("now-6h", "now"),
      timeZone: "browser",
      submitting: false,
      lastError: null,
    });
    var state = s[0];
    var setState = s[1];

    React.useEffect(function () {
      fetch("/api/search?type=dash-db&limit=500", { credentials: "include" })
        .then(function (r) {
          if (!r.ok) throw new Error("HTTP " + r.status);
          return r.json();
        })
        .then(function (list) {
          setState(function (p) {
            return Object.assign({}, p, { loading: false, dashboards: list });
          });
        })
        .catch(function (e) {
          setState(function (p) {
            return Object.assign({}, p, { loading: false, error: String(e) });
          });
        });
    }, []);

    function toggle(uid) {
      setState(function (p) {
        var sel = Object.assign({}, p.selected);
        if (sel[uid]) delete sel[uid];
        else sel[uid] = true;
        return Object.assign({}, p, { selected: sel });
      });
    }
    function selectAll() {
      setState(function (p) {
        var sel = {};
        p.dashboards.forEach(function (d) {
          sel[d.uid] = true;
        });
        return Object.assign({}, p, { selected: sel });
      });
    }
    function clearAll() {
      setState(function (p) {
        return Object.assign({}, p, { selected: {} });
      });
    }
    function setField(key, v) {
      setState(function (p) {
        var n = {};
        n[key] = v;
        return Object.assign({}, p, n);
      });
    }

    function doExport() {
      var uids = Object.keys(state.selected);
      if (uids.length === 0) return;
      var raw = rangeToRawStrings(state.timeRange);
      var q = new URLSearchParams();
      q.set("dashboards", uids.join(","));
      q.set("from", raw.from);
      q.set("to", raw.to);
      q.set("theme", currentGrafanaTheme());
      q.set("strategy", "auto");
      var url =
        "/api/plugins/" + PLUGIN_ID + "/resources/bundle?" + q.toString();
      openUrlNewTab(url);
    }

    var selectedCount = Object.keys(state.selected).length;
    var countLabel =
      selectedCount === 0
        ? t("selected_count_zero")
        : selectedCount === 1
          ? t("selected_count_one")
          : t("selected_count_many", { n: selectedCount });

    // Section options : juste la période. Thème = suit Grafana, stratégie
    // = auto (choisie par dashboard côté backend).
    var optionsSection = h(
      Field,
      { label: t("period") },
      TimeRangePicker
        ? h(TimeRangePicker, {
            value: state.timeRange,
            timeZone: state.timeZone,
            onChange: function (tr) {
              setField("timeRange", tr);
            },
            onChangeTimeZone: function (tz) {
              setField("timeZone", tz);
            },
            onMoveBackward: function () {
              setField("timeRange", shiftTimeRange(state.timeRange, -1));
            },
            onMoveForward: function () {
              setField("timeRange", shiftTimeRange(state.timeRange, +1));
            },
            onZoom: function () {
              setField("timeRange", zoomTimeRange(state.timeRange));
            },
          })
        : h(
            "div",
            { style: { display: "flex", gap: "8px" } },
            h(Input, {
              value: String(state.timeRange.raw.from),
              onChange: function (e) {
                setField(
                  "timeRange",
                  makeTimeRange(e.target.value, state.timeRange.raw.to),
                );
              },
            }),
            h(Input, {
              value: String(state.timeRange.raw.to),
              onChange: function (e) {
                setField(
                  "timeRange",
                  makeTimeRange(state.timeRange.raw.from, e.target.value),
                );
              },
            }),
          ),
    );

    // Toolbar (select all / count / export)
    var toolbar = h(
      "div",
      {
        style: {
          display: "flex",
          alignItems: "center",
          gap: "8px",
          marginBottom: "12px",
          flexWrap: "wrap",
        },
      },
      h(
        Button,
        {
          variant: "secondary",
          size: "sm",
          onClick: selectAll,
        },
        t("select_all"),
      ),
      h(
        Button,
        {
          variant: "secondary",
          fill: "outline",
          size: "sm",
          onClick: clearAll,
        },
        t("clear_all"),
      ),
      h(
        "span",
        { style: { marginLeft: "auto", opacity: 0.7 } },
        countLabel,
      ),
      h(
        Button,
        {
          variant: "primary",
          icon: state.submitting ? undefined : "file-alt",
          disabled: selectedCount === 0 || state.submitting,
          onClick: doExport,
        },
        state.submitting ? t("exporting") : t("export_btn"),
      ),
    );

    // Liste dashboards (Grafana Checkbox + style sobre)
    var listSection;
    if (state.loading) {
      listSection = h(
        "div",
        { style: { display: "flex", gap: "8px", alignItems: "center" } },
        Spinner ? h(Spinner, null) : null,
        h("span", null, t("loading")),
      );
    } else if (state.error) {
      listSection = h(
        Alert,
        { title: state.error, severity: "error" },
        null,
      );
    } else if (state.dashboards.length === 0) {
      listSection = h(
        "div",
        { style: { padding: "16px", opacity: 0.7 } },
        t("empty"),
      );
    } else {
      listSection = h(
        "div",
        {
          style: {
            border: "1px solid var(--border-weak, #2a2c33)",
            borderRadius: "4px",
            overflow: "hidden",
          },
        },
        state.dashboards.map(function (d) {
          var checked = !!state.selected[d.uid];
          return h(
            "div",
            {
              key: d.uid,
              onClick: function () {
                toggle(d.uid);
              },
              style: {
                display: "flex",
                alignItems: "center",
                gap: "12px",
                padding: "10px 14px",
                borderBottom: "1px solid var(--border-weak, #2a2c33)",
                background: checked
                  ? "rgba(61,113,217,0.08)"
                  : "transparent",
                cursor: "pointer",
              },
            },
            h(Checkbox, {
              value: checked,
              onChange: function () {
                toggle(d.uid);
              },
            }),
            h(
              "div",
              { style: { flex: 1, minWidth: 0 } },
              h(
                "div",
                {
                  style: {
                    fontWeight: 500,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  },
                },
                d.title || d.uid,
              ),
              d.folderTitle
                ? h(
                    "div",
                    { style: { fontSize: "0.85em", opacity: 0.7 } },
                    d.folderTitle,
                  )
                : null,
            ),
            h(
              LinkButton,
              {
                href: "/d/" + d.uid,
                target: "_blank",
                rel: "noopener",
                variant: "secondary",
                fill: "text",
                size: "sm",
                icon: "external-link-alt",
                onClick: function (e) {
                  e.stopPropagation();
                },
              },
              t("open_link"),
            ),
          );
        }),
      );
    }

    var content = h(
      React.Fragment,
      null,
      h(
        "div",
        {
          style: {
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            gap: "16px",
            marginBottom: "8px",
          },
        },
        h("p", { style: { opacity: 0.8, margin: 0 } }, t("page_subtitle")),
        h(
          Button,
          {
            variant: "secondary",
            fill: "outline",
            size: "sm",
            icon: "cog",
            onClick: props.onOpenSettings,
          },
          t("settings_btn"),
        ),
      ),
      optionsSection,
      state.lastError
        ? h(
            Alert,
            {
              title: state.lastError,
              severity: "error",
              onRemove: function () {
                setField("lastError", null);
              },
            },
            null,
          )
        : null,
      toolbar,
      listSection,
    );

    if (PluginPage) {
      return h(PluginPage, { pageNav: { text: t("page_title") } }, content);
    }
    return h(
      "div",
      { style: { padding: "24px", maxWidth: "1100px" } },
      h("h2", { style: { marginTop: 0 } }, t("page_title")),
      content,
    );
  }

  // ---------- Preview de la cover (rendu HTML qui mime drawCover) ----------
  //
  // Échelle de référence : A4 paysage 297×210 mm. Toutes les positions
  // exprimées en mm puis converties en % pour s'adapter à la largeur dispo.

  function CoverPreview(props) {
    var isDark = props.theme === "dark";
    var bg = isDark ? "#0F1115" : "#FFFFFF";
    // Card semi-transparente — laisse voir le background. Alpha 0.7 ≈
    // backend gofpdf SetAlpha(0.7).
    var panel = isDark
      ? "rgba(27, 30, 37, 0.7)"
      : "rgba(243, 244, 246, 0.7)";
    var textMain = isDark ? "#F3F4F6" : "#111827";
    var textDim = isDark ? "#9CA3AF" : "#6B7280";
    // A4 paysage : 297mm × 210mm.
    var ASPECT = 297 / 210;

    function pct(v, total) {
      return (v / total) * 100 + "%";
    }

    var hasLogo = !!props.logoDataURL;
    var brandLeftMm = hasLogo ? 54 : 30;

    var bgImage = props.backgroundDataURL
      ? "url(" + props.backgroundDataURL + ")"
      : "none";

    return h(
      "div",
      {
        style: {
          position: "relative",
          width: "100%",
          maxWidth: "880px",
          aspectRatio: String(ASPECT),
          background: bg,
          backgroundImage: bgImage,
          backgroundSize: "cover",
          backgroundPosition: "center",
          border: "1px solid #2a2c33",
          borderRadius: "6px",
          overflow: "hidden",
          fontFamily:
            "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
        },
      },
      // Accent stripe gauche (12 mm sur 297 mm = 4.04%)
      h("div", {
        style: {
          position: "absolute",
          top: 0,
          left: 0,
          bottom: 0,
          width: pct(12, 297),
          background: props.accentHex,
        },
      }),
      // Logo (18 × 18 mm @ 30, 22)
      hasLogo
        ? h("img", {
            src: props.logoDataURL,
            style: {
              position: "absolute",
              top: pct(22, 210),
              left: pct(30, 297),
              width: pct(18, 297),
              height: pct(18, 210),
              objectFit: "contain",
              background: panel,
              borderRadius: "3px",
              padding: "2px",
            },
          })
        : null,
      // Brand title
      h(
        "div",
        {
          style: {
            position: "absolute",
            top: pct(23, 210),
            left: pct(brandLeftMm, 297),
            right: pct(20, 297),
            color: textMain,
            fontSize: "clamp(14px, 2.4vw, 26px)",
            fontWeight: 700,
            lineHeight: 1.1,
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
          },
        },
        props.brandTitle,
      ),
      // Brand subtitle
      h(
        "div",
        {
          style: {
            position: "absolute",
            top: pct(33, 210),
            left: pct(brandLeftMm, 297),
            right: pct(20, 297),
            color: textDim,
            fontSize: "clamp(10px, 1.3vw, 14px)",
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
          },
        },
        props.brandSubtitle,
      ),
      // Card centrée (x=30 mm, y=60 mm, w=297-60=237 mm, h=210-100=110 mm)
      h(
        "div",
        {
          style: {
            position: "absolute",
            top: pct(60, 210),
            left: pct(30, 297),
            width: pct(237, 297),
            height: pct(110, 210),
            background: panel,
            borderRadius: "8px",
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
            gap: "8px",
            padding: "4%",
            boxSizing: "border-box",
          },
        },
        h(
          "div",
          {
            style: {
              color: textMain,
              fontSize: "clamp(18px, 3.6vw, 38px)",
              fontWeight: 700,
              textAlign: "center",
            },
          },
          "Dashboard title",
        ),
        h(
          "div",
          {
            style: {
              color: textDim,
              fontSize: "clamp(11px, 1.6vw, 16px)",
            },
          },
          "Période · now-6h → now",
        ),
        h(
          "div",
          {
            style: {
              color: textMain,
              fontSize: "clamp(10px, 1.3vw, 13px)",
              marginTop: "12px",
              opacity: 0.85,
            },
          },
          "Généré le 2026-06-12 18:30 UTC",
        ),
      ),
      // Footer left (x=30 mm, y=ph-15=195 mm)
      h(
        "div",
        {
          style: {
            position: "absolute",
            top: pct(195, 210),
            left: pct(30, 297),
            color: textDim,
            fontSize: "clamp(8px, 1vw, 11px)",
          },
        },
        props.footerLeft,
      ),
      // Footer right (x=pw-90=207 mm)
      h(
        "div",
        {
          style: {
            position: "absolute",
            top: pct(195, 210),
            left: pct(207, 297),
            color: textDim,
            fontSize: "clamp(8px, 1vw, 11px)",
          },
        },
        props.footerRight,
      ),
    );
  }

  // ---------- Settings view (cover branding) ----------

  // Construit dynamiquement un prompt image-gen avec la couleur d'accent
  // et le thème en input. Inclut un placeholder explicite pour que le user
  // sache où insérer son idée créative.
  function buildBackgroundPrompt(accentHex, theme) {
    var isDark = theme === "dark";
    var palette = isDark ? "dark slate (#0F1115)" : "off-white (#F9FAFB)";
    return [
      "# === ÉDITE CETTE LIGNE — décris ton thème visuel ici ===",
      "[YOUR CONCEPT HERE — e.g. \"abstract topographic terrain\", \"sparse circuit board lines\", \"flowing data streams\", \"minimalist constellation grid\"]",
      "# ====================================================",
      "",
      "Generate the above concept as an abstract background image for an A4-landscape PDF cover page (aspect ratio 1.414:1, ideally 1414×1000 px).",
      "",
      "STRICTLY NO TEXT in the image: no letters, no words, no numbers, no logos, no signs, no labels — pure abstract visuals only. (Text will be overlaid by the PDF generator.)",
      "",
      "Style: minimalist, professional, modern dashboard/observability aesthetic.",
      "",
      "Color palette:",
      "- Dominant tone: " + palette,
      "- Accent color (used sparingly, ~5–10% of pixels): " + accentHex,
      "- No other vivid colors — keep the palette tight.",
      "",
      "Layout (zones the image must respect — visible details should live in the CORNERS and EDGES, not in these reserved areas):",
      "- Left vertical strip 0%–5% width: completely uniform background (will be overlaid with an accent color stripe).",
      "- Top-left zone 7%–35% width × 8%–22% height: keep quiet/subtle (brand title will be overlaid).",
      "- Center 10%–90% width × 28%–82% height: keep quiet/subtle (a semi-transparent card with the dashboard title will be overlaid here — but the background should still show through, so make it interesting but not busy).",
      "- Bottom strip 92%–100% height: keep quiet/subtle (footer text).",
      "",
      "Push visual interest to the four corners and the right side. The central zone should be readable through a 70%-opacity dark overlay.",
    ].join("\n");
  }

  // Copy-to-clipboard avec fallback document.execCommand pour les contextes
  // sans Clipboard API (Safari ancien, contextes insecure).
  function copyToClipboard(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text);
    }
    return new Promise(function (resolve, reject) {
      try {
        var ta = document.createElement("textarea");
        ta.value = text;
        ta.style.position = "fixed";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.select();
        document.execCommand("copy");
        ta.remove();
        resolve();
      } catch (e) {
        reject(e);
      }
    });
  }

  function SettingsView(props) {
    var s = React.useState({
      loading: true,
      error: null,
      saving: false,
      savedAt: 0,
      copiedAt: 0,
      pluginSettings: null,
      form: {
        coverBrandTitle: "",
        coverBrandSubtitle: "",
        coverFooterLeft: "",
        coverFooterRight: "",
        coverAccentHex: "#10B981",
        coverLogoDataURL: "",
        coverBackgroundDataURL: "",
      },
    });
    var state = s[0];
    var setState = s[1];

    React.useEffect(function () {
      fetch("/api/plugins/" + PLUGIN_ID + "/settings", {
        credentials: "include",
      })
        .then(function (r) {
          if (!r.ok) throw new Error("HTTP " + r.status);
          return r.json();
        })
        .then(function (settings) {
          var jd = settings.jsonData || {};
          setState(function (p) {
            return Object.assign({}, p, {
              loading: false,
              pluginSettings: settings,
              form: {
                coverBrandTitle: jd.coverBrandTitle || "",
                coverBrandSubtitle: jd.coverBrandSubtitle || "",
                coverFooterLeft: jd.coverFooterLeft || "",
                coverFooterRight: jd.coverFooterRight || "",
                coverAccentHex: jd.coverAccentHex || "#10B981",
                coverLogoDataURL: jd.coverLogoDataURL || "",
                coverBackgroundDataURL: jd.coverBackgroundDataURL || "",
              },
            });
          });
        })
        .catch(function (e) {
          setState(function (p) {
            return Object.assign({}, p, {
              loading: false,
              error: String(e),
            });
          });
        });
    }, []);

    function setFormField(key, v) {
      setState(function (p) {
        var nf = Object.assign({}, p.form);
        nf[key] = v;
        return Object.assign({}, p, { form: nf });
      });
    }
    function onLogoFile(e) {
      var file = e.target.files && e.target.files[0];
      if (!file) return;
      if (file.size > 200 * 1024) {
        setState(function (p) {
          return Object.assign({}, p, { error: t("logo_too_large") });
        });
        return;
      }
      var fr = new FileReader();
      fr.onload = function () {
        setFormField("coverLogoDataURL", String(fr.result));
      };
      fr.readAsDataURL(file);
    }
    function onBackgroundFile(e) {
      var file = e.target.files && e.target.files[0];
      if (!file) return;
      if (file.size > 2 * 1024 * 1024) {
        setState(function (p) {
          return Object.assign({}, p, { error: t("background_too_large") });
        });
        return;
      }
      var fr = new FileReader();
      fr.onload = function () {
        setFormField("coverBackgroundDataURL", String(fr.result));
      };
      fr.readAsDataURL(file);
    }
    function doCopyPrompt() {
      var promptStr = buildBackgroundPrompt(
        state.form.coverAccentHex || "#10B981",
        currentGrafanaTheme(),
      );
      copyToClipboard(promptStr)
        .then(function () {
          setState(function (p) {
            return Object.assign({}, p, { copiedAt: Date.now() });
          });
        })
        .catch(function (err) {
          setState(function (p) {
            return Object.assign({}, p, {
              error: "Copy failed: " + (err && err.message ? err.message : err),
            });
          });
        });
    }
    function doSave() {
      var current = state.pluginSettings || {};
      var mergedJSON = Object.assign({}, current.jsonData || {}, {
        coverBrandTitle: state.form.coverBrandTitle,
        coverBrandSubtitle: state.form.coverBrandSubtitle,
        coverFooterLeft: state.form.coverFooterLeft,
        coverFooterRight: state.form.coverFooterRight,
        coverAccentHex: state.form.coverAccentHex,
        coverLogoDataURL: state.form.coverLogoDataURL,
        coverBackgroundDataURL: state.form.coverBackgroundDataURL,
      });
      var body = {
        enabled: current.enabled !== false,
        pinned: current.pinned !== false,
        jsonData: mergedJSON,
      };
      setState(function (p) {
        return Object.assign({}, p, { saving: true, error: null });
      });
      fetch("/api/plugins/" + PLUGIN_ID + "/settings", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(body),
      })
        .then(function (r) {
          if (r.status === 403) throw new Error(t("forbidden"));
          if (!r.ok) {
            return r.text().then(function (txt) {
              throw new Error("HTTP " + r.status + ": " + txt);
            });
          }
        })
        .then(function () {
          setState(function (p) {
            return Object.assign({}, p, { saving: false, savedAt: Date.now() });
          });
        })
        .catch(function (e) {
          setState(function (p) {
            return Object.assign({}, p, {
              saving: false,
              error: String(e && e.message ? e.message : e),
            });
          });
        });
    }

    var content = h(
      React.Fragment,
      null,
      h(
        "div",
        {
          style: {
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            gap: "16px",
            marginBottom: "8px",
          },
        },
        h("p", { style: { opacity: 0.8, margin: 0 } }, t("settings_subtitle")),
        h(
          Button,
          {
            variant: "secondary",
            fill: "outline",
            size: "sm",
            icon: "arrow-left",
            onClick: props.onBack,
          },
          t("settings_back"),
        ),
      ),
      state.error
        ? h(
            Alert,
            {
              title: state.error,
              severity: "error",
              onRemove: function () {
                setState(function (p) {
                  return Object.assign({}, p, { error: null });
                });
              },
            },
            null,
          )
        : null,
      state.savedAt > 0 && Date.now() - state.savedAt < 5000
        ? h(Alert, { title: t("saved"), severity: "success" }, null)
        : null,
      state.loading
        ? h(
            "div",
            { style: { display: "flex", gap: "8px", alignItems: "center" } },
            Spinner ? h(Spinner, null) : null,
            h("span", null, t("loading")),
          )
        : h(
            "div",
            {
              style: {
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))",
                gap: "16px",
              },
            },
            h(
              Field,
              { label: t("brand_title") },
              h(Input, {
                value: state.form.coverBrandTitle,
                onChange: function (e) {
                  setFormField("coverBrandTitle", e.target.value);
                },
                placeholder: "Grafana",
              }),
            ),
            h(
              Field,
              { label: t("brand_subtitle") },
              h(Input, {
                value: state.form.coverBrandSubtitle,
                onChange: function (e) {
                  setFormField("coverBrandSubtitle", e.target.value);
                },
                placeholder: "Dashboard report",
              }),
            ),
            h(
              Field,
              { label: t("footer_left") },
              h(Input, {
                value: state.form.coverFooterLeft,
                onChange: function (e) {
                  setFormField("coverFooterLeft", e.target.value);
                },
                placeholder: "Confidentiel — ne pas redistribuer",
              }),
            ),
            h(
              Field,
              { label: t("footer_right") },
              h(Input, {
                value: state.form.coverFooterRight,
                onChange: function (e) {
                  setFormField("coverFooterRight", e.target.value);
                },
                placeholder: "grafana-pdf-reporter",
              }),
            ),
            h(
              Field,
              { label: t("accent_hex") },
              h(
                "div",
                {
                  style: {
                    display: "flex",
                    gap: "8px",
                    alignItems: "center",
                  },
                },
                h("input", {
                  type: "color",
                  value: state.form.coverAccentHex,
                  onChange: function (e) {
                    setFormField("coverAccentHex", e.target.value);
                  },
                  style: {
                    width: "44px",
                    height: "32px",
                    border: "1px solid #2a2c33",
                    background: "transparent",
                    cursor: "pointer",
                    flex: "0 0 auto",
                  },
                }),
                h(Input, {
                  value: state.form.coverAccentHex,
                  onChange: function (e) {
                    setFormField("coverAccentHex", e.target.value);
                  },
                  placeholder: "#10B981",
                  style: { fontFamily: "monospace" },
                }),
              ),
            ),
            h(
              Field,
              { label: t("logo") },
              h(
                "div",
                {
                  style: {
                    display: "flex",
                    gap: "12px",
                    alignItems: "center",
                    flexWrap: "wrap",
                  },
                },
                state.form.coverLogoDataURL
                  ? h("img", {
                      src: state.form.coverLogoDataURL,
                      style: {
                        height: "40px",
                        width: "40px",
                        objectFit: "contain",
                        border: "1px solid #2a2c33",
                        borderRadius: "4px",
                        padding: "4px",
                      },
                    })
                  : null,
                h("input", {
                  type: "file",
                  accept: "image/png,image/jpeg",
                  onChange: onLogoFile,
                  style: { fontSize: "12px" },
                }),
                state.form.coverLogoDataURL
                  ? h(
                      Button,
                      {
                        variant: "destructive",
                        fill: "outline",
                        size: "sm",
                        onClick: function () {
                          setFormField("coverLogoDataURL", "");
                        },
                      },
                      t("logo_remove"),
                    )
                  : null,
              ),
            ),
            h(
              Field,
              { label: t("background") },
              h(
                "div",
                {
                  style: {
                    display: "flex",
                    gap: "12px",
                    alignItems: "center",
                    flexWrap: "wrap",
                  },
                },
                h("input", {
                  type: "file",
                  accept: "image/png,image/jpeg",
                  onChange: onBackgroundFile,
                  style: { fontSize: "12px" },
                }),
                state.form.coverBackgroundDataURL
                  ? h(
                      Button,
                      {
                        variant: "destructive",
                        fill: "outline",
                        size: "sm",
                        onClick: function () {
                          setFormField("coverBackgroundDataURL", "");
                        },
                      },
                      t("background_remove"),
                    )
                  : null,
              ),
            ),
          ),
      // Prompt image-gen (full width sous le grid)
      !state.loading
        ? h(
            "div",
            { style: { marginTop: "16px" } },
            h(
              Field,
              { label: t("prompt_title"), description: t("prompt_help") },
              h(
                "div",
                { style: { display: "flex", flexDirection: "column", gap: "8px" } },
                h("textarea", {
                  readOnly: true,
                  value: buildBackgroundPrompt(
                    state.form.coverAccentHex || "#10B981",
                    currentGrafanaTheme(),
                  ),
                  rows: 10,
                  style: {
                    background: "#0f1115",
                    color: "#e5e7eb",
                    border: "1px solid #2a2c33",
                    borderRadius: "4px",
                    padding: "10px",
                    fontFamily: "monospace",
                    fontSize: "12px",
                    width: "100%",
                    boxSizing: "border-box",
                    resize: "vertical",
                  },
                }),
                h(
                  "div",
                  null,
                  h(
                    Button,
                    {
                      variant: "secondary",
                      fill: "outline",
                      size: "sm",
                      icon: "copy",
                      onClick: doCopyPrompt,
                    },
                    state.copiedAt > 0 && Date.now() - state.copiedAt < 3000
                      ? t("copied")
                      : t("copy_prompt"),
                  ),
                ),
              ),
            ),
          )
        : null,
      // Preview de la cover
      !state.loading
        ? h(
            "div",
            { style: { marginTop: "16px" } },
            h(
              "div",
              {
                style: {
                  color: "#e5e7eb",
                  fontSize: "0.9em",
                  fontWeight: 500,
                  marginBottom: "8px",
                },
              },
              t("preview"),
            ),
            h(CoverPreview, {
              brandTitle: state.form.coverBrandTitle || "Grafana",
              brandSubtitle:
                state.form.coverBrandSubtitle || "Dashboard report",
              footerLeft:
                state.form.coverFooterLeft ||
                "Confidential — do not redistribute",
              footerRight: state.form.coverFooterRight || "grafana-pdf-reporter",
              accentHex: state.form.coverAccentHex || "#10B981",
              logoDataURL: state.form.coverLogoDataURL,
              backgroundDataURL: state.form.coverBackgroundDataURL,
              theme: currentGrafanaTheme(),
            }),
          )
        : null,
      h(
        "div",
        { style: { marginTop: "16px" } },
        h(
          Button,
          {
            variant: "primary",
            icon: "save",
            disabled: state.loading || state.saving,
            onClick: doSave,
          },
          state.saving ? t("saving") : t("save"),
        ),
      ),
    );

    if (PluginPage) {
      return h(
        PluginPage,
        { pageNav: { text: t("settings_title") } },
        content,
      );
    }
    return h(
      "div",
      { style: { padding: "24px", maxWidth: "1100px" } },
      h("h2", { style: { marginTop: 0 } }, t("settings_title")),
      content,
    );
  }

  // ---------- AppPlugin ----------

  // 2 entrées séparées : la sidebar navigue (path → router Grafana, pas de
  // full reload), la palette de commandes déclenche l'export du dashboard
  // courant (ou retombe sur la page d'app si on n'est pas sur un dashboard).
  var plugin = new AppPlugin()
    .setRootPage(RootPage)
    .addLink({
      title: "Export PDF",
      description: "Open the PDF Reporter page",
      targets: ["grafana/megamenu/action"],
      path: "/a/" + PLUGIN_ID,
    })
    .addLink({
      title: "Export PDF",
      description: "Generate a PDF report of the current dashboard",
      targets: ["grafana/commandpalette/action"],
      onClick: function (event, helpers) {
        var ctx = helpers && (helpers.context || helpers);
        var uid = uidFromContext(ctx);
        if (uid) exportSingle(uid);
        else window.location.href = "/a/" + PLUGIN_ID;
      },
    });

  return { plugin: plugin };
});
