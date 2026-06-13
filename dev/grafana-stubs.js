// Stubs minimaux pour @grafana/data, @grafana/ui, @grafana/runtime.
// L'objectif n'est PAS la fidélité visuelle avec Grafana mais de pouvoir
// tester l'UX (layout, états, i18n, échanges réseau mockés) sans Grafana.
//
// Quand le composant n'est pas critique au look, on utilise un wrapper
// HTML sobre dans le ton Grafana sombre. Pour TimeRangePicker on fait
// quelque chose de fonctionnel mais simplifié.

(function () {
  var React = window.React;
  var h = React.createElement;

  // ---------- @grafana/data ----------

  function AppPlugin() {
    this.rootPage = null;
    this.links = [];
  }
  AppPlugin.prototype.setRootPage = function (rp) {
    this.rootPage = rp;
    // Expose pour mount direct par index.html
    window.__pdfreporter_RootPage = rp;
    return this;
  };
  AppPlugin.prototype.addLink = function (cfg) {
    this.links.push(cfg);
    return this;
  };

  function dateTime(input) {
    var d = input == null ? new Date() : new Date(input);
    return {
      valueOf: function () {
        return d.getTime();
      },
      toISOString: function () {
        return d.toISOString();
      },
      format: function () {
        return d.toLocaleString();
      },
      _date: d,
    };
  }

  var rangeUtil = {
    convertRawToRange: function (raw) {
      // Très grossier : reconnaît "now", "now-Nh/d/m" ; tout le reste → now
      function parse(s) {
        if (typeof s !== "string") return new Date();
        if (s === "now") return new Date();
        var m = /^now-(\d+)([smhdwM])$/.exec(s);
        if (m) {
          var n = parseInt(m[1], 10);
          var unit = m[2];
          var ms =
            unit === "s"
              ? 1000
              : unit === "m"
                ? 60000
                : unit === "h"
                  ? 3600000
                  : unit === "d"
                    ? 86400000
                    : unit === "w"
                      ? 7 * 86400000
                      : 30 * 86400000;
          return new Date(Date.now() - n * ms);
        }
        return new Date();
      }
      var from = parse(raw.from);
      var to = parse(raw.to);
      return {
        from: dateTime(from),
        to: dateTime(to),
        raw: raw,
      };
    },
  };

  window.__gdata = {
    AppPlugin: AppPlugin,
    dateTime: dateTime,
    rangeUtil: rangeUtil,
  };

  // ---------- @grafana/ui ----------

  function PluginPage(props) {
    var navText =
      (props.pageNav && props.pageNav.text) || (props.navModel && props.navModel.text) || "Page";
    return h(
      "div",
      { style: { padding: "24px", maxWidth: "1100px" } },
      h("h2", { style: { marginTop: 0 } }, navText),
      props.children,
    );
  }

  function Button(props) {
    var rest = Object.assign({}, props);
    delete rest.variant;
    delete rest.fill;
    delete rest.size;
    delete rest.icon;
    delete rest.children;
    var bg =
      props.variant === "primary"
        ? "#10b981"
        : props.fill === "outline"
          ? "transparent"
          : "#2a2c33";
    var color = props.variant === "primary" ? "#0f1115" : "#e5e7eb";
    var borderColor = props.variant === "primary" ? "#10b981" : "#2a2c33";
    return h(
      "button",
      Object.assign({}, rest, {
        style: Object.assign(
          {
            padding: "6px 12px",
            borderRadius: "4px",
            background: bg,
            color: color,
            border: "1px solid " + borderColor,
            cursor: props.disabled ? "not-allowed" : "pointer",
            opacity: props.disabled ? 0.5 : 1,
            fontSize: "13px",
            display: "inline-flex",
            alignItems: "center",
            gap: "6px",
          },
          props.style || {},
        ),
      }),
      props.icon ? h("span", null, "▸") : null,
      props.children,
    );
  }

  function Field(props) {
    return h(
      "div",
      { style: { marginBottom: "12px" } },
      h(
        "div",
        {
          style: {
            color: "#e5e7eb",
            fontSize: "0.9em",
            fontWeight: 500,
            marginBottom: "4px",
          },
        },
        props.label,
      ),
      props.description
        ? h(
            "div",
            { style: { fontSize: "0.85em", color: "#9ca3af", marginBottom: "6px" } },
            props.description,
          )
        : null,
      props.children,
    );
  }

  function Input(props) {
    return h(
      "input",
      Object.assign({}, props, {
        style: Object.assign(
          {
            background: "#0f1115",
            color: "#e5e7eb",
            border: "1px solid #2a2c33",
            borderRadius: "4px",
            padding: "6px 10px",
            fontSize: "13px",
            width: "100%",
            boxSizing: "border-box",
          },
          props.style || {},
        ),
      }),
    );
  }

  function Checkbox(props) {
    return h("input", {
      type: "checkbox",
      checked: !!props.value,
      onChange: props.onChange,
      style: { transform: "scale(1.1)" },
    });
  }

  function RadioButtonGroup(props) {
    return h(
      "div",
      { style: { display: "flex", gap: "0" } },
      (props.options || []).map(function (opt, i) {
        var active = opt.value === props.value;
        var isFirst = i === 0;
        var isLast = i === props.options.length - 1;
        return h(
          "button",
          {
            key: opt.value,
            onClick: function () {
              if (props.onChange) props.onChange(opt.value);
            },
            style: {
              padding: "6px 12px",
              background: active ? "#10b981" : "#1b1e25",
              color: active ? "#0f1115" : "#e5e7eb",
              border: "1px solid " + (active ? "#10b981" : "#2a2c33"),
              borderRadius:
                isFirst && isLast
                  ? "4px"
                  : isFirst
                    ? "4px 0 0 4px"
                    : isLast
                      ? "0 4px 4px 0"
                      : "0",
              marginLeft: isFirst ? 0 : "-1px",
              cursor: "pointer",
              fontSize: "13px",
              flex: props.fullWidth ? 1 : "none",
            },
          },
          opt.label,
        );
      }),
    );
  }

  function Alert(props) {
    var bg =
      props.severity === "error"
        ? "#7f1d1d"
        : props.severity === "warning"
          ? "#78350f"
          : "#1e3a8a";
    var color =
      props.severity === "error"
        ? "#fecaca"
        : props.severity === "warning"
          ? "#fde68a"
          : "#bfdbfe";
    return h(
      "div",
      {
        style: {
          background: bg,
          color: color,
          padding: "8px 12px",
          borderRadius: "4px",
          marginBottom: "12px",
          display: "flex",
          alignItems: "center",
          gap: "8px",
        },
      },
      h("span", { style: { flex: 1 } }, props.title),
      props.onRemove
        ? h(
            "button",
            {
              onClick: props.onRemove,
              style: {
                background: "transparent",
                border: "1px solid currentColor",
                color: "inherit",
                cursor: "pointer",
                padding: "2px 8px",
                borderRadius: "3px",
              },
            },
            "✕",
          )
        : null,
    );
  }

  function Spinner() {
    return h(
      "span",
      {
        style: {
          display: "inline-block",
          width: "14px",
          height: "14px",
          border: "2px solid #10b981",
          borderTopColor: "transparent",
          borderRadius: "50%",
          animation: "spin 0.6s linear infinite",
        },
      },
      null,
    );
  }
  // injecte keyframes
  var st = document.createElement("style");
  st.innerHTML = "@keyframes spin{to{transform:rotate(360deg)}}";
  document.head.appendChild(st);

  function Icon(props) {
    return h("span", { style: { opacity: 0.8 } }, "•");
  }

  // TimeRangePicker (toolbar style) — version simplifiée mais fonctionnelle.
  function TimeRangePicker(props) {
    var open = React.useState(false);
    var isOpen = open[0];
    var setOpen = open[1];
    var raw = (props.value && props.value.raw) || {
      from: "now-6h",
      to: "now",
    };
    function fmtBound(v) {
      if (v == null) return "?";
      if (typeof v === "string") return v;
      if (typeof v.format === "function") return v.format();
      if (typeof v.toISOString === "function") {
        return v.toISOString().slice(0, 16).replace("T", " ");
      }
      if (v instanceof Date) {
        return v.toISOString().slice(0, 16).replace("T", " ");
      }
      return String(v);
    }
    var label = fmtBound(raw.from) + " → " + fmtBound(raw.to);

    function applyPreset(from, to) {
      var tr = window.__gdata.rangeUtil.convertRawToRange({
        from: from,
        to: to,
      });
      if (props.onChange) props.onChange(tr);
      setOpen(false);
    }

    return h(
      "div",
      {
        style: {
          display: "inline-flex",
          alignItems: "center",
          gap: "0",
          position: "relative",
        },
      },
      h(
        "button",
        {
          onClick: function () {
            if (props.onMoveBackward) props.onMoveBackward();
          },
          style: chromeBtn("left"),
        },
        "«",
      ),
      h(
        "button",
        {
          onClick: function () {
            setOpen(!isOpen);
          },
          style: chromeBtn("middle"),
        },
        "🕒 " + label,
        " ",
        h(
          "span",
          { style: { color: "#f97316", marginLeft: "6px" } },
          props.timeZone === "browser" ? "browser" : props.timeZone,
        ),
      ),
      h(
        "button",
        {
          onClick: function () {
            if (props.onMoveForward) props.onMoveForward();
          },
          style: chromeBtn("middle"),
        },
        "»",
      ),
      h(
        "button",
        {
          onClick: function () {
            if (props.onZoom) props.onZoom();
          },
          style: chromeBtn("right"),
          title: "Zoom out",
        },
        "⊖",
      ),
      isOpen
        ? h(
            "div",
            {
              style: {
                position: "absolute",
                top: "100%",
                left: 0,
                marginTop: "4px",
                background: "#1b1e25",
                border: "1px solid #2a2c33",
                borderRadius: "4px",
                padding: "12px",
                minWidth: "240px",
                zIndex: 10,
              },
            },
            h(
              "div",
              { style: { fontWeight: 500, marginBottom: "8px" } },
              "Plages relatives",
            ),
            [
              ["Last 5 min", "now-5m", "now"],
              ["Last 15 min", "now-15m", "now"],
              ["Last 1 hour", "now-1h", "now"],
              ["Last 6 hours", "now-6h", "now"],
              ["Last 24 hours", "now-24h", "now"],
              ["Last 7 days", "now-7d", "now"],
              ["Last 30 days", "now-30d", "now"],
            ].map(function (row) {
              return h(
                "div",
                {
                  key: row[0],
                  onClick: function () {
                    applyPreset(row[1], row[2]);
                  },
                  style: {
                    padding: "6px 8px",
                    cursor: "pointer",
                    borderRadius: "3px",
                  },
                  onMouseEnter: function (e) {
                    e.currentTarget.style.background = "#2a2c33";
                  },
                  onMouseLeave: function (e) {
                    e.currentTarget.style.background = "transparent";
                  },
                },
                row[0],
              );
            }),
          )
        : null,
    );
  }
  function chromeBtn(pos) {
    return {
      background: "#1b1e25",
      color: "#e5e7eb",
      border: "1px solid #2a2c33",
      borderRadius:
        pos === "left"
          ? "4px 0 0 4px"
          : pos === "right"
            ? "0 4px 4px 0"
            : "0",
      marginLeft: pos === "left" ? 0 : "-1px",
      padding: "6px 10px",
      cursor: "pointer",
      fontSize: "13px",
    };
  }

  window.__gui = {
    PluginPage: PluginPage,
    Button: Button,
    LinkButton: Button,
    Field: Field,
    Input: Input,
    Checkbox: Checkbox,
    RadioButtonGroup: RadioButtonGroup,
    Alert: Alert,
    Spinner: Spinner,
    TimeRangePicker: TimeRangePicker,
    Icon: Icon,
    Stack: function (p) {
      return h(
        "div",
        { style: { display: "flex", gap: "8px" } },
        p.children,
      );
    },
  };

  // ---------- @grafana/runtime ----------

  window.__gruntime = {
    PluginPage: PluginPage,
    // En dev on est toujours en thème sombre (correspond au fond #0f1115
    // du harness). En prod c'est gRT.config injecté par Grafana qui répond.
    config: {
      theme2: { isDark: true },
      theme: { type: "dark" },
    },
  };
})();
