(function () {
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

  function clamp(v, a, b) {
    return v < a ? a : v > b ? b : v;
  }

  function draw(ctx, opts = {}) {
    const stormIntensity = opts.stormIntensity ?? 0;
    const enginesOn = opts.enginesOn ?? true;
    const motion = opts.motion ?? true;
    const now = opts.now ?? performance.now() / 1000;
    const bob = motion ? Math.sin(now * 0.7) * 0.5 + Math.sin(now * 0.35) * 0.3 : 0;
    const tilt = motion && stormIntensity > 0.4 ? Math.sin(now * 2.5) * stormIntensity * 1.2 : 0;
    const oy = (bob + tilt) | 0;
    const ox = motion && stormIntensity > 0.4 ? (Math.sin(now * 3) * stormIntensity * 0.6) | 0 : 0;

    function r(x, y, w, h, col) {
      ctx.fillStyle = col;
      ctx.fillRect(x | 0, y | 0, Math.max(0, w | 0), Math.max(0, h | 0));
    }

    // Anchor — visual center lands around canvas (340, 170) → screen ~71% across, 63% down.
    const AX = opts.x ?? 210;
    const AY = opts.y ?? 135;
    const X = AX + ox;
    const Y = AY + oy;

    // ============== DORSAL SPINE (continuous top beam) ==============
    // Trapezoidal beam — sits flush against hull top edge at y = Y+16.
    // Drawn brighter than the hull underside so it reads against deep space.
    r(X + 48, Y + 4, 159, 12, C.HULL);              // main spine body (medium)
    r(X + 50, Y + 4, 155, 1, C.METAL);              // top bright edge
    r(X + 49, Y + 5, 157, 1, C.HULL_HH);            // sub-highlight
    r(X + 48, Y + 14, 159, 1, C.HULL_D);            // bottom shadow
    r(X + 48, Y + 15, 159, 1, '#0A0C12');           // bottom dark seam
    // Tapered end shoulders that blend the spine into the hull top
    r(X + 46, Y + 6, 2, 10, C.HULL);
    r(X + 207, Y + 6, 2, 10, C.HULL);
    r(X + 44, Y + 8, 2, 8, C.HULL_D);
    r(X + 209, Y + 8, 2, 8, C.HULL_D);
    // Service hatches + vents along the spine
    for (let i = 0; i < 6; i++) {
      const sxh = X + 56 + i * 26;
      r(sxh, Y + 8, 6, 4, C.HULL_D);                // recessed hatch
      r(sxh, Y + 8, 6, 1, '#0A0C12');
      r(sxh + 1, Y + 9, 4, 2, C.HULL);
      r(sxh + 1, Y + 9, 1, 1, C.HULL_HH);
    }
    // Anchor rivets (top + bottom of spine)
    for (let i = 0; i < 8; i++) {
      r(X + 52 + i * 20, Y + 6, 1, 1, C.HULL_HH);
      r(X + 52 + i * 20, Y + 13, 1, 1, C.HULL_HH);
    }

    // ============== ANTENNA MAST + BEACON ==============
    // Mast extends into the spine for a clean structural join.
    r(X + 75, Y - 9, 1, 17, C.METAL);
    r(X + 73, Y - 9, 5, 1, C.METAL);
    r(X + 74, Y - 7, 3, 1, C.METAL);
    // Mounting collar at spine top
    r(X + 73, Y + 4, 5, 2, C.HULL_HH);
    r(X + 72, Y + 5, 7, 1, C.HULL_HH);
    const beacon = Math.sin(now * 3) > 0 ? C.HATCH : C.HATCH_DD;
    r(X + 75, Y - 11, 1, 2, beacon);
    if (beacon === C.HATCH) {
      ctx.save(); ctx.globalAlpha = 0.45;
      r(X + 72, Y - 12, 7, 4, C.HATCH);
      ctx.restore();
    }

    // ============== SATELLITE DISH (mounted on spine) ==============
    const dishCx = X + 105;
    const dishCy = Y - 3;
    r(dishCx, dishCy + 5, 1, 8, C.METAL);            // stalk into spine
    r(dishCx - 1, dishCy + 11, 3, 2, C.HULL_HH);     // dish mount collar
    r(dishCx - 4, dishCy, 9, 2, C.HULL_HH);          // dish rim
    r(dishCx - 3, dishCy + 2, 7, 2, C.HULL_H);       // dish belly
    r(dishCx - 2, dishCy + 4, 5, 1, C.HULL);
    r(dishCx - 1, dishCy + 1, 3, 1, C.HULL);         // inner shadow
    r(dishCx, dishCy + 4, 1, 2, C.METAL);            // feed horn
    r(dishCx - 1, dishCy + 5, 3, 1, C.METAL);

    // ============== FLAG MAST + INSIGNIA ==============
    r(X + 140, Y - 6, 1, 13, C.METAL);               // mast runs into spine
    r(X + 139, Y + 5, 3, 1, C.HULL_HH);              // base collar
    r(X + 141, Y - 4, 12, 7, C.HULL_D);
    r(X + 141, Y - 4, 12, 1, C.HATCH);
    r(X + 141, Y + 2, 12, 1, C.HATCH);
    // Triangle insignia (airlock mark)
    r(X + 145, Y - 2, 4, 1, C.HATCH);
    r(X + 144, Y - 1, 6, 1, C.HATCH);
    r(X + 146, Y, 2, 1, C.HATCH);
    if (Math.sin(now * 2.2) > 0) {
      r(X + 153, Y - 3, 1, 5, C.HULL_D);
    }

    // ============== COMMS RADAR ==============
    r(X + 178, Y - 4, 1, 11, C.METAL);               // mast extends into spine
    r(X + 177, Y + 5, 3, 1, C.HULL_HH);              // base collar
    r(X + 175, Y - 5, 7, 1, C.METAL);
    r(X + 176, Y - 4, 5, 1, C.METAL);
    r(X + 174, Y - 4, 1, 4, C.METAL);
    r(X + 182, Y - 4, 1, 4, C.METAL);
    if (Math.sin(now * 5) > 0.5) {
      r(X + 178, Y - 6, 1, 1, C.SIGNAL);
    }

    // ============== COCKPIT (left-pointing wedge) ==============
    // Continuous silhouette: rear edge (x=45) shares Y range with hull,
    // forward tip at x=5. Drawn as horizontal scanlines.
    for (let dy = 16; dy <= 59; dy++) {
      let leftX = dy <= 37
        ? 45 - 40 * (dy - 16) / 21
        : 5 + 40 * (dy - 37) / 22;
      leftX = Math.floor(leftX);
      if (leftX < 5) leftX = 5;
      r(X + leftX, Y + dy, 45 - leftX + 1, 1, C.HULL);
    }
    // Upper-jaw highlight
    for (let dy = 16; dy <= 37; dy++) {
      const leftX = Math.floor(45 - 40 * (dy - 16) / 21);
      r(X + leftX, Y + dy, 1, 1, C.HULL_H);
      if (dy > 16) r(X + leftX + 1, Y + dy, 1, 1, C.HULL_HH);
    }
    // Lower-jaw shadow
    for (let dy = 37; dy <= 59; dy++) {
      const leftX = Math.floor(5 + 40 * (dy - 37) / 22);
      r(X + leftX, Y + dy, 1, 1, C.HULL_D);
      if (dy > 37) r(X + leftX + 1, Y + dy, 1, 1, '#0A0C12');
    }
    // Forward nose bumper / running light
    r(X + 4, Y + 36, 2, 3, C.HATCH_DD);
    r(X + 3, Y + 37, 1, 1, C.HATCH);
    if (stormIntensity > 0.4 && Math.sin(now * 8) > 0) {
      ctx.save(); ctx.globalAlpha = 0.55;
      r(X - 2, Y + 35, 8, 5, C.HATCH);
      ctx.restore();
    }

    // Cockpit windshield — wraps along inside of upper-jaw line
    for (let dy = 21; dy <= 32; dy++) {
      const baseLeft = Math.floor(45 - 40 * (dy - 16) / 21);
      const winLeft = baseLeft + 5;
      if (winLeft < 42) r(X + winLeft, Y + dy, 42 - winLeft, 1, C.GLASS_D);
    }
    for (let dy = 23; dy <= 30; dy++) {
      const baseLeft = Math.floor(45 - 40 * (dy - 16) / 21);
      const winLeft = baseLeft + 6;
      if (winLeft < 40) r(X + winLeft, Y + dy, 40 - winLeft, 1, C.GLASS);
    }
    // Glints along leading windshield edge
    for (let i = 0; i < 7; i++) {
      const dy = 24 + i;
      const baseLeft = Math.floor(45 - 40 * (dy - 16) / 21);
      r(X + baseLeft + 7, Y + dy, 1, 1, '#FFFFFF');
    }
    // Pilot silhouettes
    r(X + 28, Y + 26, 3, 4, C.HULL_D);
    r(X + 27, Y + 28, 5, 2, C.HULL_D);
    r(X + 34, Y + 27, 3, 3, C.HULL_D);
    r(X + 33, Y + 29, 5, 2, C.HULL_D);

    // ============== MAIN HULL (long rectangular body) ==============
    r(X + 45, Y + 16, 170, 44, C.HULL);
    r(X + 45, Y + 16, 170, 1, C.HULL_HH);
    r(X + 45, Y + 17, 170, 1, C.HULL_H);
    r(X + 45, Y + 58, 170, 1, C.HULL_D);
    r(X + 45, Y + 59, 170, 1, '#0A0C12');
    // Painted stripe (matches mission flag)
    r(X + 45, Y + 36, 170, 1, C.HATCH_DD);
    r(X + 45, Y + 37, 170, 2, C.HATCH);
    r(X + 45, Y + 39, 170, 1, C.HATCH_DD);
    // Plate seams with rivets
    for (let i = 0; i < 5; i++) {
      const sx = X + 78 + i * 28;
      r(sx, Y + 18, 1, 40, C.HULL_D);
      r(sx - 2, Y + 19, 1, 1, C.HULL_HH);
      r(sx + 2, Y + 19, 1, 1, C.HULL_HH);
      r(sx - 2, Y + 57, 1, 1, C.HULL_HH);
      r(sx + 2, Y + 57, 1, 1, C.HULL_HH);
    }
    // Hull number "001" stenciled near cockpit
    const sn = X + 53, ny = Y + 23;
    r(sn, ny, 3, 1, C.SERIAL); r(sn, ny + 4, 3, 1, C.SERIAL);
    r(sn, ny, 1, 5, C.SERIAL); r(sn + 2, ny, 1, 5, C.SERIAL);
    r(sn + 5, ny, 3, 1, C.SERIAL); r(sn + 5, ny + 4, 3, 1, C.SERIAL);
    r(sn + 5, ny, 1, 5, C.SERIAL); r(sn + 7, ny, 1, 5, C.SERIAL);
    r(sn + 10, ny, 1, 5, C.SERIAL);
    r(sn + 9, ny, 2, 1, C.SERIAL);
    r(sn + 9, ny + 4, 3, 1, C.SERIAL);

    // ============== PORTHOLES (4 octagonal windows; airlock takes #2 slot) ==============
    const portholeXs = [70, 130, 160, 190];
    const crewVisible = [true, false, true, false];
    for (let i = 0; i < portholeXs.length; i++) {
      const pxr = portholeXs[i];
      const pyr = 44;
      r(X + pxr + 1, Y + pyr, 8, 1, C.HULL_HH);
      r(X + pxr + 1, Y + pyr + 9, 8, 1, C.HULL_HH);
      r(X + pxr, Y + pyr + 1, 1, 8, C.HULL_HH);
      r(X + pxr + 9, Y + pyr + 1, 1, 8, C.HULL_HH);
      r(X + pxr + 1, Y + pyr + 1, 8, 8, C.HULL_D);
      r(X + pxr + 2, Y + pyr + 2, 6, 6, C.GLASS_D);
      r(X + pxr + 2, Y + pyr + 2, 6, 3, C.GLASS);
      r(X + pxr + 2, Y + pyr + 2, 6, 1, '#FFFFFF');
      r(X + pxr + 7, Y + pyr + 2, 1, 6, C.GLASS_D);
      if (crewVisible[i]) {
        r(X + pxr + 3, Y + pyr + 4, 2, 2, C.HULL_D);
        r(X + pxr + 2, Y + pyr + 6, 4, 2, C.HULL_D);
        const glow = Math.sin(now * 4 + i) > 0 ? C.SIGNAL : C.GLASS;
        r(X + pxr + 5, Y + pyr + 6, 1, 1, glow);
      }
    }

    // ============== AIRLOCK HATCH (cut into the hull side) ==============
    // Rectangular opening in the lower-front of the hull, between cockpit
    // and porthole #2. Twin doors split horizontally — upper slides up,
    // lower slides down — revealing the dark interior. Astronaut ejects
    // sideways through the LEFT edge of the opening.
    const heroP = opts.airlockProgress ?? (window.__heroProgress?.() ?? 0);
    const airlockOpen = clamp((heroP - 0.45) / 0.30, 0, 1);
    const dx0 = X + 93;
    const dyTop = Y + 22;
    const doorW = 22;
    const doorH = 34;
    const dyMid = dyTop + doorH / 2;
    const halfH = doorH / 2;
    // ─── Airlock chamber interior (revealed when outer doors slide apart) ──
    // Looking THROUGH the outer hatch into the chamber: far wall paneling,
    // inner pressure door at the back, status LEDs, hazard-striped threshold.
    r(dx0, dyTop, doorW, doorH, '#0F1218');                 // base dark metal
    r(dx0, dyTop, doorW, 1, '#04060A');                     // ceiling shadow
    r(dx0, dyTop + 1, doorW, 1, '#0A0C12');                 // ceiling sub-shadow
    r(dx0, dyTop + 2, 1, doorH - 8, C.HULL_D);              // left side wall
    r(dx0 + doorW - 1, dyTop + 2, 1, doorH - 8, C.HULL_D);  // right side wall

    // Far wall paneling — the chamber's back surface
    const fwX = dx0 + 1, fwY = dyTop + 2;
    const fwW = doorW - 2, fwH = doorH - 9;
    r(fwX, fwY, fwW, fwH, '#181C24');
    r(fwX, fwY + 9, fwW, 1, '#0A0C12');                     // panel seam
    r(fwX, fwY + 10, fwW, 1, C.HULL);

    // Pressure / status LEDs along the top of the far wall
    const eqOn = Math.sin(now * 1.7) > 0;
    r(fwX + 1, fwY + 1, 1, 1, eqOn ? C.SIGNAL : '#1F8C5B');
    r(fwX + 4, fwY + 1, 1, 1, eqOn ? C.HATCH : C.HATCH_DD);
    r(fwX + 7, fwY + 1, 1, 1, eqOn ? C.SERIAL : '#A87B2A');
    // Tiny "SEALED" stencil (block of mute pixels — texture, not glyphs)
    r(fwX + 11, fwY + 1, 7, 1, C.MUTED);

    // Inner pressure door — the SECOND door of the airlock, closed.
    const idX = fwX + 3, idY = fwY + 4;
    const idW = fwW - 6, idH = fwH - 7;
    r(idX, idY, idW, idH, C.HULL);
    r(idX, idY, idW, 1, C.HULL_HH);
    r(idX, idY + idH - 1, idW, 1, '#0A0C12');
    r(idX, idY, 1, idH, C.HULL_H);
    r(idX + idW - 1, idY, 1, idH, C.HULL_D);
    // central vertical seam (twin panels meeting in the middle)
    const seamX = idX + Math.floor(idW / 2);
    r(seamX, idY + 1, 1, idH - 2, '#0A0C12');
    // small porthole window on the inner door
    const phY = idY + 4;
    r(seamX - 2, phY, 5, 1, C.HULL_D);
    r(seamX - 2, phY + 1, 5, 1, C.GLASS_D);
    r(seamX - 1, phY + 1, 3, 1, C.GLASS);
    r(seamX, phY + 1, 1, 1, '#FFFFFF');
    r(seamX - 2, phY + 2, 5, 1, C.HULL_D);
    // door wheel handles
    r(idX + 2, idY + idH - 5, 2, 2, C.METAL);
    r(idX + idW - 4, idY + idH - 5, 2, 2, C.METAL);

    // Floor threshold with hazard chevrons
    const flY = dyTop + doorH - 6;
    r(dx0, flY, doorW, 6, C.HULL_D);
    r(dx0, flY, doorW, 1, C.HULL_HH);
    r(dx0, flY + 1, doorW, 1, C.HULL);
    ctx.save();
    ctx.beginPath();
    ctx.rect(dx0, flY + 2, doorW, 3);
    ctx.clip();
    ctx.fillStyle = C.HATCH;
    for (let i = -3; i < doorW + 3; i += 4) {
      ctx.beginPath();
      ctx.moveTo(dx0 + i, flY + 2);
      ctx.lineTo(dx0 + i + 1.5, flY + 2);
      ctx.lineTo(dx0 + i + 1.5 + 3, flY + 5);
      ctx.lineTo(dx0 + i + 3, flY + 5);
      ctx.closePath();
      ctx.fill();
    }
    ctx.restore();
    r(dx0, flY + 5, doorW, 1, '#0A0C12');

    // Emergency lighting wash — warm tint when the chamber is exposed
    if (airlockOpen > 0.15) {
      ctx.save();
      ctx.globalAlpha = (airlockOpen - 0.15) * 0.22;
      r(dx0, dyTop + 2, doorW, doorH - 8, C.HATCH);
      ctx.restore();
    }

    // Slide distance for each door half
    const slide = Math.floor(airlockOpen * (halfH + 1));

    // ─── Upper door half (slides UP into the hull above) ───
    if (slide < halfH) {
      const visH = halfH - slide;
      const drawY = dyTop - slide;
      ctx.save();
      ctx.beginPath();
      ctx.rect(dx0, dyTop, doorW, visH);
      ctx.clip();
      r(dx0, drawY, doorW, halfH, C.HULL);
      r(dx0, drawY, doorW, 1, C.HULL_HH);
      r(dx0, drawY + 1, doorW, 1, C.HULL_H);
      // hazard chevrons
      ctx.fillStyle = C.HATCH;
      for (let i = -halfH; i < doorW + halfH; i += 5) {
        ctx.beginPath();
        ctx.moveTo(dx0 + i, drawY);
        ctx.lineTo(dx0 + i + 2, drawY);
        ctx.lineTo(dx0 + i + 2 + halfH, drawY + halfH);
        ctx.lineTo(dx0 + i + halfH, drawY + halfH);
        ctx.closePath();
        ctx.fill();
      }
      // door bottom seal (the leading edge that slides)
      r(dx0, drawY + halfH - 1, doorW, 1, C.HATCH);
      r(dx0, drawY + halfH - 2, doorW, 1, C.HATCH_DD);
      // door handle indent
      r(dx0 + 9, drawY + halfH - 5, 4, 2, C.HULL_D);
      ctx.restore();
    }

    // ─── Lower door half (slides DOWN into the hull below) ───
    if (slide < halfH) {
      const visH = halfH - slide;
      const drawY = dyMid + slide;
      ctx.save();
      ctx.beginPath();
      ctx.rect(dx0, drawY, doorW, visH);
      ctx.clip();
      r(dx0, drawY, doorW, halfH, C.HULL);
      r(dx0, drawY + halfH - 1, doorW, 1, C.HULL_D);
      r(dx0, drawY + halfH - 2, doorW, 1, '#0A0C12');
      // hazard chevrons (opposite-sloping so the two halves mirror)
      ctx.fillStyle = C.HATCH;
      for (let i = -halfH; i < doorW + halfH; i += 5) {
        ctx.beginPath();
        ctx.moveTo(dx0 + i, drawY + halfH);
        ctx.lineTo(dx0 + i + 2, drawY + halfH);
        ctx.lineTo(dx0 + i + 2 + halfH, drawY);
        ctx.lineTo(dx0 + i + halfH, drawY);
        ctx.closePath();
        ctx.fill();
      }
      // leading edge (top of lower half)
      r(dx0, drawY, doorW, 1, C.HATCH);
      r(dx0, drawY + 1, doorW, 1, C.HATCH_DD);
      // handle indent
      r(dx0 + 9, drawY + 3, 4, 2, C.HULL_D);
      ctx.restore();
    }

    // Bright orange frame outline AROUND the cutout (always visible)
    r(dx0 - 1, dyTop - 1, doorW + 2, 1, C.HATCH);
    r(dx0 - 1, dyTop + doorH, doorW + 2, 1, C.HATCH);
    r(dx0 - 1, dyTop - 1, 1, doorH + 2, C.HATCH);
    r(dx0 + doorW, dyTop - 1, 1, doorH + 2, C.HATCH);
    // outer frame shadow
    r(dx0 - 2, dyTop - 2, doorW + 4, 1, C.HATCH_DD);
    r(dx0 - 2, dyTop + doorH + 1, doorW + 4, 1, C.HATCH_DD);
    r(dx0 - 2, dyTop - 1, 1, doorH + 2, C.HATCH_DD);
    r(dx0 + doorW + 1, dyTop - 1, 1, doorH + 2, C.HATCH_DD);
    // frame corner brackets (bolt-on look)
    for (const [cx, cy] of [
      [dx0 - 2, dyTop - 2],
      [dx0 + doorW - 1, dyTop - 2],
      [dx0 - 2, dyTop + doorH - 1],
      [dx0 + doorW - 1, dyTop + doorH - 1],
    ]) {
      r(cx, cy, 3, 3, C.HATCH);
      r(cx + 1, cy + 1, 1, 1, C.HATCH_DD);
    }

    // Chevron indicator strip below the frame
    for (let i = 0; i < 5; i++) {
      const phase = (Math.floor(now * 5) + i) % 5;
      const lit = airlockOpen > 0 && phase < 2;
      r(dx0 + i * 4 + 2, dyTop + doorH + 3, 3, 1, lit ? C.HATCH : C.HATCH_DD);
    }
    // Hatch label "A4" stenciled to the right of the frame (5px pixel font)
    // A
    r(dx0 + doorW + 4, dyTop + 1, 3, 1, C.SERIAL);
    r(dx0 + doorW + 4, dyTop + 2, 1, 4, C.SERIAL);
    r(dx0 + doorW + 6, dyTop + 2, 1, 4, C.SERIAL);
    r(dx0 + doorW + 4, dyTop + 3, 3, 1, C.SERIAL);
    // 4
    r(dx0 + doorW + 8, dyTop + 1, 1, 3, C.SERIAL);
    r(dx0 + doorW + 10, dyTop + 1, 1, 5, C.SERIAL);
    r(dx0 + doorW + 8, dyTop + 3, 3, 1, C.SERIAL);
    // Rotating warning beacon mounted above the frame
    const warnLit = (airlockOpen > 0 || stormIntensity > 0.4) && Math.sin(now * 6) > 0;
    r(dx0 + 9, dyTop - 5, 4, 2, warnLit ? C.HATCH : C.HATCH_DD);
    if (warnLit) {
      ctx.save(); ctx.globalAlpha = 0.5;
      r(dx0 + 4, dyTop - 7, 14, 4, C.HATCH);
      ctx.restore();
    }

    // ============== ENGINE SHOULDER (tapered transition) ==============
    for (let dx = 0; dx <= 15; dx++) {
      const topY = 16 + Math.floor(dx * 4 / 15);
      const botY = 59 - Math.floor(dx * 4 / 15);
      r(X + 215 + dx, Y + topY, 1, botY - topY + 1, C.HULL);
    }
    for (let dx = 0; dx <= 15; dx++) {
      const topY = 16 + Math.floor(dx * 4 / 15);
      r(X + 215 + dx, Y + topY, 1, 1, C.HULL_HH);
      if (dx > 0) r(X + 215 + dx, Y + topY + 1, 1, 1, C.HULL_H);
    }
    for (let dx = 0; dx <= 15; dx++) {
      const botY = 59 - Math.floor(dx * 4 / 15);
      r(X + 215 + dx, Y + botY, 1, 1, C.HULL_D);
      if (dx > 0) r(X + 215 + dx, Y + botY - 1, 1, 1, '#0A0C12');
    }
    // Structural rib at shoulder root
    r(X + 215, Y + 17, 1, 42, C.HULL_D);

    // ============== ENGINE NACELLE ==============
    r(X + 230, Y + 20, 28, 35, C.HULL_D);
    r(X + 230, Y + 20, 28, 1, C.HULL_H);
    r(X + 230, Y + 21, 28, 1, C.HULL);
    r(X + 230, Y + 54, 28, 1, '#0A0C12');
    // Grille
    for (let i = 0; i < 6; i++) {
      r(X + 233 + i * 4, Y + 24, 1, 28, C.HULL);
    }
    // Stripe continuation
    r(X + 230, Y + 36, 28, 1, C.HATCH_DD);
    r(X + 230, Y + 37, 28, 2, C.HATCH);
    r(X + 230, Y + 39, 28, 1, C.HATCH_DD);
    // Cooling fins
    for (let i = 0; i < 4; i++) {
      const fx = X + 234 + i * 6;
      r(fx, Y + 14, 2, 6, C.HULL_HH);
      r(fx, Y + 14, 2, 1, C.METAL);
      r(fx + 2, Y + 16, 1, 4, C.HULL_D);
    }
    // Reactor heat glow seeping through grille
    if (enginesOn) {
      ctx.save(); ctx.globalAlpha = 0.5;
      r(X + 244, Y + 27, 1, 22, C.HATCH_D);
      r(X + 254, Y + 27, 1, 22, C.HATCH_D);
      ctx.restore();
    }

    // ============== THRUST CONES (3 burning thrusters) ==============
    const thrusters = [
      { ty: 23, th: 6 },
      { ty: 32, th: 9 },
      { ty: 45, th: 7 },
    ];
    for (let i = 0; i < thrusters.length; i++) {
      const t2 = thrusters[i];
      const tx = X + 258;
      const ty2 = Y + t2.ty;
      r(tx, ty2, 4, t2.th, C.METAL);
      r(tx, ty2, 4, 1, C.HULL_HH);
      r(tx, ty2 + t2.th - 1, 4, 1, C.HULL_D);
      r(tx + 2, ty2 + 1, 2, t2.th - 2, '#040608');
      if (enginesOn) {
        const thrust = 0.7 + Math.sin(now * 12 + i * 1.7) * 0.3;
        ctx.save();
        ctx.globalAlpha = thrust * 0.9;
        r(tx + 4, ty2 + 1, 8, t2.th - 2, C.GLASS);
        ctx.globalAlpha = thrust * 0.6;
        r(tx + 10, ty2 + 1, 14, t2.th - 2, C.SIGNAL);
        ctx.globalAlpha = thrust * 0.3;
        r(tx + 22, ty2 + 1, 18, t2.th - 2, C.SIGNAL);
        ctx.globalAlpha = thrust * 0.12;
        r(tx + 38, ty2 + 1, 24, t2.th - 2, C.SIGNAL);
        ctx.restore();
      }
    }

    // ============== STORM SPARKS ==============
    if (stormIntensity > 0.5) {
      const sparkN = (stormIntensity - 0.5) * 6;
      for (let i = 0; i < sparkN; i++) {
        if (Math.random() < 0.15) {
          const sx2 = X + 45 + Math.random() * 215;
          const sy2 = Y + 16 + Math.random() * 43;
          r(sx2, sy2, 1, 1, C.SERIAL);
          r(sx2 - 1, sy2, 1, 1, C.HATCH);
        }
      }
    }
  }

  window.AirlockCruiser = { draw, palette: C };
})();
