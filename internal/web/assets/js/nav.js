(function () {
  function bindBusy() {
    var host = document.querySelector("[data-signals]");
    var dlg = document.getElementById("page-busy");
    if (!host || !dlg) return;
    var sync = function () {
      var on = host.getAttribute("data-busy") === "true";
      if (on && !dlg.open) dlg.showModal();
      if (!on && dlg.open) dlg.close();
    };
    dlg.addEventListener("cancel", function (event) {
      event.preventDefault();
    });
    new MutationObserver(sync).observe(host, { attributes: true, attributeFilter: ["data-busy"] });
    sync();
  }

  function prefixFromPath(path) {
    path = decodeURIComponent(path || "/").replace(/^\/+|\/+$/g, "");
    if (path !== "files" && path.indexOf("files/") !== 0) return null;
    if (path === "files") return "";
    return path.slice("files/".length) + "/";
  }

  window.addEventListener("popstate", function () {
    var prefix = prefixFromPath(location.pathname);
    if (prefix === null) return;
    window.dispatchEvent(new CustomEvent("station-navigate", { detail: { prefix: prefix } }));
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bindBusy);
  } else {
    bindBusy();
  }
})();
