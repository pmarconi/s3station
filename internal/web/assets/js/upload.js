(function () {
  const SKIP = { ".DS_Store": true, "Thumbs.db": true, "desktop.ini": true };

  function prefix() {
    return document.querySelector("[data-current-prefix]")?.getAttribute("data-current-prefix") || "";
  }

  function isLocked() {
    return document.querySelector("[data-locked='true']") != null;
  }

  function emit(name, detail) {
    window.dispatchEvent(new CustomEvent(name, { detail: detail || {} }));
  }

  function skipName(name) {
    return Boolean(SKIP[name]);
  }

  function readAllEntries(reader) {
    return new Promise(function (resolve, reject) {
      const all = [];
      const next = function () {
        reader.readEntries(function (batch) {
          if (!batch.length) {
            resolve(all);
            return;
          }
          all.push.apply(all, batch);
          next();
        }, reject);
      };
      next();
    });
  }

  function walkEntry(entry, path, out) {
    if (!entry) return Promise.resolve();
    if (entry.isFile) {
      return new Promise(function (resolve, reject) {
        entry.file(function (file) {
          if (!skipName(file.name)) {
            out.push({ file: file, relativePath: path + file.name });
          }
          resolve();
        }, reject);
      });
    }
    if (!entry.isDirectory) return Promise.resolve();
    const dirPath = path + entry.name + "/";
    return readAllEntries(entry.createReader()).then(function (children) {
      if (!children.length) {
        out.push({ file: null, relativePath: dirPath.replace(/\/$/, ""), isDir: true });
        return;
      }
      return children.reduce(function (prev, child) {
        return prev.then(function () {
          return walkEntry(child, dirPath, out);
        });
      }, Promise.resolve());
    });
  }

  function collectDropped(dataTransfer) {
    const out = [];
    const items = dataTransfer && dataTransfer.items;
    if (items && items.length) {
      const walks = [];
      for (let i = 0; i < items.length; i++) {
        const item = items[i];
        const entry = item.webkitGetAsEntry && item.webkitGetAsEntry();
        if (entry) {
          walks.push(walkEntry(entry, "", out));
        } else if (item.kind === "file") {
          const file = item.getAsFile();
          if (file && !skipName(file.name)) {
            out.push({ file: file, relativePath: file.webkitRelativePath || file.name });
          }
        }
      }
      return Promise.all(walks).then(function () {
        return out;
      });
    }
    return Promise.resolve(
      Array.from((dataTransfer && dataTransfer.files) || [])
        .filter(function (file) {
          return !skipName(file.name);
        })
        .map(function (file) {
          return { file: file, relativePath: file.webkitRelativePath || file.name };
        })
    );
  }

  function hasFileItems(dataTransfer) {
    if (!dataTransfer) return false;
    if (dataTransfer.items && dataTransfer.items.length) {
      for (let i = 0; i < dataTransfer.items.length; i++) {
        if (dataTransfer.items[i].kind === "file") return true;
      }
    }
    return Boolean(dataTransfer.files && dataTransfer.files.length);
  }

  async function uploadItems(items) {
    const files = [];
    const folders = [];
    (items || []).forEach(function (item) {
      if (item.isDir && item.relativePath) {
        folders.push(item.relativePath);
        return;
      }
      if (item.file) files.push(item);
    });
    if (!files.length && !folders.length) {
      emit("station-upload-error", { message: "Nothing to upload from that drop." });
      return;
    }
    if (isLocked()) {
      emit("station-upload-error", { message: "Unlock this folder first." });
      return;
    }

    emit("station-upload-start", { message: "Uploading…" });

    const keys = [];
    const dest = prefix();
    try {
      for (let i = 0; i < files.length; i++) {
        const item = files[i];
        const file = item.file;
        const name = item.relativePath || file.name;
        emit("station-upload-start", {
          message: "Uploading " + (i + 1) + "/" + files.length + " · " + name,
        });

        const presignRes = await fetch("/uploads/presign", {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            name: name,
            contentType: file.type || "application/octet-stream",
            size: file.size,
            prefix: dest,
          }),
        });
        if (!presignRes.ok) {
          throw new Error((await presignRes.text()) || "Could not prepare upload");
        }

        const spec = await presignRes.json();
        const putRes = await fetch(spec.url, {
          method: spec.method || "PUT",
          headers: spec.headers || {},
          body: file,
        });
        if (!putRes.ok) {
          throw new Error("S3 rejected " + name + " (" + putRes.status + ")");
        }
        keys.push(spec.key);
      }

      const done = await fetch("/uploads/complete", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ keys: keys, prefix: dest, folders: folders }),
      });
      if (!done.ok) {
        throw new Error("Upload finished but the listing cache could not be updated");
      }

      const n = files.length;
      emit("station-uploaded", {
        message: n
          ? "Uploaded " + n + " file" + (n === 1 ? "" : "s")
          : "Folder created.",
      });
    } catch (err) {
      console.error(err);
      emit("station-upload-error", { message: err.message || "Upload failed" });
    }
  }

  function bind() {
    const picker = document.getElementById("upload-input");
    const button = document.getElementById("upload-btn");
    if (button && picker) {
      button.addEventListener("click", function () {
        picker.click();
      });
      picker.addEventListener("change", function () {
        const items = Array.from(picker.files || []).map(function (file) {
          return { file: file, relativePath: file.webkitRelativePath || file.name };
        });
        uploadItems(items);
        picker.value = "";
      });
    }

    const moveType = "application/x-station-key";
    let movingKey = "";

    function isInternalMove() {
      return Boolean(movingKey);
    }

    function cannotMoveInto(src, dest) {
      if (!src || dest == null) return true;
      return dest === src || dest.indexOf(src) === 0;
    }

    document.addEventListener("dragstart", function (event) {
      const card = event.target.closest("[data-station-key]");
      if (!card) return;
      movingKey = card.getAttribute("data-station-key") || "";
      if (!movingKey) return;
      event.dataTransfer.setData(moveType, movingKey);
      event.dataTransfer.setData("text/plain", movingKey);
      event.dataTransfer.effectAllowed = "move";
      card.classList.add("border-primary");
    });
    document.addEventListener("dragend", function () {
      movingKey = "";
      document.querySelectorAll("[data-station-folder].border-primary, [data-station-key].border-primary").forEach(function (el) {
        el.classList.remove("border-primary");
      });
    });
    document.addEventListener("dragover", function (event) {
      if (!isInternalMove()) return;
      const folder = event.target.closest("[data-station-folder]");
      document.querySelectorAll("[data-station-folder].border-primary").forEach(function (el) {
        el.classList.remove("border-primary");
      });
      if (!folder) return;
      const dest = folder.getAttribute("data-station-folder") || "";
      if (cannotMoveInto(movingKey, dest)) return;
      event.preventDefault();
      event.dataTransfer.dropEffect = "move";
      folder.classList.add("border-primary");
    });
    document.addEventListener("drop", async function (event) {
      if (!isInternalMove()) return;
      const folder = event.target.closest("[data-station-folder]");
      const key = movingKey || event.dataTransfer.getData(moveType) || event.dataTransfer.getData("text/plain");
      document.querySelectorAll("[data-station-folder].border-primary").forEach(function (el) {
        el.classList.remove("border-primary");
      });
      if (!folder || !key) return;
      const dest = folder.getAttribute("data-station-folder") || "";
      event.preventDefault();
      event.stopPropagation();
      if (cannotMoveInto(key, dest)) {
        emit("station-upload-error", { message: "Can't move a folder into itself." });
        return;
      }
      emit("station-upload-start", { message: "Moving…" });
      try {
        const res = await fetch("/files/move", {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            key: key,
            destPrefix: dest,
          }),
        });
        if (!res.ok) {
          throw new Error((await res.text()) || "Could not move item");
        }
        emit("station-moved", { message: "Moved into the folder." });
      } catch (err) {
        console.error(err);
        emit("station-upload-error", { message: err.message || "Move failed" });
      }
    });

    const dropOverlay = document.getElementById("drop-overlay");
    window.addEventListener("dragover", function (event) {
      if (isInternalMove() || !event.dataTransfer?.types?.includes("Files")) return;
      event.preventDefault();
      dropOverlay?.removeAttribute("hidden");
    });
    window.addEventListener("dragleave", function (event) {
      if (event.relatedTarget) return;
      dropOverlay?.setAttribute("hidden", "");
    });
    window.addEventListener("drop", function (event) {
      dropOverlay?.setAttribute("hidden", "");
      if (isInternalMove() || !hasFileItems(event.dataTransfer)) return;
      event.preventDefault();
      collectDropped(event.dataTransfer)
        .then(uploadItems)
        .catch(function (err) {
          console.error(err);
          emit("station-upload-error", { message: err.message || "Could not read the dropped folder." });
        });
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bind);
  } else {
    bind();
  }
})();
