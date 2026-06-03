(function () {
  const body = document.body;
  const endpoint = body && body.dataset ? body.dataset.stateEndpoint : "";
  const eventsEndpoint = body && body.dataset ? body.dataset.eventsEndpoint : "";
  let eventSource;
  let renderedEventIds = new Set();
  let eventFeedRendered = false;

  function setField(name, value) {
    if (value === undefined || value === null) return;
    document.querySelectorAll(`[data-state-field="${name}"]`).forEach((el) => {
      el.textContent = String(value);
    });
  }

  function priceValue(price) {
    if (!price) return undefined;
    if (typeof price.display === "string") return price.display.replace(/^\$/, "");
    if (typeof price.cents === "number") return Math.round(price.cents / 100);
    return undefined;
  }

  function hydrateState(snapshot) {
    if (!snapshot || typeof snapshot !== "object") return;
    const metrics = snapshot.metrics || {};
    const nextSeat = snapshot.next_seat || {};
    const nextTier = metrics.next_tier_opens || {};

    setField("occupied", metrics.seats_occupied);
    setField("vacated", metrics.seats_vacated);
    setField("open", metrics.seats_open);
    setField("nextSeat", nextSeat.seat_number);
    setField("nextSeatPrice", priceValue(nextSeat.price));
    setField("nextTierPrice", priceValue(nextTier.price));
    setField("nextTier", nextTier.tier);
    setField("seatsUntilNextTier", nextTier.in_seats);
    setField("confirmedThisWeek", metrics.launches_confirmed_this_week);
    setField("confirmedLifetime", metrics.launches_confirmed_lifetime);
    setField("airlocksLifetime", metrics.airlocks_lifetime);
    setField("activeMissions", metrics.active_missions);

    updateNearestHatch(snapshot.nearest_hatch);

    if (Array.isArray(snapshot.seats)) {
      updateSeatGrids(snapshot.seats);
      updateMissionLog(snapshot);
    }
    updatePriceLadder(snapshot);
    if (Array.isArray(snapshot.recent_events)) {
      updateEventFeed(snapshot.recent_events);
    }
    tickCountdowns();
  }

  function updateNearestHatch(hatch) {
    document.querySelectorAll("[data-nearest-hatch]").forEach((el) => {
      if (!hatch || !hatch.deadline_at) {
        el.textContent = "NO ACTIVE HATCHES";
        return;
      }
      const handle = hatch.handle_display || (hatch.handle ? `@${hatch.handle}` : "@unknown");
      const seat = hatch.seat_label
        ? `${hatch.cohort === "THE_100" ? "SEAT" : "POST-100 SEAT"} ${hatch.seat_label}`
        : "SEAT --";
      el.innerHTML = [
        `${escapeHtml(handle)} · ${escapeHtml(seat)} · `,
        `<span data-countdown="${escapeHtml(hatch.deadline_at)}">calculating</span> until airlock`,
      ].join("");
    });
  }

  function updateSeatGrids(seats) {
    for (const seat of seats) {
      document.querySelectorAll(`[data-seat-number="${seat.seat_number}"]`).forEach((cell) => {
        const status = String(seat.status || "OPEN").toLowerCase();
        cell.classList.remove("is-occupied", "is-open", "is-vacated");
        cell.classList.add(`is-${status}`);
        cell.setAttribute("data-seat-status", seat.status || "OPEN");

        const handle = seat.occupant && seat.occupant.handle;
        const mission = seat.mission && seat.mission.declaration;
        const hatchText = seat.mission && seat.mission.deadline_at ? countdownText(seat.mission.deadline_at, "hatch") : "hatch pending";
        const vacatedAt = seat.vacated_at ? String(seat.vacated_at).slice(0, 10) : "date unknown";
        const price = seat.next_price && seat.next_price.display;
        const seatLabel = formatSeatLabel(seat);
        const tooltip = status === "open"
          ? `${seatLabel} · NEXT TO FILL AT ${price || "$672"}`
          : status === "vacated"
            ? `${seatLabel} · @${handle || "unknown"} · ${mission || "mission unknown"} · VACATED ${vacatedAt}`
            : `@${handle || "unknown"} · ${mission || "mission pending"} · ${hatchText}`;
        cell.setAttribute("data-tooltip", tooltip);
        cell.setAttribute("aria-label", tooltip);
        if (status === "open") {
          cell.setAttribute("href", "/manifest");
        } else if (handle) {
          cell.setAttribute("href", `/c/${encodeURIComponent(handle)}`);
        } else {
          cell.setAttribute("href", "/airlock");
        }

        const glyph = cell.querySelector("span");
        if (glyph) {
          glyph.textContent = status === "vacated"
            ? "x"
            : status === "occupied"
              ? "■"
              : padSeat(seat.seat_number || 0);
        }
      });
    }
  }

  function formatSeatLabel(seat) {
    const seatNumber = Number(seat && seat.seat_number) || 0;
    const seatLabel = seat && seat.seat_label ? String(seat.seat_label) : padSeat(seatNumber);
    return seatNumber <= 100
      ? `THE 100 · SEAT ${seatLabel}`
      : `SEAT ${seatLabel}`;
  }

  function padSeat(seatNumber) {
    return String(seatNumber).padStart(2, "0");
  }

  function updateEventFeed(events) {
    const feed = document.querySelector("[data-event-feed] .feed");
    if (!feed) return;
    const rows = events.slice(0, 8);
    feed.innerHTML = rows.map((event, index) => {
      const eventId = event.event_id || `${event.kind || "event"}-${event.occurred_at || index}`;
      const when = event.occurred_at ? new Date(event.occurred_at) : null;
      const time = when && !Number.isNaN(when.getTime())
        ? when.toISOString().slice(11, 19)
        : "--:--:--";
      const icon = event.kind === "AIRLOCKED" ? "v" : event.kind === "CONFIRMED" ? "^" : "->";
      const tone = event.kind === "AIRLOCKED" ? " airlock" : event.kind === "CONFIRMED" ? " launch" : "";
      const fresh = eventFeedRendered && !renderedEventIds.has(eventId) ? " is-new" : "";
      const handle = event.handle || "unknown";
      const handleToken = `@${handle}`;
      const message = String(event.message || "");
      const displayMessage = message.startsWith(handleToken)
        ? message.slice(handleToken.length).trim()
        : message;
      return [
        `<div class="entry${fresh}" data-event-id="${escapeHtml(eventId)}">`,
        `<span class="ts">${escapeHtml(time)}</span>`,
        `<span class="msg${tone}">${escapeHtml(icon)} <span class="who">@${escapeHtml(handle)}</span> ${escapeHtml(displayMessage)}</span>`,
        `<span class="tag">${escapeHtml(event.display_label || event.kind || "")}</span>`,
        "</div>",
      ].join("");
    }).join("");
    renderedEventIds = new Set(rows.map((event, index) => event.event_id || `${event.kind || "event"}-${event.occurred_at || index}`));
    eventFeedRendered = true;
  }

  function updateMissionLog(snapshot) {
    const log = document.querySelector("[data-mission-log]");
    const body = log && log.querySelector("[data-mission-log-body]");
    if (!body) return;

    const seats = Array.isArray(snapshot.seats) ? snapshot.seats : [];
    const rows = seats.filter((seat) => {
      return seat && seat.status !== "OPEN" && (seat.occupant || seat.mission);
    });
    if (!rows.length) {
      body.innerHTML = '<tr><td colspan="5"><div class="mission-text">No active missions yet.</div></td></tr>';
      setLogCount("active", 0);
      setLogCount("vacated", 0);
      setLogCount("showing", 0);
      setLogCount("total", 0);
      setupMissionLog();
      return;
    }

    body.innerHTML = rows.map((seat) => {
      const occupant = seat.occupant || {};
      const mission = seat.mission || {};
      const handle = occupant.handle || "unknown";
      const profilePath = `/c/${encodeURIComponent(handle)}`;
      const status = mission.status || seat.mission_status || seat.status || "DECLARED";
      const isAirlocked = status === "AIRLOCKED";
      const isVacated = isAirlocked || seat.status === "VACATED";
      const rowScope = isFreshAirlock(mission.airlocked_at) ? "active" : isVacated ? "vacated" : "active";
      const deadline = mission.deadline_at || "";
      const role = occupant.role || "founder";
      const seatNumber = Number(seat.seat_number || 0);
      const seatLabel = seatNumber <= 100
        ? `THE 100 · SEAT ${String(seatNumber).padStart(2, "0")}`
        : `SEAT ${seatNumber}`;
      return [
        `<tr${isVacated ? ' class="is-airlocked"' : ""} data-log-status="${rowScope}" data-href="${escapeHtml(profilePath)}" data-sort-seat="${seatNumber}" data-sort-handle="${escapeHtml(handle)}" data-sort-mission="${escapeHtml(mission.declaration || "mission pending")}" data-sort-deadline="${escapeHtml(deadline)}" data-sort-status="${escapeHtml(status)}">`,
        `<td class="seat-number-cell">${escapeHtml(seatLabel)}</td>`,
        "<td>",
        `<a class="crew" href="${escapeHtml(profilePath)}">`,
        '<canvas class="pixel-avatar" data-avatar="0" width="16" height="16"></canvas>',
        '<span class="crew-meta">',
        `<span class="crew-name">@${escapeHtml(handle)}</span>`,
        `<span class="crew-handle">${escapeHtml(role)}</span>`,
        "</span></a></td>",
        `<td><div class="mission-text">${escapeHtml(mission.declaration || "mission pending")}</div></td>`,
        "<td><div class=\"deadline\">",
        deadline ? `<span data-countdown="${escapeHtml(deadline)}">calculating</span>` : "<span>pending</span>",
        deadline ? `<span class="sub">deadline ${escapeHtml(formatDate(deadline))}</span>` : "",
        "</div></td>",
        `<td>${badgeHtml(status)}</td>`,
        "</tr>",
      ].join("");
    }).join("");

    const active = rows.filter((seat) => {
      const mission = seat.mission || {};
      const status = mission.status || seat.mission_status || seat.status;
      if (isFreshAirlock(mission.airlocked_at)) return true;
      return status !== "AIRLOCKED" && seat.status !== "VACATED";
    }).length;
    const vacated = rows.length - active;
    setLogCount("active", active);
    setLogCount("vacated", vacated);
    setLogCount("showing", rows.length);
    setLogCount("total", rows.length);
    renderPixelAvatars(body);
    setupMissionLog();
    sortMissionLog(log);
    applyMissionLogFilter(log);
  }

  function setLogCount(name, value) {
    document.querySelectorAll(`[data-log-count="${name}"]`).forEach((el) => {
      el.textContent = String(value);
    });
  }

  function isFreshAirlock(airlockedAt) {
    if (!airlockedAt) return false;
    const occurred = new Date(airlockedAt).getTime();
    if (!occurred || Number.isNaN(occurred)) return false;
    return Date.now() - occurred < 24 * 60 * 60 * 1000;
  }

  function badgeHtml(status) {
    const normalized = String(status || "DECLARED").toLowerCase().replace(/_/g, "-");
    const label = String(status || "DECLARED").replace(/_/g, " ");
    return `<span class="badge ${escapeHtml(normalized)}"><i class="b-dot"></i>${escapeHtml(label)}</span>`;
  }

  function formatDate(iso) {
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return iso;
    return date.toLocaleDateString("en", { timeZone: "UTC", year: "numeric", month: "2-digit", day: "2-digit" });
  }

  function updatePriceLadder(snapshot) {
    const tiers = Array.isArray(snapshot.price_tiers) ? snapshot.price_tiers : [];
    let totalFilled = 0;
    let totalSeats = 0;
    for (const tier of tiers) {
      const filled = firstNumber(tier.occupied, tier.filled);
      const capacity = firstNumber(tier.capacity, tier.seats);
      updateTierRow(tier.tier || tier.id, filled, capacity, tier);
      if (typeof filled === "number") totalFilled += filled;
      if (typeof capacity === "number") totalSeats += capacity;
    }

    if (!tiers.length && snapshot.tier_progress) {
      updateTierRow(snapshot.tier_progress.tier, snapshot.tier_progress.occupied, snapshot.tier_progress.capacity);
    }

    if (tiers.length) {
      document.querySelectorAll("[data-ladder-total-filled]").forEach((el) => {
        el.textContent = String(totalFilled);
      });
      document.querySelectorAll("[data-ladder-total-seats]").forEach((el) => {
        el.textContent = String(totalSeats);
      });
    } else if (snapshot.metrics && typeof snapshot.metrics.seats_occupied === "number") {
      document.querySelectorAll("[data-ladder-total-filled]").forEach((el) => {
        el.textContent = String(snapshot.metrics.seats_occupied);
      });
    }
  }

  function updateTierRow(tierId, filled, capacity, tier) {
    if (!tierId || typeof filled !== "number" || typeof capacity !== "number" || capacity <= 0) return;
    const row = document.querySelector(`[data-ladder-tier="${CSS.escape(String(tierId).padStart(2, "0"))}"]`);
    if (!row) return;
    const percent = Math.max(0, Math.min(100, Math.round((filled / capacity) * 100)));
    const filledEl = row.querySelector("[data-ladder-filled]");
    const capacityEl = row.querySelector("[data-ladder-capacity]");
    const bar = row.querySelector("[data-ladder-bar]");
    const priceEl = row.querySelector("[data-ladder-price]");
    const rangeEl = row.querySelector("[data-ladder-range]");
    if (filledEl) filledEl.textContent = String(filled);
    if (capacityEl) capacityEl.textContent = String(capacity);
    if (bar) bar.style.width = `${percent}%`;
    if (tier && priceEl) priceEl.textContent = displayPrice(tier.price);
    if (tier && rangeEl) rangeEl.textContent = tier.seat_range || tier.range || rangeEl.textContent;
    row.setAttribute("aria-label", `Tier ${String(tierId).padStart(2, "0")} ${filled} of ${capacity} filled`);
  }

  function firstNumber() {
    for (const value of arguments) {
      if (typeof value === "number") return value;
    }
    return undefined;
  }

  function displayPrice(price) {
    if (!price) return "";
    if (typeof price.display === "string") return price.display;
    if (typeof price.cents === "number") return `$${Math.round(price.cents / 100)}`;
    if (typeof price === "number") return `$${price}`;
    return "";
  }

  function renderPixelAvatars(root) {
    if (!window.__airlockPaintAvatars) return;
    window.__airlockPaintAvatars(root);
  }

  function escapeHtml(value) {
    return String(value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function tickCountdowns() {
    const now = Date.now();
    document.querySelectorAll("[data-countdown]").forEach((el) => {
      const value = countdownText(el.getAttribute("data-countdown") || "", "", now);
      if (value) el.textContent = value;
    });
  }

  function countdownText(deadline, prefix, now) {
    const target = new Date(deadline || "").getTime();
    if (!target || Number.isNaN(target)) return "";
    const remaining = target - (typeof now === "number" ? now : Date.now());
    if (remaining <= 0) return "HATCH CLOSED";
    const totalSeconds = Math.floor(remaining / 1000);
    const days = Math.floor(totalSeconds / 86400);
    const hours = Math.floor((totalSeconds % 86400) / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;
    const label = prefix ? `${prefix} in ` : "";
    if (days > 0) return `${label}${days}d ${String(hours).padStart(2, "0")}h`;
    if (hours > 0) return `${label}${hours}h ${String(minutes).padStart(2, "0")}m`;
    return `${label}${minutes}m ${String(seconds).padStart(2, "0")}s`;
  }

  function setupMissionLog() {
    document.querySelectorAll("[data-mission-log]").forEach((log) => {
      if (log.dataset.bound === "true") return;
      log.dataset.bound = "true";
      const tabs = log.querySelectorAll("[data-tab]");
      tabs.forEach((tab) => {
        tab.addEventListener("click", () => {
          tabs.forEach((other) => other.classList.toggle("active", other === tab));
          applyMissionLogFilter(log);
        });
      });
      log.querySelectorAll("[data-sort-key]").forEach((button) => {
        button.addEventListener("click", () => {
          const alreadyActive = button.classList.contains("is-active");
          const nextDir = alreadyActive && button.dataset.sortDir === "ASC" ? "DESC" : "ASC";
          log.querySelectorAll("[data-sort-key]").forEach((other) => {
            other.classList.toggle("is-active", other === button);
            if (other !== button) other.removeAttribute("data-sort-dir");
          });
          button.dataset.sortDir = nextDir;
          sortMissionLog(log);
          applyMissionLogFilter(log);
        });
      });
      log.addEventListener("click", (event) => {
        const row = event.target && event.target.closest ? event.target.closest("[data-href]") : null;
        if (!row) return;
        const href = row.getAttribute("data-href");
        if (href) window.location.href = href;
      });
    });
  }

  function sortMissionLog(log) {
    const active = log.querySelector("[data-sort-key].is-active");
    if (!active) return;
    const key = active.getAttribute("data-sort-key");
    const dir = active.dataset.sortDir === "DESC" ? -1 : 1;
    const body = log.querySelector("[data-mission-log-body]");
    if (!key || !body) return;
    const rows = Array.from(body.querySelectorAll("[data-log-status]"));
    rows.sort((a, b) => compareSortValue(a, b, key) * dir);
    rows.forEach((row) => body.appendChild(row));
  }

  function compareSortValue(a, b, key) {
    const left = a.getAttribute(`data-sort-${key}`) || "";
    const right = b.getAttribute(`data-sort-${key}`) || "";
    if (key === "seat") {
      return (Number(left) || 0) - (Number(right) || 0);
    }
    return left.localeCompare(right);
  }

  function applyMissionLogFilter(log) {
    const selected = (log.querySelector("[data-tab].active") || log.querySelector("[data-tab]"))?.getAttribute("data-tab") || "active";
    log.querySelectorAll("[data-log-status]").forEach((row) => {
      const visible = selected === "all" || row.getAttribute("data-log-status") === selected;
      row.classList.toggle("is-hidden", !visible);
    });
    const showing = Array.from(log.querySelectorAll("[data-log-status]")).filter((row) => !row.classList.contains("is-hidden")).length;
    const showingEl = log.querySelector('[data-log-count="showing"]');
    if (showingEl) showingEl.textContent = String(showing);
  }

  async function fetchState() {
    if (!endpoint) return;
    try {
      const response = await fetch(endpoint, { headers: { accept: "application/json" } });
      if (!response.ok) return;
      hydrateState(await response.json());
    } catch (_) {
      // Local web dev may run without the Go API. Static launch data remains authoritative.
    }
  }

  function connectEventStream() {
    if (!eventsEndpoint || typeof EventSource === "undefined") return;
    eventSource = new EventSource(eventsEndpoint);
    eventSource.addEventListener("state", (event) => {
      try {
        hydrateState(JSON.parse(event.data));
      } catch (_) {
        // Ignore malformed stream frames; the polling fallback will catch up.
      }
    });
    eventSource.addEventListener("error", () => {
      // EventSource reconnects automatically. Polling remains the fallback.
    });
    window.addEventListener("beforeunload", () => {
      if (eventSource) eventSource.close();
    }, { once: true });
  }

  setupMissionLog();
  tickCountdowns();
  setInterval(tickCountdowns, 1000);
  connectEventStream();
  fetchState();
  setInterval(fetchState, 30000);
})();
