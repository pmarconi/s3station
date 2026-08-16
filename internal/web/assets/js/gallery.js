(function () {
  const NEIGHBORS = 1;

  let items = [];
  let index = 0;
  let scrollLock = false;

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

  function slideImage(i) {
    return document.querySelector('#gallery-carousel [data-slide="' + i + '"] img');
  }

  function hydrate(i) {
    const img = slideImage(i);
    const item = items[i];
    if (!img || !item) return;
    if (img.dataset.loaded === item.src) return;
    img.src = item.src;
    img.dataset.loaded = item.src;
  }

  function unload(i) {
    const img = slideImage(i);
    const item = items[i];
    if (!img || !item) return;
    if (img.dataset.loaded !== item.src) return;
    img.removeAttribute("src");
    if (item.thumb) img.src = item.thumb;
    delete img.dataset.loaded;
  }

  function hydrateWindow(center) {
    for (let i = 0; i < items.length; i++) {
      if (Math.abs(i - center) <= NEIGHBORS) hydrate(i);
      else unload(i);
    }
  }

  function updateChrome() {
    const item = items[index];
    if ($("gallery-title")) $("gallery-title").textContent = item ? item.name : "";
    if ($("gallery-count")) $("gallery-count").textContent = items.length ? index + 1 + " / " + items.length : "";
    if ($("gallery-prev")) $("gallery-prev").hidden = items.length < 2;
    if ($("gallery-next")) $("gallery-next").hidden = items.length < 2;
  }

  function goTo(next, smooth) {
    if (!items.length) return;
    index = (next + items.length) % items.length;
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
    }, smooth ? 280 : 0);
  }

  function render() {
    const carousel = $("gallery-carousel");
    if (!carousel) return;
    carousel.replaceChildren();
    items.forEach(function (item, i) {
      const slide = document.createElement("div");
      slide.className = "carousel-item w-full items-center justify-center";
      slide.dataset.slide = String(i);
      const img = document.createElement("img");
      img.alt = item.name;
      img.draggable = false;
      img.loading = "lazy";
      img.decoding = "async";
      img.className = "mx-auto max-h-[78vh] w-auto max-w-full object-contain";
      if (item.thumb) img.src = item.thumb;
      slide.appendChild(img);
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
    const carousel = $("gallery-carousel");
    if (carousel) carousel.replaceChildren();
  }

  function onCarouselScroll() {
    if (scrollLock) return;
    const carousel = $("gallery-carousel");
    if (!carousel || !items.length) return;
    const next = Math.round(carousel.scrollLeft / Math.max(carousel.clientWidth, 1));
    if (next === index || next < 0 || next >= items.length) return;
    index = next;
    hydrateWindow(index);
    updateChrome();
  }

  document.addEventListener("click", function (event) {
    if (event.target.closest("#gallery-close, #gallery-backdrop")) {
      event.preventDefault();
      close();
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
    if (event.target.closest(".card-actions, .dropdown")) return;
    const card = event.target.closest("#file-panel [data-gallery-src], #trash-panel [data-gallery-src]");
    if (!card) return;
    event.preventDefault();
    event.stopPropagation();
    open(card);
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
    }
  });

  function bind() {
    $("gallery-carousel")?.addEventListener("scroll", onCarouselScroll, { passive: true });
    $("gallery-modal")?.addEventListener("close", function () {
      items.forEach(function (_, i) {
        unload(i);
      });
      items = [];
      index = 0;
      $("gallery-carousel")?.replaceChildren();
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bind);
  } else {
    bind();
  }
})();
