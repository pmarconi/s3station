(function () {
  const NEIGHBORS = 1;
  const MIN_ZOOM = 1;
  const MAX_ZOOM = 8;

  let items = [];
  let index = 0;
  let scrollLock = false;
  let scale = 1;
  let tx = 0;
  let ty = 0;
  let dragging = false;
  let dragX = 0;
  let dragY = 0;
  let dragTx = 0;
  let dragTy = 0;

  function $(id) {
    return document.getElementById(id);
  }

  function visibleCards() {
    return Array.from(document.querySelectorAll("#file-panel [data-gallery-src], #trash-panel [data-gallery-src]")).filter(function (el) {
      return el.offsetParent !== null;
    });
  }

  function collect(startCard) {
    const cards = visibleCards();
    const list = cards.map(function (el) {
      const thumb = el.querySelector("img");
      return {
        src: el.getAttribute("data-gallery-src") || "",
        name: el.getAttribute("data-gallery-name") || "",
        kind: el.getAttribute("data-gallery-kind") || "image",
        thumb: thumb?.currentSrc || thumb?.src || "",
      };
    }).filter(function (item) {
      return Boolean(item.src);
    });
    const startSrc = startCard.getAttribute("data-gallery-src");
    const start = Math.max(0, list.findIndex(function (item) {
      return item.src === startSrc;
    }));
    return { list: list, start: start };
  }

  function currentItem() {
    return items[index];
  }

  function isVideo(item) {
    return item && item.kind === "video";
  }

  function slideEl(i) {
    return document.querySelector('#gallery-carousel [data-slide="' + i + '"]');
  }

  function slideMedia(i) {
    const slide = slideEl(i);
    if (!slide) return null;
    return slide.querySelector("img, video");
  }

  function currentImage() {
    const item = currentItem();
    if (isVideo(item)) return null;
    return slideMedia(index);
  }

  function hydrate(i) {
    const media = slideMedia(i);
    const item = items[i];
    if (!media || !item) return;
    if (media.dataset.loaded === item.src) return;
    if (item.kind === "video") {
      if (item.thumb) media.poster = item.thumb;
      media.src = item.src;
    } else {
      media.src = item.src;
    }
    media.dataset.loaded = item.src;
  }

  function unload(i) {
    const media = slideMedia(i);
    const item = items[i];
    if (!media || !item) return;
    if (media.tagName === "VIDEO") {
      media.pause();
      media.removeAttribute("src");
      media.removeAttribute("poster");
      media.load();
    } else if (media.dataset.loaded === item.src) {
      media.removeAttribute("src");
      if (item.thumb) media.src = item.thumb;
    }
    delete media.dataset.loaded;
  }

  function pauseOthers(center) {
    items.forEach(function (item, i) {
      if (i === center || item.kind !== "video") return;
      const media = slideMedia(i);
      if (media && media.tagName === "VIDEO") media.pause();
    });
  }

  function playCurrent() {
    const item = currentItem();
    const media = slideMedia(index);
    if (!isVideo(item) || !media) return;
    const play = media.play();
    if (play && play.catch) play.catch(function () {});
  }

  function hydrateWindow(center) {
    for (let i = 0; i < items.length; i++) {
      if (Math.abs(i - center) <= NEIGHBORS) hydrate(i);
      else unload(i);
    }
    pauseOthers(center);
  }

  function applyZoom() {
    const img = currentImage();
    const carousel = $("gallery-carousel");
    if (carousel) carousel.classList.toggle("is-zoomed", scale > 1);
    if (!img) return;
    img.style.transform = "translate(" + tx + "px, " + ty + "px) scale(" + scale + ")";
    img.classList.toggle("is-zoomed", scale > 1);
    img.classList.toggle("is-panning", dragging);
  }

  function resetZoom() {
    scale = 1;
    tx = 0;
    ty = 0;
    dragging = false;
    applyZoom();
    updateChrome();
  }

  function setZoom(next) {
    if (isVideo(currentItem())) return;
    scale = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, next));
    if (scale === 1) {
      tx = 0;
      ty = 0;
    }
    applyZoom();
    updateChrome();
  }

  function updateChrome() {
    const item = currentItem();
    if ($("gallery-title")) $("gallery-title").textContent = item ? item.name : "";
    if ($("gallery-count")) $("gallery-count").textContent = items.length ? index + 1 + " / " + items.length : "";
    if ($("gallery-prev")) $("gallery-prev").hidden = items.length < 2;
    if ($("gallery-next")) $("gallery-next").hidden = items.length < 2;
    const zoom = $("gallery-zoom");
    if (zoom) zoom.hidden = isVideo(item);
    if ($("gallery-zoom-reset")) $("gallery-zoom-reset").textContent = Math.round(scale * 100) + "%";
    if ($("gallery-zoom-out")) $("gallery-zoom-out").disabled = scale <= MIN_ZOOM;
    if ($("gallery-zoom-in")) $("gallery-zoom-in").disabled = scale >= MAX_ZOOM;
  }

  function goTo(next, smooth) {
    if (!items.length) return;
    index = (next + items.length) % items.length;
    resetZoom();
    hydrateWindow(index);
    updateChrome();
    const carousel = $("gallery-carousel");
    if (!carousel) return;
    scrollLock = true;
    carousel.scrollTo({
      left: index * carousel.clientWidth,
      behavior: smooth ? "smooth" : "instant",
    });
    window.setTimeout(function () {
      scrollLock = false;
      playCurrent();
    }, smooth ? 280 : 0);
  }

  function render() {
    const carousel = $("gallery-carousel");
    if (!carousel) return;
    carousel.replaceChildren();
    items.forEach(function (item, i) {
      const slide = document.createElement("div");
      slide.className = "carousel-item";
      slide.dataset.slide = String(i);
      const stage = document.createElement("div");
      stage.className = "gallery-stage";
      let media;
      if (item.kind === "video") {
        media = document.createElement("video");
        media.controls = true;
        media.playsInline = true;
        media.preload = "metadata";
        if (item.thumb) media.poster = item.thumb;
      } else {
        media = document.createElement("img");
        media.draggable = false;
        media.decoding = "async";
        if (item.thumb) media.src = item.thumb;
      }
      media.alt = item.name;
      media.className = "gallery-media";
      stage.appendChild(media);
      slide.appendChild(stage);
      carousel.appendChild(slide);
    });
  }

  function open(startCard) {
    const collected = collect(startCard);
    if (!collected.list.length) return;
    items = collected.list;
    index = collected.start;
    render();
    const modal = $("gallery-modal");
    if (modal && typeof modal.showModal === "function") modal.showModal();
    requestAnimationFrame(function () {
      goTo(index, false);
    });
  }

  function close() {
    const modal = $("gallery-modal");
    if (modal?.open) modal.close();
    items.forEach(function (_, i) {
      unload(i);
    });
    items = [];
    index = 0;
    resetZoom();
    const carousel = $("gallery-carousel");
    if (carousel) carousel.replaceChildren();
  }

  function onCarouselScroll() {
    if (scrollLock || scale > 1) return;
    const carousel = $("gallery-carousel");
    if (!carousel || !items.length) return;
    const next = Math.round(carousel.scrollLeft / Math.max(carousel.clientWidth, 1));
    if (next === index || next < 0 || next >= items.length) return;
    index = next;
    resetZoom();
    hydrateWindow(index);
    updateChrome();
    playCurrent();
  }

  function onWheel(event) {
    const modal = $("gallery-modal");
    if (!modal?.open || isVideo(currentItem())) return;
    if (!event.target.closest(".gallery-stage")) return;
    event.preventDefault();
    const factor = event.deltaY < 0 ? 1.15 : 1 / 1.15;
    setZoom(scale * factor);
  }

  function onPointerDown(event) {
    if (scale <= 1 || isVideo(currentItem())) return;
    if (!event.target.closest(".gallery-stage img")) return;
    dragging = true;
    dragX = event.clientX;
    dragY = event.clientY;
    dragTx = tx;
    dragTy = ty;
    applyZoom();
    event.preventDefault();
  }

  function onPointerMove(event) {
    if (!dragging) return;
    tx = dragTx + (event.clientX - dragX);
    ty = dragTy + (event.clientY - dragY);
    applyZoom();
  }

  function onPointerUp() {
    if (!dragging) return;
    dragging = false;
    applyZoom();
  }

  document.addEventListener("click", function (event) {
    if (event.target.closest("#gallery-close, #gallery-backdrop")) {
      event.preventDefault();
      close();
      return;
    }
    if (event.target.closest("#gallery-zoom-in")) {
      event.preventDefault();
      setZoom(scale * 1.25);
      return;
    }
    if (event.target.closest("#gallery-zoom-out")) {
      event.preventDefault();
      setZoom(scale / 1.25);
      return;
    }
    if (event.target.closest("#gallery-zoom-reset")) {
      event.preventDefault();
      resetZoom();
      return;
    }
    if (event.target.closest("#gallery-prev")) {
      event.preventDefault();
      goTo(index - 1, true);
      return;
    }
    if (event.target.closest("#gallery-next")) {
      event.preventDefault();
      goTo(index + 1, true);
      return;
    }
    if (event.target.closest(".file-tile-actions, .card-actions, .dropdown")) return;
    const card = event.target.closest("#file-panel [data-gallery-src], #trash-panel [data-gallery-src]");
    if (!card) return;
    event.preventDefault();
    event.stopPropagation();
    open(card);
  });

  document.addEventListener("dblclick", function (event) {
    const modal = $("gallery-modal");
    if (!modal?.open || isVideo(currentItem())) return;
    if (!event.target.closest(".gallery-stage img")) return;
    event.preventDefault();
    setZoom(scale > 1 ? 1 : 2);
  });

  document.addEventListener("keydown", function (event) {
    const modal = $("gallery-modal");
    if (!modal?.open) return;
    if (event.key === "Escape") {
      event.preventDefault();
      close();
    } else if (event.key === "ArrowLeft") {
      event.preventDefault();
      goTo(index - 1, true);
    } else if (event.key === "ArrowRight") {
      event.preventDefault();
      goTo(index + 1, true);
    } else if (event.key === "+" || event.key === "=") {
      event.preventDefault();
      setZoom(scale * 1.25);
    } else if (event.key === "-" || event.key === "_") {
      event.preventDefault();
      setZoom(scale / 1.25);
    } else if (event.key === "0") {
      event.preventDefault();
      resetZoom();
    }
  });

  function bind() {
    $("gallery-carousel")?.addEventListener("scroll", onCarouselScroll, { passive: true });
    $("gallery-modal")?.addEventListener("wheel", onWheel, { passive: false });
    $("gallery-modal")?.addEventListener("pointerdown", onPointerDown);
    window.addEventListener("pointermove", onPointerMove);
    window.addEventListener("pointerup", onPointerUp);
    $("gallery-modal")?.addEventListener("close", function () {
      items.forEach(function (_, i) {
        unload(i);
      });
      items = [];
      index = 0;
      resetZoom();
      $("gallery-carousel")?.replaceChildren();
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bind);
  } else {
    bind();
  }
})();
