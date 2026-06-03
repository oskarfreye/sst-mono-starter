(function () {
  const buttons = Array.from(document.querySelectorAll("[data-wake-filter]"));
  const entries = Array.from(document.querySelectorAll("[data-cohort]"));
  if (!buttons.length) return;

  function applyFilter(filter) {
    buttons.forEach((button) => {
      const active = button.getAttribute("data-wake-filter") === filter;
      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", active ? "true" : "false");
    });
    entries.forEach((entry) => {
      const hidden = filter !== "ALL" && entry.getAttribute("data-cohort") !== filter;
      entry.toggleAttribute("hidden", hidden);
      entry.classList.toggle("is-hidden", hidden);
    });
  }

  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      applyFilter(button.getAttribute("data-wake-filter") || "ALL");
    });
  });
  applyFilter("ALL");
})();
