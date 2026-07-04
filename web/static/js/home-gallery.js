(() => {
  const initHomeGallery = () => {
    const slider = document.querySelector("[data-gallery-swiper]");

    if (!slider || typeof Swiper === "undefined") {
      return;
    }

    new Swiper(slider, {
      slidesPerView: 1,
      spaceBetween: 16,
      grabCursor: true,
      keyboard: {
        enabled: true,
      },
      pagination: {
        el: ".gallery-swiper-pagination",
        clickable: true,
      },
      navigation: {
        nextEl: ".gallery-swiper-next",
        prevEl: ".gallery-swiper-prev",
      },
      breakpoints: {
        640: {
          slidesPerView: 2,
        },
        1024: {
          slidesPerView: 3,
        },
      },
    });

  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initHomeGallery);
    return;
  }

  initHomeGallery();
})();
