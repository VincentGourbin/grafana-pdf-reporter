// Mini AMD loader pour servir le module.amd.js du plugin en dev.
// Suffisant pour 1 define() avec des deps qu'on a déjà collées sur window.

(function () {
  var registry = {
    react: window.React,
    "@grafana/data": window.__gdata,
    "@grafana/ui": window.__gui,
    "@grafana/runtime": window.__gruntime,
  };

  window.define = function (deps, factory) {
    var resolved = deps.map(function (d) {
      if (!(d in registry)) {
        console.warn("[amd-shim] unknown dep:", d);
      }
      return registry[d];
    });
    var exports = factory.apply(null, resolved);
    // Module entry attendu : { plugin: AppPlugin } — on récupère le RootPage
    // posé sur window par setRootPage stub.
    window.__pdfreporter_module = exports;
  };
  window.define.amd = {};
})();
