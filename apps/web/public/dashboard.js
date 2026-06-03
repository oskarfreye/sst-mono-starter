(function () {
  const forms = document.querySelectorAll("[data-airlock-form]");
  forms.forEach((form) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const status = form.querySelector("[data-form-status]") || form.closest(".profile-card-body")?.querySelector("[data-form-status]");
      setStatus(status, "TRANSMITTING");
      const button = form.querySelector("button[type='submit']");
      button?.setAttribute("disabled", "true");

      try {
        const payload = Object.fromEntries(new FormData(form).entries());
        const response = await fetch(form.action, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
        const json = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(json.error || "Request failed");
        }
        const mission = json.mission;
        setStatus(status, mission?.status ? `ACCEPTED · ${mission.status}` : "ACCEPTED");
        form.reset();
      } catch (error) {
        setStatus(status, error instanceof Error ? `REJECTED · ${error.message}` : "REJECTED");
      } finally {
        button?.removeAttribute("disabled");
      }
    });
  });

  function setStatus(node, value) {
    if (node) node.textContent = value;
  }
})();
