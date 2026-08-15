import PhotoSwipeLightbox from "/static/vendor/photoswipe/photoswipe-lightbox.esm.js";
import PhotoSwipe from "/static/vendor/photoswipe/photoswipe.esm.js";

function visibleCards() {
  return Array.from(document.querySelectorAll("#file-panel [data-gallery-src]")).filter(function (el) {
    return el.offsetParent !== null;
  });
}

function measure(src) {
  return new Promise(function (resolve, reject) {
    const img = new Image();
    img.onload = function () {
      resolve({ width: img.naturalWidth, height: img.naturalHeight });
    };
    img.onerror = reject;
    img.src = src;
  });
}

const lightbox = new PhotoSwipeLightbox({
  pswpModule: PhotoSwipe,
  bgOpacity: 0.92,
  padding: { top: 56, bottom: 64, left: 16, right: 16 },
  wheelToZoom: true,
  imageClickAction: "zoom",
  tapAction: "zoom",
});

lightbox.init();

async function openGallery(startCard) {
  const cards = visibleCards();
  const index = Math.max(0, cards.indexOf(startCard));
  const items = [];

  for (const el of cards) {
    const src = el.getAttribute("data-gallery-src");
    if (!src) continue;
    const thumb = el.querySelector("img");
    const item = {
      src: src,
      width: 1600,
      height: 1200,
      alt: el.getAttribute("data-gallery-name") || "",
      msrc: thumb?.currentSrc || thumb?.src || "",
    };
    try {
      const size = await measure(src);
      if (size.width && size.height) {
        item.width = size.width;
        item.height = size.height;
      }
    } catch (_) {}
    items.push(item);
  }

  if (!items.length) return;
  lightbox.loadAndOpen(Math.min(index, items.length - 1), items);
}

document.addEventListener("click", function (event) {
  if (event.target.closest(".card-actions")) return;
  const card = event.target.closest("[data-gallery-src]");
  if (!card) return;
  event.preventDefault();
  event.stopPropagation();
  openGallery(card);
});
