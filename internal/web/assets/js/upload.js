(function () {
  function prefix() {
    return document.querySelector("[data-current-prefix]")?.getAttribute("data-current-prefix") || "";
  }

  function emit(name, detail) {
    window.dispatchEvent(new CustomEvent(name, { detail: detail || {} }));
  }

  async function uploadFiles(fileList) {
    const files = Array.from(fileList || []);
    if (!files.length) return;

    emit("station-upload-start", { message: "Uploading…" });

    const keys = [];
    try {
      for (let i = 0; i < files.length; i++) {
        const file = files[i];
        emit("station-upload-start", {
          message: "Uploading " + (i + 1) + "/" + files.length + " · " + file.name,
        });

        const presignRes = await fetch("/uploads/presign", {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            name: file.name,
            contentType: file.type || "application/octet-stream",
            size: file.size,
            prefix: prefix(),
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
          throw new Error("S3 rejected " + file.name + " (" + putRes.status + ")");
        }
        keys.push(spec.key);
      }

      const done = await fetch("/uploads/complete", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ keys, prefix: prefix() }),
      });
      if (!done.ok) {
        throw new Error("Upload finished but the listing cache could not be updated");
      }

      emit("station-uploaded", {
        message: "Uploaded " + files.length + " file" + (files.length === 1 ? "" : "s"),
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
        uploadFiles(picker.files);
        picker.value = "";
      });
    }

    const moveType = "application/x-station-key";
    let movingKey = "";

    function isInternalMove() {
      return Boolean(movingKey);
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
      event.preventDefault();
      event.stopPropagation();
      emit("station-upload-start", { message: "Moving…" });
      try {
        const res = await fetch("/files/move", {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            key: key,
            destPrefix: folder.getAttribute("data-station-folder") || "",
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
      if (isInternalMove() || !event.dataTransfer?.files?.length) return;
      event.preventDefault();
      uploadFiles(event.dataTransfer.files);
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bind);
  } else {
    bind();
  }
})();
