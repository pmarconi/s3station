(function () {
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
})();
