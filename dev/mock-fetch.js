// Intercepte fetch() pour les endpoints Grafana qu'on utilise.
(function () {
  var origFetch = window.fetch.bind(window);

  var DASHBOARDS = [
    {
      uid: "system-health",
      title: "System health",
      folderTitle: "Reachy",
      type: "dash-db",
    },
    {
      uid: "authentik-sso",
      title: "Authentik SSO",
      folderTitle: "Reachy",
      type: "dash-db",
    },
    {
      uid: "hardening-backups",
      title: "Hardening & Backups",
      folderTitle: "Reachy",
      type: "dash-db",
    },
    {
      uid: "cve-scan",
      title: "CVE scan",
      folderTitle: "Reachy",
      type: "dash-db",
    },
    {
      uid: "llm-providers",
      title: "LLM providers",
      folderTitle: "Reachy",
      type: "dash-db",
    },
    {
      uid: "yamnet-audio",
      title: "YAMNet audio",
      folderTitle: "Reachy",
      type: "dash-db",
    },
    {
      uid: "bird-monitoring",
      title: "Bird monitoring",
      folderTitle: "Jardin",
      type: "dash-db",
    },
    {
      uid: "action-plans-monitoring",
      title: "Action plans monitoring",
      folderTitle: "ops",
      type: "dash-db",
    },
  ];

  // Stockage en mémoire des plugin settings (persiste tant que la page est ouverte)
  var MOCK_PLUGIN_SETTINGS = {
    enabled: true,
    pinned: true,
    jsonData: {
      coverBrandTitle: "Reachy Jardin",
      coverBrandSubtitle: "Rapport d'observabilité Jetson Orin Nano",
      coverFooterLeft: "Confidentiel — ne pas redistribuer",
      coverFooterRight: "grafana-pdf-reporter",
      coverAccentHex: "#10B981",
      coverLogoDataURL: "",
    },
  };

  window.fetch = function (url, opts) {
    if (typeof url === "string") {
      if (url.indexOf("/api/plugins/") === 0 && url.indexOf("/settings") > 0) {
        if (opts && opts.method === "POST") {
          var body = JSON.parse(opts.body);
          MOCK_PLUGIN_SETTINGS = Object.assign({}, MOCK_PLUGIN_SETTINGS, body);
          console.log("[mock-fetch] PUT /settings →", body);
          return Promise.resolve(new Response("", { status: 200 }));
        }
        return Promise.resolve(
          new Response(JSON.stringify(MOCK_PLUGIN_SETTINGS), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      if (url.indexOf("/api/search") === 0) {
        console.log("[mock-fetch] /api/search →", DASHBOARDS.length, "items");
        return Promise.resolve(
          new Response(JSON.stringify(DASHBOARDS), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      if (url.indexOf("/api/plugins/") === 0 && url.indexOf("/bundle") > 0) {
        console.log("[mock-fetch] /bundle", opts && JSON.parse(opts.body));
        // Renvoie un faux PDF (header %PDF + payload bidon)
        var blob = new Blob(["%PDF-1.4\nmock bundle\n"], {
          type: "application/pdf",
        });
        return new Promise(function (resolve) {
          setTimeout(function () {
            resolve(new Response(blob, { status: 200 }));
          }, 800);
        });
      }
    }
    return origFetch(url, opts);
  };
})();
