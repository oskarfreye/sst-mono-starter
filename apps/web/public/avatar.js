(function () {
  "use strict";

  const DEFAULT_SRC = "/astronaut-default.svg";

  const avatarForm = document.getElementById("avatar-form");
  const fileInput = document.getElementById("avatar_file");
  const generateBtn = document.getElementById("avatar-generate");
  const dropBtn = document.getElementById("avatar-drop");
  const img = document.getElementById("avatar-img");
  const status = document.querySelector("[data-avatar-status]");

  function setStatus(value) {
    if (status) status.textContent = value;
  }

  function setBusy(busy) {
    [generateBtn, dropBtn].forEach((btn) => {
      if (!btn) return;
      if (busy) {
        btn.setAttribute("disabled", "true");
      } else {
        btn.removeAttribute("disabled");
      }
    });
  }

  function bustCache(url) {
    if (!url) return url;
    const sep = url.indexOf("?") === -1 ? "?" : "&";
    return url + sep + "t=" + Date.now();
  }

  // Suit upload: multipart POST, then swap the image with a cache-busted URL.
  if (avatarForm) {
    avatarForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      const file = fileInput && fileInput.files && fileInput.files[0];
      if (!file) {
        setStatus("PICK A SUIT PHOTO FIRST");
        return;
      }

      setBusy(true);
      setStatus("GENERATING SUIT · STAND BY");

      try {
        const body = new FormData();
        body.append("avatar", file);
        const response = await fetch("/api/me/avatar", { method: "POST", body });
        const json = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(json.error || "avatar generation failed");
        }
        if (img && json.avatar_url) {
          img.src = bustCache(json.avatar_url);
        }
        setStatus("VISOR UP · FACE LOCKED");
        if (fileInput) fileInput.value = "";
      } catch (error) {
        setStatus(error instanceof Error ? `REJECTED · ${error.message}` : "REJECTED");
      } finally {
        setBusy(false);
      }
    });
  }

  // Drop visor: DELETE the generated avatar, fall back to the default suit.
  if (dropBtn) {
    dropBtn.addEventListener("click", async () => {
      setBusy(true);
      setStatus("DROPPING VISOR");
      try {
        const response = await fetch("/api/me/avatar", { method: "DELETE" });
        const json = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(json.error || "could not drop visor");
        }
        if (img) img.src = DEFAULT_SRC;
        setStatus("VISOR DOWN · DEFAULT SUIT");
      } catch (error) {
        setStatus(error instanceof Error ? `REJECTED · ${error.message}` : "REJECTED");
      } finally {
        setBusy(false);
      }
    });
  }

  // Profile PUT. The shared dashboard.js handler is hardcoded to POST, so the
  // profile form (display_name + bio) is driven here with a real PUT.
  const profileForm = document.getElementById("profile-form");
  if (profileForm) {
    profileForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      const statusNode = profileForm.querySelector("[data-form-status]");
      const button = profileForm.querySelector("button[type='submit']");
      if (statusNode) statusNode.textContent = "TRANSMITTING";
      if (button) button.setAttribute("disabled", "true");

      try {
        const payload = Object.fromEntries(new FormData(profileForm).entries());
        const response = await fetch(profileForm.action, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
        const json = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(json.error || "Request failed");
        }
        if (statusNode) statusNode.textContent = "ACCEPTED";
      } catch (error) {
        if (statusNode) {
          statusNode.textContent = error instanceof Error ? `REJECTED · ${error.message}` : "REJECTED";
        }
      } finally {
        if (button) button.removeAttribute("disabled");
      }
    });
  }
})();
