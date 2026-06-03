// =========================================================
// The Airlock — Hero Scene
// Single-scene pixel-art: space cruiser at bottom-right,
// scroll progress drives:
//   - storm intensity (solar flares, asteroids, ship warning lights)
//   - airlock door opening
//   - astronaut being ejected and drifting away
// =========================================================

(function () {
  const canvas = document.getElementById('cinematic-canvas');
  const stage = document.querySelector('.cinematic-stage');
  const pin = document.querySelector('.cinematic-pin');
  if (!canvas || !stage) return;

  const W = 480, H = 270;
  const SCENE_SAFE_W = 560;
  let viewW = W, viewH = H;
  let sceneX = 0, sceneY = 0;
  let viewportStars = [];
  canvas.width = W;
  canvas.height = H;
  const ctx = canvas.getContext('2d');
  ctx.imageSmoothingEnabled = false;
  const textMeasureCtx = document.createElement('canvas').getContext('2d');

  // ---------- palette ----------
  const C = {
    BG:        '#07090C',
    BG_STORM:  '#1A1410',
    SPACE:     '#04060A',
    HULL_D:    '#15181F',
    HULL:      '#2A2F38',
    HULL_H:    '#3F4654',
    HULL_HH:   '#5A6172',
    METAL:     '#7A8294',
    GLASS:     '#5BA8FF',
    GLASS_D:   '#2A5080',
    LANTERN:   '#FFD25C',
    LANTERN_D: '#A87B2A',
    TEXT:      '#E7EAF0',
    MUTED:     '#8A93A4',
    DIM:       '#5A6371',
    HATCH:     '#FF6651',
    HATCH_D:   '#A03828',
    HATCH_DD:  '#5C1F12',
    SIGNAL:    '#5BFFA8',
    SERIAL:    '#FFD25C',
    PLANET_A:  '#5B3A8A',
    PLANET_B:  '#3A2B5C',
    NEBULA:    '#241836',
  };

  // ---------- helpers ----------
  function r(x, y, w, h, col) {
    ctx.fillStyle = col;
    ctx.fillRect(x | 0, y | 0, Math.max(0, w | 0), Math.max(0, h | 0));
  }
  function px(x, y, col) { r(x, y, 1, 1, col); }
  function clamp(v, a, b) { return v < a ? a : v > b ? b : v; }
  function lerp(a, b, t) { return a + (b - a) * clamp(t, 0, 1); }
  function ease(t) { return t < .5 ? 2*t*t : 1 - Math.pow(-2*t + 2, 2) / 2; }
  function easeOut(t) { return 1 - Math.pow(1 - clamp(t, 0, 1), 3); }
  function canvasFont(el) {
    const cs = getComputedStyle(el);
    return `${cs.fontStyle} ${cs.fontVariant} ${cs.fontWeight} ${cs.fontSize} ${cs.fontFamily}`;
  }
  function inkLeftBearing(el) {
    if (!textMeasureCtx || !el) return 0;
    textMeasureCtx.font = canvasFont(el);
    const metrics = textMeasureCtx.measureText(el.textContent || '');
    return Number.isFinite(metrics.actualBoundingBoxLeft) ? metrics.actualBoundingBoxLeft : 0;
  }

  // ---------- deterministic random helpers ----------
  function mulberry32(seed) {
    return function() {
      let t = seed += 0x6D2B79F5;
      t = Math.imul(t ^ t >>> 15, t | 1);
      t ^= t + Math.imul(t ^ t >>> 7, t | 61);
      return ((t ^ t >>> 14) >>> 0) / 4294967296;
    };
  }

  function rebuildViewportStars() {
    const rng = mulberry32(4242 + viewW * 31 + viewH * 17);
    const density = (viewW * viewH) / (W * H);
    const count = Math.max(90, Math.min(180, Math.round(density * 34)));
    viewportStars = [];
    for (let i = 0; i < count; i++) {
      viewportStars.push({
        x: rng() * viewW,
        y: rng() * viewH,
        bright: rng(),
        twinkle: rng() * 6.28,
      });
    }
  }

  function syncCanvasSize() {
    const cssW = Math.max(1, canvas.clientWidth || window.innerWidth || W);
    const cssH = Math.max(1, canvas.clientHeight || window.innerHeight || H);
    const cssAspect = cssW / cssH;
    const sceneAspect = W / H;
    let nextW;
    let nextH;

    nextW = Math.max(SCENE_SAFE_W, Math.round(H * cssAspect));
    nextH = Math.max(H, Math.round(nextW / cssAspect));

    if (canvas.width !== nextW || canvas.height !== nextH || viewportStars.length === 0) {
      canvas.width = nextW;
      canvas.height = nextH;
      ctx.imageSmoothingEnabled = false;
      viewW = nextW;
      viewH = nextH;
      rebuildViewportStars();
    }

    const extraX = viewW - W;
    const extraY = viewH - H;
    const portraitBias = cssAspect < 0.75 ? 0.58 : 0.5;
    sceneX = Math.min(Math.round(extraX * 0.5), Math.max(0, viewW - 540));
    sceneY = Math.round(extraY * portraitBias);
  }

  function drawViewportStars(stormIntensity) {
    const now = performance.now() / 1000;
    for (const s of viewportStars) {
      const twinkle = (Math.sin(now * 1.5 + s.twinkle) + 1) / 2;
      const col = s.bright > 0.86 ? C.TEXT : (s.bright > 0.55 ? '#B6BCC8' : C.DIM);
      const yOffset = stormIntensity > 0.3 ? Math.sin(now * 4 + s.x) * stormIntensity * 0.6 : 0;
      if (twinkle > 0.35 || s.bright > 0.62) {
        px(s.x | 0, (s.y + yOffset) | 0, col);
      }
    }
  }

  // ---------- stars ----------
  const STARS = [];
  {
    const rng = mulberry32(1337);
    for (let i = 0; i < 90; i++) {
      STARS.push({
        x: rng() * W,
        y: rng() * (H * 0.85),
        layer: Math.floor(rng() * 3),
        bright: rng(),
        twinkle: rng() * 6.28,
      });
    }
  }
  function drawStars(t, stormIntensity) {
    const now = performance.now() / 1000;
    for (const s of STARS) {
      const twinkle = (Math.sin(now * 1.5 + s.twinkle) + 1) / 2;
      const col = s.bright > 0.8 ? C.TEXT : (s.bright > 0.5 ? '#B6BCC8' : C.DIM);
      // stars get more "agitated" during storm
      const yOffset = stormIntensity > 0.3 ? Math.sin(now * 4 + s.x) * stormIntensity * 0.6 : 0;
      if (twinkle > 0.3 || s.bright > 0.6) {
        px(s.x | 0, (s.y + yOffset) | 0, col);
      }
    }
  }

  // ---------- distant planet ----------
  function drawPlanet(cx, cy, radius, palette) {
    for (let dy = -radius; dy <= radius; dy++) {
      for (let dx = -radius; dx <= radius; dx++) {
        const d = Math.sqrt(dx*dx + dy*dy);
        if (d > radius) continue;
        const shade = (dx - dy * 0.6) / (radius * 1.3);
        let col;
        if (shade > 0.45) col = palette[0];
        else if (shade > 0.05) col = palette[1];
        else if (shade > -0.35) col = palette[2];
        else col = palette[3];
        // surface bands
        if (Math.floor((dy + radius) / 3) % 3 === 0 && shade > -0.2) {
          col = palette[Math.max(0, palette.indexOf(col) - 1)] || col;
        }
        px(cx + dx, cy + dy, col);
      }
    }
  }

  // ---------- nebula puff (cloud-like) ----------
  function drawNebula(cx, cy, radius, color, alpha) {
    ctx.save();
    ctx.globalAlpha = alpha;
    for (let i = 0; i < 60; i++) {
      const a = (i / 60) * Math.PI * 2;
      const r2 = radius * (0.6 + 0.4 * Math.sin(i * 7.3));
      const x = cx + Math.cos(a) * r2;
      const y = cy + Math.sin(a) * r2 * 0.5;
      r(x - 1, y - 1, 3, 3, color);
    }
    ctx.restore();
  }

  // ---------- asteroid streaks during storm ----------
  function drawAsteroidStreaks(stormIntensity) {
    if (stormIntensity <= 0.2) return;
    const now = performance.now() / 1000;
    ctx.save();
    ctx.globalAlpha = clamp(stormIntensity - 0.2, 0, 1);
    for (let i = 0; i < 6; i++) {
      const seed = i * 73.1 + Math.floor(now * 1.5);
      const yStart = ((seed * 37) % viewH) | 0;
      const speed = 60 + (i % 3) * 30;
      const cycle = (now * speed + i * 50) % (viewW + 80);
      const x = cycle - 40;
      // streak tail trails behind the rightward-moving head
      for (let dx = 0; dx < 12; dx++) {
        const alpha = 1 - dx / 12;
        ctx.fillStyle = `rgba(255,210,92,${alpha * 0.7})`;
        ctx.fillRect((x - dx) | 0, yStart | 0, 1, 1);
      }
      // head
      r(x | 0, yStart | 0, 2, 2, C.SERIAL);
    }
    ctx.restore();
  }

  // ---------- solar flare lightning ----------
  // crackles across the sky during storm. Flashes briefly.
  function drawFlare(stormIntensity) {
    if (stormIntensity <= 0.25) return;
    const now = performance.now();
    // flash every ~1.4 seconds during storm, more frequent + more intense
    const cycle = Math.floor(now / 1400);
    const inFlash = (now % 1400) < 280;
    if (!inFlash) return;
    ctx.save();
    const flashStrength = (280 - (now % 1400)) / 280;
    ctx.globalAlpha = flashStrength * stormIntensity * 1.2;

    // big jagged crack from top
    const rng = mulberry32(cycle * 91 + 7);
    let x = rng() * viewW * 0.7 + viewW * 0.15;
    let y = 0;
    ctx.strokeStyle = '#FFEEC0';
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(x | 0, y | 0);
    for (let i = 0; i < 22; i++) {
      const dx = (rng() - 0.5) * 22;
      const dy = 5 + rng() * 9;
      x += dx; y += dy;
      ctx.lineTo(x | 0, y | 0);
      // forks
      if (rng() < 0.3 && y < viewH * 0.8) {
        const fx = x + (rng() - 0.5) * 36;
        const fy = y + 6 + rng() * 14;
        ctx.lineTo(fx | 0, fy | 0);
        ctx.moveTo(x | 0, y | 0);
      }
    }
    ctx.stroke();
    // second smaller crack
    if (rng() > 0.4) {
      ctx.lineWidth = 1;
      x = rng() * viewW * 0.8 + viewW * 0.1;
      y = 0;
      ctx.beginPath();
      ctx.moveTo(x | 0, y | 0);
      for (let i = 0; i < 12; i++) {
        x += (rng() - 0.5) * 16;
        y += 6 + rng() * 8;
        ctx.lineTo(x | 0, y | 0);
      }
      ctx.stroke();
    }
    // flash overlay
    ctx.globalAlpha = flashStrength * stormIntensity * 0.5;
    ctx.fillStyle = '#FFEEC0';
    ctx.fillRect(0, 0, viewW, viewH);
    ctx.restore();
  }

  // ============================================================
  // SHIP — side-view cruiser, nose pointing LEFT
  // Anchored between center and right-bottom of canvas.
  // Built as a single unified silhouette: every subsystem shares
  // an edge with the next so the ship reads as one object.
  //
  //   ╔═══════════════════════════════════════════╗
  //   ║   dorsal spine ─ antenna │ dish │ flag    ║
  //   ╠═══════════════════════════════════════════╣
  //   ║ cockpit ▷ ▥▥▥▥ hull ▥▥▥▥▥ ╲ engine ▥▥▥ ⊳ ║
  //   ║          ▼ airlock ▼                      ║
  //   ╚═══════════════════════════════════════════╝
  // ============================================================
  function drawShip(stormIntensity) {
    window.AirlockCruiser?.draw(ctx, {
      stormIntensity,
      enginesOn: true,
      motion: true,
      x: 210,
      y: 135,
    });
  }

  // ============================================================
  // ASTRONAUT (12x18 sprite) — drawn when ejected
  // drifts down-left from airlock position, spinning slightly
  // ============================================================
  function drawAstronautEjected(progress) {
    if (progress <= 0) return;
    const t = easeOut(progress);
    // Start at the side-hatch opening: ship anchor (210,135) + hatch center (104, 39).
    // Astronaut emerges from the LEFT edge of the hatch and drifts away.
    const sx = 300, sy = 174;
    // Zero-g push: mostly horizontal-left with subtle downward drift + wobble.
    const ex = sx - 230 * t;
    const ey = sy + 20 * t + Math.sin(progress * 3.5) * 6;
    const x = ex | 0, y = ey | 0;
    const rot = t * 1.6 + Math.sin(t * 6.5) * 0.28;

    ctx.save();
    ctx.translate(x + 6, y + 8);
    ctx.rotate(rot);
    // Helmet
    r(-4, -10, 8, 7, C.TEXT);
    r(-3, -11, 6, 1, C.TEXT);
    r(-3, -9, 6, 4, C.HULL_D);
    r(-2, -8, 4, 2, C.GLASS_D);
    r(-2, -8, 4, 1, C.GLASS);
    r(-1, -8, 1, 1, '#FFFFFF');
    r(2, -8, 1, 1, C.SIGNAL);
    // Suit / chest
    r(-4, -3, 8, 6, C.TEXT);
    r(-4, -3, 8, 1, C.MUTED);
    r(-1, -1, 2, 2, C.HATCH);
    r(-3, 0, 1, 1, C.HULL_D);
    r(2, 0, 1, 1, C.SIGNAL);
    // Shoulder pads
    r(-5, -3, 1, 3, C.MUTED);
    r(4, -3, 1, 3, C.MUTED);
    // Arms flailing
    r(-7, -2, 2, 3, C.TEXT);
    r(-8, -4, 2, 2, C.TEXT);
    r(4, -2, 3, 2, C.TEXT);
    r(6, -4, 2, 2, C.TEXT);
    r(7, -2, 1, 1, C.HULL_D);
    r(-8, -4, 1, 1, C.HULL_D);
    // Legs splayed
    r(-3, 3, 2, 6, C.TEXT);
    r(1, 3, 2, 6, C.TEXT);
    r(-4, 8, 3, 2, C.MUTED);
    r(1, 8, 3, 2, C.MUTED);
    r(-4, 9, 1, 1, C.HULL_D);
    r(3, 9, 1, 1, C.HULL_D);
    // Backpack
    r(-2, -1, 4, 1, C.MUTED);
    ctx.restore();

    // Oxygen-puff debris trail back toward the airlock
    for (let i = 0; i < 10; i++) {
      const tt = clamp(t - i * 0.05, 0, 1);
      if (tt <= 0) continue;
      const tx = sx - 230 * tt + (i % 2 ? 2 : -2);
      const ty = sy + 20 * tt + Math.sin((tt + i) * 4) * 4 + i * 0.4;
      ctx.save();
      ctx.globalAlpha = (1 - i / 10) * 0.6;
      const col = i < 3 ? '#FFFFFF' : (i < 6 ? C.TEXT : (i < 8 ? C.GLASS : C.DIM));
      r(tx, ty, 1, 1, col);
      ctx.restore();
    }

    // Initial ejection blast — pressurized air puff + motion-blur stripes
    if (progress < 0.18) {
      ctx.save();
      ctx.globalAlpha = (0.18 - progress) * 5;
      for (let i = 0; i < 8; i++) {
        const pa = Math.PI * 0.4 + Math.random() * Math.PI * 1.2;
        const pd = 4 + Math.random() * 10;
        r(sx + Math.cos(pa) * pd, sy + Math.sin(pa) * pd, 1, 1, '#FFFFFF');
      }
      ctx.globalAlpha = (0.18 - progress) * 4;
      for (let i = 0; i < 4; i++) {
        r(sx - 14 - i * 5, sy + i, 5, 1, '#FFEEC0');
      }
      ctx.restore();
    }
  }

  // ---------- asteroid belt band ----------
  // Tiny pixel rocks drift across the whole responsive viewport. The old
  // hard horizon line exposed the fixed-size scene rect on non-16:9 screens.
  const BELT = [];
  {
    const rng = mulberry32(9001);
    for (let i = 0; i < 50; i++) {
      BELT.push({
        x: rng(),
        y: rng() * 12,
        size: rng() > 0.7 ? 2 : 1,
        speed: 0.3 + rng() * 0.4,
        bright: rng(),
      });
    }
  }
  function drawAsteroidBelt(p) {
    const now = performance.now() / 1000;
    for (const a of BELT) {
      const x = (a.x * viewW + now * a.speed * 20) % viewW;
      const y = Math.min(viewH - 2, sceneY + 255 + a.y);
      const col = a.bright > 0.7 ? C.HULL_HH : (a.bright > 0.4 ? C.HULL : C.HULL_D);
      r(x, y, a.size, a.size, col);
      // occasional spark
      if (a.bright > 0.85 && Math.sin(now * 3 + a.x) > 0.7) {
        r(x, y - 1, 1, 1, C.SERIAL);
      }
    }
  }

  // ============================================================
  // FOREGROUND VIGNETTE / SCAN
  // ============================================================
  function drawAtmosphere(stormIntensity) {
    // top sky tint — warmer reds as storm builds
    if (stormIntensity > 0) {
      const warm = ctx.createLinearGradient(0, 0, 0, viewH * 0.68);
      warm.addColorStop(0, `rgba(60,15,8,${stormIntensity * 0.55})`);
      warm.addColorStop(0.58, `rgba(60,15,8,${stormIntensity * 0.34})`);
      warm.addColorStop(1, 'rgba(60,15,8,0)');
      ctx.fillStyle = warm;
      ctx.fillRect(0, 0, viewW, viewH * 0.68);

      const top = ctx.createLinearGradient(0, 0, 0, viewH * 0.28);
      top.addColorStop(0, `rgba(20,5,3,${stormIntensity * 0.42})`);
      top.addColorStop(1, 'rgba(20,5,3,0)');
      ctx.fillStyle = top;
      ctx.fillRect(0, 0, viewW, viewH * 0.28);
    }
  }

  // ============================================================
  // MAIN DRAW
  // ============================================================
  function draw(p) {
    syncCanvasSize();
    ctx.setTransform(1, 0, 0, 1, 0, 0);

    // base background — shifts to warmer tone during storm
    const bgCol = p > 0.1 ? `rgb(${lerp(7, 26, p*1.5)|0}, ${lerp(9, 20, p*1.5)|0}, ${lerp(12, 16, p*1.5)|0})` : C.BG;
    ctx.fillStyle = bgCol;
    ctx.fillRect(0, 0, viewW, viewH);
    drawViewportStars(p);

    // distant planet (lower-right, away from title area)
    drawPlanet(sceneX + 440, sceneY + 95, 22, [C.PLANET_A, C.PLANET_B, '#1F1633', '#0D0820']);

    // asteroid streaks during storm
    drawAsteroidStreaks(p);

    // sky tint
    drawAtmosphere(p);

    // asteroid belt band
    drawAsteroidBelt(p);

    ctx.save();
    ctx.translate(sceneX, sceneY);

    // ship
    drawShip(p);

    // ejected astronaut (only after airlock opens)
    const ejection = clamp((p - 0.55) / 0.45, 0, 1);
    drawAstronautEjected(ejection);

    // solar flare lightning
    ctx.restore();
    drawFlare(p);

    // subtle global scanlines (very faint for pixel feel)
    ctx.save();
    ctx.globalAlpha = 0.04;
    for (let yy = 0; yy < viewH; yy += 2) {
      ctx.fillStyle = '#000';
      ctx.fillRect(0, yy, viewW, 1);
    }
    ctx.restore();
  }

  // ============================================================
  // SCROLL LOOP
  // ============================================================
  let progress = 0;
  window.__heroProgress = () => progress;

  function getProgress() {
    const rect = stage.getBoundingClientRect();
    const total = stage.offsetHeight - window.innerHeight;
    if (total <= 0) return 0;
    return clamp(-rect.top / total, 0, 1);
  }

  function syncSubtitleIndent(titleEl, subEl, customProp) {
    if (!titleEl || !subEl) return;
    const titleBox = titleEl.getBoundingClientRect();
    const subBox = subEl.getBoundingClientRect();
    const titleInkLeft = titleBox.left - inkLeftBearing(titleEl);
    const indent = titleInkLeft - subBox.left + inkLeftBearing(subEl);
    subEl.style.setProperty(customProp, `${Math.max(0, indent).toFixed(2)}px`);
  }

  function syncOverlay(p) {
    // Reveal "OR AIRLOCK" + alt subtitle as scroll progresses
    const titleMain = document.querySelector('.hero-title-main');
    const altSlot = document.querySelector('.hero-alt-slot');
    const titleAlt = document.querySelector('.hero-title-alt');
    const subDefault = document.querySelector('.hero-sub-default');
    const subAlt = document.querySelector('.hero-sub-alt');
    const scrollPrompt = document.querySelector('.scroll-prompt');

    syncSubtitleIndent(titleMain, subDefault, '--hero-sub-default-indent');
    syncSubtitleIndent(titleAlt, subAlt, '--hero-sub-alt-indent');

    // OR AIRLOCK reveal starts ~0.10, fully in by 0.55
    const tReveal = clamp((p - 0.10) / 0.45, 0, 1);
    if (titleAlt && altSlot) {
      // measure natural height once (cache on element)
      const titleAltStyle = getComputedStyle(titleAlt);
      const fontStatus = document.fonts?.status || 'no-font-api';
      const heightMeasureKey = `${titleAltStyle.fontSize}|${titleAltStyle.fontFamily}|${fontStatus}`;
      if (!titleAlt.__naturalHeight || titleAlt.__naturalHeightKey !== heightMeasureKey) {
        // temporarily expand to measure
        altSlot.style.height = 'auto';
        const measuredHeight = titleAlt.offsetHeight;
        const lineHeight = parseFloat(titleAltStyle.lineHeight);
        const fontSize = parseFloat(titleAltStyle.fontSize);
        const maxExpectedHeight = Number.isFinite(lineHeight) ? lineHeight : fontSize;
        titleAlt.__naturalHeight = Number.isFinite(maxExpectedHeight)
          ? Math.min(measuredHeight, maxExpectedHeight)
          : measuredHeight;
        titleAlt.__naturalHeightKey = heightMeasureKey;
        altSlot.style.height = '0px';
      }
      const nh = titleAlt.__naturalHeight;
      altSlot.style.height = (nh * tReveal) + 'px';
      titleAlt.style.opacity = tReveal;
      titleAlt.style.transform = `translateY(${(1 - tReveal) * 40}px)`;
    }
    // subtitle crossfade
    if (subDefault) {
      const fadeOut = clamp((p - 0.05) / 0.20, 0, 1);
      subDefault.style.opacity = 1 - fadeOut;
    }
    if (subAlt) {
      const fadeIn = clamp((p - 0.30) / 0.25, 0, 1);
      subAlt.style.opacity = fadeIn;
      subAlt.style.transform = `translateY(${(1 - fadeIn) * 8}px)`;
    }
    if (scrollPrompt) {
      scrollPrompt.style.opacity = clamp(1 - p * 6, 0, 1);
    }
  }

  function loop() {
    progress = getProgress();
    const stageBot = stage.getBoundingClientRect().bottom;
    const hide = stageBot <= window.innerHeight * 0.05;
    if (pin) pin.classList.toggle('is-hidden', hide);
    if (!hide) draw(progress);
    syncOverlay(progress);
    requestAnimationFrame(loop);
  }
  loop();

  // Backup: ensure progress updates on scroll even if rAF is throttled.
  function manualUpdate() {
    progress = getProgress();
    syncOverlay(progress);
    const stageBot = stage.getBoundingClientRect().bottom;
    const hide = stageBot <= window.innerHeight * 0.05;
    if (pin) pin.classList.toggle('is-hidden', hide);
    if (!hide) draw(progress);
  }
  window.addEventListener('scroll', manualUpdate, { passive: true });
  window.addEventListener('resize', () => {
    const titleAlt = document.querySelector('.hero-title-alt');
    if (titleAlt) {
      titleAlt.__naturalHeight = 0;
      titleAlt.__naturalHeightKey = '';
    }
    manualUpdate();
  }, { passive: true });
  if (document.fonts?.ready) {
    document.fonts.ready.then(() => {
      const titleAlt = document.querySelector('.hero-title-alt');
      if (titleAlt) {
        titleAlt.__naturalHeight = 0;
        titleAlt.__naturalHeightKey = '';
      }
      manualUpdate();
    }).catch(() => {});
  }
  // poll every 100ms as a last-resort backup (covers preview-pane environments
  // where rAF and scroll events are both throttled)
  setInterval(manualUpdate, 100);
})();
