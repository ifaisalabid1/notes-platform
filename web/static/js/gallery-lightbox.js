(() => {
  const initGalleryLightbox = () => {
    if (typeof GLightbox === "undefined") {
      return;
    }

    GLightbox({
      selector: ".gallery-lightbox",
      touchNavigation: true,
      loop: true,
      openEffect: "fade",
      closeEffect: "fade",
    });
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initGalleryLightbox);
    return;
  }

  initGalleryLightbox();
})();
