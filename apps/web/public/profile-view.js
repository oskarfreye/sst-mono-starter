(function () {
  function fire() {
    const main = document.querySelector(".page-shell");
    const handle = main && main.dataset ? main.dataset.handle : "";
    if (!handle) return;
    const url = "/api/c/" + encodeURIComponent(handle) + "/view";
    if (navigator.sendBeacon) {
      navigator.sendBeacon(url);
      return;
    }
    fetch(url, { method: "POST", keepalive: true }).catch(() => {});
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", fire);
  } else {
    fire();
  }
})();
