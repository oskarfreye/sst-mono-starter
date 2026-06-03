(function () {
  "use strict";

  const nodes = Array.from(document.querySelectorAll("[data-deadline]"));
  if (!nodes.length) return;

  function remaining(deadline, now) {
    const target = new Date(deadline || "").getTime();
    if (!target || Number.isNaN(target)) return "";
    const ms = target - now;
    if (ms <= 0) return "HATCH CLOSED";
    const totalSeconds = Math.floor(ms / 1000);
    const days = Math.floor(totalSeconds / 86400);
    const hours = Math.floor((totalSeconds % 86400) / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;
    return (
      String(days) +
      "d " +
      String(hours).padStart(2, "0") +
      "h " +
      String(minutes).padStart(2, "0") +
      "m " +
      String(seconds).padStart(2, "0") +
      "s"
    );
  }

  function tick() {
    const now = Date.now();
    nodes.forEach((node) => {
      const readout = node.querySelector(".cc-readout") || node;
      const value = remaining(node.getAttribute("data-deadline") || "", now);
      if (value) readout.textContent = value;
    });
  }

  tick();
  setInterval(tick, 1000);
})();
