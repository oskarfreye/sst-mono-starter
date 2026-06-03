(function () {
  "use strict";

  const tabs = Array.from(document.querySelectorAll("[data-tab]"));
  const panels = Array.from(document.querySelectorAll("[data-tab-panel]"));
  if (!tabs.length || !panels.length) return;

  function activate(id) {
    tabs.forEach((tab) => {
      if (tab.getAttribute("data-tab") === id) {
        tab.setAttribute("data-active", "true");
      } else {
        tab.removeAttribute("data-active");
      }
    });
    panels.forEach((panel) => {
      if (panel.getAttribute("data-tab-panel") === id) {
        panel.setAttribute("data-active", "true");
      } else {
        panel.removeAttribute("data-active");
      }
    });
  }

  tabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      const id = tab.getAttribute("data-tab");
      if (id) activate(id);
    });
  });

  const initial = tabs.find((tab) => tab.getAttribute("data-active") === "true") || tabs[0];
  if (initial) activate(initial.getAttribute("data-tab") || "console");
})();
