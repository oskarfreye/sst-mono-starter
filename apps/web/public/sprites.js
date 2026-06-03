// Small pixel-art sprites for content sections.
// Renders into <canvas data-sprite="name"> and <canvas data-avatar="N"> elements.
(function () {
  const PAL = {
    BG:      '#0B0E13',
    TEXT:    '#E7EAF0',
    MUTED:   '#8A93A4',
    DIM:     '#5A6371',
    DARK:    '#0E1218',
    HATCH:   '#FF7A3D',
    HATCH_D: '#C9461A',
    SIGNAL:  '#5BFFA8',
    SIG_D:   '#1F8C5B',
    ICE:     '#9CD2FF',
    SERIAL:  '#FFD25C',
    PURPLE:  '#A88BE0',
    BLUE:    '#5B92FF',
  };

  // ---------- generic sprite renderer ----------
  // Each sprite is an array of strings, characters map to palette keys
  function paint(canvas, lines, key) {
    const ctx = canvas.getContext('2d');
    ctx.imageSmoothingEnabled = false;
    const w = canvas.width, h = canvas.height;
    ctx.clearRect(0, 0, w, h);
    const rows = lines.length;
    const cols = lines[0].length;
    const cell = Math.min(w / cols, h / rows);
    const offX = (w - cell * cols) / 2;
    const offY = (h - cell * rows) / 2;
    for (let y = 0; y < rows; y++) {
      for (let x = 0; x < lines[y].length; x++) {
        const c = lines[y][x];
        if (c === '.' || c === ' ') continue;
        ctx.fillStyle = key[c] || PAL.TEXT;
        ctx.fillRect((offX + x*cell)|0, (offY + y*cell)|0, Math.ceil(cell), Math.ceil(cell));
      }
    }
  }

  // ============================================================
  // STEP SPRITES (32×32 base, drawn at 16×16 grid scaled 2x)
  // ============================================================

  // STEP 1 — Board (ship docked at platform with arrow)
  const STEP_BOARD = [
    '................',
    '......WWWWW.....',
    '.....WTTTTTW....',
    '....WTTTOTTTW...',
    '....WTHTHTTTW...',
    '....WTTTTTTTW...',
    '.....DTTTDD.....',
    '......DDD.......',
    '................',
    '...OOOOOOOOOO...',
    '...OdddddddOO...',
    '...OOOOOOOOOO...',
    '........O.......',
    '........O.......',
    '......OOOOO.....',
    '.......OOO......',
  ];
  const KEY_BOARD = { W:PAL.TEXT, T:PAL.MUTED, H:PAL.ICE, D:PAL.DIM, O:PAL.HATCH, d:PAL.HATCH_D };

  // STEP 2 — Declare (clipboard / mission card)
  const STEP_DECLARE = [
    '................',
    '......TTTT......',
    '.....TWWWWT.....',
    '.WWWWWWWWWWWW...',
    '.WHHHHHHHHHHW...',
    '.WHWWWWWWWWHW...',
    '.WHWooooooWHW...',
    '.WHWWWWWWWWHW...',
    '.WHWoooooWWHW...',
    '.WHWWWWWWWWHW...',
    '.WHWooooWWWHW...',
    '.WHWWWWWWWWHW...',
    '.WHHHHHHHHHHW...',
    '.WWWWWWWWWWWW...',
    '................',
    '................',
  ];
  const KEY_DECLARE = { W:PAL.TEXT, H:PAL.DIM, T:PAL.MUTED, o:PAL.HATCH };

  // STEP 3 — Ship (paper airplane / rocket launching)
  const STEP_SHIP = [
    '................',
    '.......W........',
    '......WTW.......',
    '......WTW.......',
    '.....WTTTW......',
    '.....WTITW......',
    '.....WTTTW......',
    '....WTTTTTW.....',
    '....WTOOOTW.....',
    '...WTTOOOTTW....',
    '...DTDDDDDTD....',
    '....OOOOOOO.....',
    '.....OOOOO......',
    '......dOd.......',
    '......d.d.......',
    '................',
  ];
  const KEY_SHIP = { W:PAL.TEXT, T:PAL.MUTED, I:PAL.ICE, D:PAL.DIM, O:PAL.HATCH, d:PAL.HATCH_D };

  // STEP 4 — Airlock (hatch opening / warning)
  const STEP_AIRLOCK = [
    '................',
    '.OOOOOOOOOOOOOO.',
    '.OdddddddddddO..',
    '.OdLLLLLLLLLdO..',
    '.OdLwwww.wwwLdO.',
    '.OdLw..w.w..LdO.',
    '.OdLw.ww.w.wLdO.',
    '.OdLw..w.w..LdO.',
    '.OdLw.ww.w.wLdO.',
    '.OdLwwww.wwwLdO.',
    '.OdLLLLLLLLLdO..',
    '.OdddddddddddO..',
    '.OOOOOOOOOOOOOO.',
    '..oo.oo.oo.oo...',
    '..oo.oo.oo.oo...',
    '................',
  ];
  const KEY_AIRLOCK = { O:PAL.HATCH_D, d:PAL.HATCH, L:PAL.DIM, w:PAL.DARK, o:PAL.HATCH };

  // ============================================================
  // PROTOCOL SPRITES (18×18)
  // ============================================================

  // P1 — Public miss log (scroll with red mark)
  const P_LOG = [
    '..............',
    '.WWWWWWWWWWWW.',
    '.WMMMMMMMMMMW.',
    '.WMMMOMMMMMMW.',
    '.WMMMOOMMMMMW.',
    '.WMMMOMMMMMMW.',
    '.WMMMMMMMMMMW.',
    '.WMMMMMMMMMMW.',
    '.WMMMMMOMMMMW.',
    '.WMMMMMMMMMMW.',
    '.WMMMMMMMMMMW.',
    '.WMMOMMMMMMMW.',
    '.WMMMMMMMMMMW.',
    '.WWWWWWWWWWWW.',
  ];
  const P_LOG_K = { W:PAL.DIM, M:PAL.MUTED, O:PAL.HATCH };

  // P2 — Streak reset (broken bars)
  const P_RESET = [
    '..............',
    '..G..G..G..G..',
    '..G..G..G..G..',
    '..G..G..G..G..',
    '..G..G..G..G..',
    '..............',
    '..O..O..O..O..',
    '..O..O..O..O..',
    '..............',
    '....OO........',
    '..OOOOOO......',
    '....OO........',
    '..............',
    '..............',
  ];
  const P_RESET_K = { G:PAL.DIM, O:PAL.HATCH };

  // P3 — Donation (coin)
  const P_COIN = [
    '..............',
    '....SSSSS.....',
    '...SssssyS....',
    '..SssYYssyS...',
    '..SsYssYsyS...',
    '..SsYssYsyS...',
    '..SsYssYsyS...',
    '..SsYYYYsyS...',
    '..SsssYssyS...',
    '..SsssYssyS...',
    '..SssssyysS...',
    '...SyyyyysS...',
    '....SSSSS.....',
    '..............',
  ];
  const P_COIN_K = { S:PAL.HATCH_D, s:PAL.SERIAL, Y:PAL.HATCH, y:PAL.HATCH_D };

  // P4 — Crew alert (megaphone / signal burst)
  const P_ALERT = [
    '..............',
    '..............',
    '.....O........',
    '....OO.....O..',
    '...OO..O....O.',
    '..OO..OOOOO...',
    '.OO..OOWWWO.O.',
    '.OO..OOWWWO...',
    '.OO..OOOOO.O..',
    '...OO..O...O..',
    '....OO.....O..',
    '.....O........',
    '..............',
    '..............',
  ];
  const P_ALERT_K = { O:PAL.HATCH, W:PAL.DARK };

  // P5 — Re-entry (circular arrow)
  const P_REENTRY = [
    '..............',
    '....OOOO......',
    '...O....OO....',
    '..O..OOOOOO...',
    '..O.OOOO..O...',
    '.O.OO.....OO..',
    '.O.O..........',
    '.O.O..........',
    '.O.OO.....OO..',
    '..O.OOOO..O...',
    '..O..OOOOOO...',
    '...O....OO....',
    '....OOOO......',
    '..............',
  ];
  const P_REENTRY_K = { O:PAL.HATCH };

  // ============================================================
  // USE CASE AVATARS (16×16)
  // ============================================================

  function role(headColor, suitColor, badgeColor) {
    return [
      '................',
      '......TTTT......',
      '.....TWWWWT.....',
      '....TWHHHHWT....',
      '....TWHvvHWT....',  // visor or face
      '....TWHHHHWT....',
      '.....TWWWWT.....',
      '......TTTT......',
      '.....SSSSSS.....',
      '....SSSSSSSS....',
      '...SSBBBBBBSS...',
      '...SSBBssBBSS...',
      '....SSSSSSSS....',
      '....SS....SS....',
      '....SS....SS....',
      '...DDD....DDD...',
    ].map(r => r
      .replace(/H/g, headColor)
      .replace(/S/g, suitColor)
      .replace(/B/g, badgeColor)
    );
  }

  // Generic pixel character for case cards (icon-style, not realistic)
  function caseIcon(suit, accent) {
    return [
      '................',
      '......XXXX......',
      '.....XSSSSX.....',
      '.....XSAASX.....',
      '.....XSSSSX.....',
      '......XSSX......',
      '.....XSSSSX.....',
      '....XSSAASSX....',
      '...XSSSAASSSX...',
      '...XSSSAASSSX...',
      '....XSSSSSSSX...',
      '.....XS..SX.....',
      '.....XS..SX.....',
      '.....XS..SX.....',
      '....XX....XX....',
      '................',
    ];
  }

  // ============================================================
  // 16×16 PIXEL AVATARS for crew log
  // Six unique faces with different palettes
  // ============================================================

  function avatar(skin, hair, accent, eye) {
    return {
      lines: [
        '................',
        '....HHHHHHHH....',
        '...HHHHHHHHHH...',
        '..HHsssssssHH...',
        '..HsssssssssH...',
        '..ss.eEee.ess...',
        '..sss.....sss...',
        '...sssAAAsss....',
        '....ssssssss....',
        '...aaaaaaaaaa...',
        '..aaaAAAAAAaaa..',
        '..aAAAAAAAAAaa..',
        '..aAAAAAAAAAaa..',
        '..aaaaAaAaaaaa..',
        '...aa.....aa....',
        '...aa.....aa....',
      ],
      key: { H: hair, s: skin, e: PAL.DARK, E: eye, A: accent, a: accent }
    };
  }

  const AVATARS = [
    avatar('#E8C9A4', '#3A2515', PAL.SIGNAL, PAL.DARK),
    avatar('#D4A483', '#1A1A1A', PAL.ICE,    PAL.DARK),
    avatar('#F0D8B8', '#C8A858', PAL.SERIAL, PAL.DARK),
    avatar('#D8B090', '#5A3520', PAL.HATCH,  PAL.DARK),
    avatar('#C49070', '#2A1A0F', PAL.PURPLE, PAL.DARK),
    avatar('#E8C9A4', '#806040', PAL.SIGNAL, PAL.DARK),
  ];

  // ============================================================
  // BANNER (final CTA) — pixel-art "AIRLOCK" word mark
  // ============================================================
  // 120×30 canvas. We'll draw a stylized airlock-door pixel pattern
  // with light streaming through, as a decorative banner.
  function drawBanner(canvas) {
    const ctx = canvas.getContext('2d');
    ctx.imageSmoothingEnabled = false;
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    const W = canvas.width, H = canvas.height;
    // Frame
    ctx.fillStyle = PAL.HATCH_D;
    ctx.fillRect(0, 0, W, 2);
    ctx.fillRect(0, H-2, W, 2);
    ctx.fillRect(0, 0, 2, H);
    ctx.fillRect(W-2, 0, 2, H);
    // Warning chevrons top/bottom
    for (let i = 2; i < W - 2; i += 6) {
      ctx.fillStyle = (i / 6 | 0) % 2 ? PAL.HATCH : PAL.DARK;
      ctx.fillRect(i, 0, 4, 2);
      ctx.fillRect(i, H-2, 4, 2);
    }
    // Open hatch (light streams from center)
    const cx = W/2;
    const gap = 18;
    // Space behind
    ctx.fillStyle = '#04060A';
    ctx.fillRect(cx - gap, 2, gap*2, H - 4);
    // light streams (radial-ish)
    for (let i = 0; i < 12; i++) {
      const ang = (i / 12) * Math.PI - Math.PI/2;
      const x1 = cx, y1 = H/2;
      const x2 = cx + Math.cos(ang) * 60;
      const y2 = H/2 + Math.sin(ang) * 30;
      ctx.strokeStyle = i % 2 ? 'rgba(255,210,92,0.3)' : 'rgba(255,122,61,0.4)';
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(x1|0, y1|0);
      ctx.lineTo(x2|0, y2|0);
      ctx.stroke();
    }
    // doors (left & right halves)
    ctx.fillStyle = '#1A2030';
    ctx.fillRect(2, 2, cx - gap - 2, H - 4);
    ctx.fillRect(cx + gap, 2, W - cx - gap - 2, H - 4);
    // diagonal stripes
    ctx.fillStyle = PAL.HATCH;
    for (let i = -H; i < cx; i += 6) {
      ctx.fillRect(i + 2, 2, 2, H - 4);
    }
    // restore door background over stripes for contrast
    ctx.fillStyle = 'rgba(26,32,48,0.6)';
    ctx.fillRect(2, 2, cx - gap - 2, H - 4);
    ctx.fillRect(cx + gap, 2, W - cx - gap - 2, H - 4);
    // door edges glow
    ctx.fillStyle = PAL.HATCH;
    ctx.fillRect(cx - gap - 1, 2, 1, H - 4);
    ctx.fillRect(cx + gap, 2, 1, H - 4);
    // small stars in the void
    ctx.fillStyle = PAL.TEXT;
    ctx.fillRect(cx - 6, 8, 1, 1);
    ctx.fillRect(cx + 4, 14, 1, 1);
    ctx.fillRect(cx - 2, 20, 1, 1);
    ctx.fillRect(cx + 8, 6, 1, 1);
    // tiny rocket silhouette in the gap
    ctx.fillStyle = PAL.TEXT;
    ctx.fillRect(cx - 1, H/2 - 5, 2, 5);
    ctx.fillRect(cx - 2, H/2 - 3, 4, 2);
    ctx.fillStyle = PAL.HATCH;
    ctx.fillRect(cx - 1, H/2, 2, 2);
    ctx.fillStyle = PAL.SERIAL;
    ctx.fillRect(cx - 1, H/2 + 2, 2, 2);
  }

  // ============================================================
  // BIND
  // ============================================================
  const SPRITE_MAP = {
    'board':     [STEP_BOARD, KEY_BOARD],
    'declare':   [STEP_DECLARE, KEY_DECLARE],
    'ship':      [STEP_SHIP, KEY_SHIP],
    'airlock':   [STEP_AIRLOCK, KEY_AIRLOCK],
    'proto-log': [P_LOG, P_LOG_K],
    'proto-reset': [P_RESET, P_RESET_K],
    'proto-coin':  [P_COIN, P_COIN_K],
    'proto-alert': [P_ALERT, P_ALERT_K],
    'proto-reentry':[P_REENTRY, P_REENTRY_K],
  };

  // For use-case sprites we'll use small distinctive icons (briefcase, terminal, pen, phone, code)
  const ICON_BRIEFCASE = [
    '................',
    '......OOOO......',
    '.....OOOOOO.....',
    '....OOOOOOOO....',
    '...WWWWWWWWWW...',
    '...WHHHHHHHHW...',
    '...WHooooooHW...',
    '...WHooooooHW...',
    '...WHooHHooHW...',
    '...WHHHHHHHHW...',
    '...WHooooooHW...',
    '...WHooooooHW...',
    '...WHHHHHHHHW...',
    '...WWWWWWWWWW...',
    '................',
    '................',
  ];
  const ICON_BRIEFCASE_K = { O:PAL.HATCH_D, W:PAL.HATCH, H:PAL.SERIAL, o:PAL.HATCH_D };

  const ICON_TERMINAL = [
    '................',
    '..WWWWWWWWWWWW..',
    '..WoooooooooHW..',
    '..WoOO.......W..',
    '..Wo.OO......W..',
    '..Wo..OO.....W..',
    '..Wo.OO......W..',
    '..WoOO..GGGG.W..',
    '..Wo.........W..',
    '..Wo.........W..',
    '..WHHHHHHHHHHW..',
    '..WWWWWWWWWWWW..',
    '......WWWW......',
    '....WWWWWWWW....',
    '...WWWWWWWWWW...',
    '................',
  ];
  const ICON_TERMINAL_K = { W:PAL.DIM, o:PAL.DARK, O:PAL.SIGNAL, G:PAL.SIGNAL, H:PAL.MUTED };

  const ICON_PEN = [
    '................',
    '............OO..',
    '...........OWO..',
    '..........OWO...',
    '.........OWO....',
    '........OWO.....',
    '.......OWO......',
    '......OWO.......',
    '.....OWO........',
    '....OWO.........',
    '...OWO..........',
    '..OWO...........',
    '.OWO............',
    '.WO.............',
    '.O..............',
    '................',
  ];
  const ICON_PEN_K = { O:PAL.HATCH_D, W:PAL.HATCH };

  const ICON_PHONE = [
    '................',
    '....WWWWWW......',
    '...WoooooooW....',
    '..WooHHHHHHooW..',
    '..WoHooooooHoW..',
    '..WoHoIIIIoHoW..',
    '..WoHoooooohoW..',
    '..WoHooooooHoW..',
    '..WoHoIIIIoHoW..',
    '..WoHooooooHoW..',
    '..WoHHHHHHHHoW..',
    '..WoooooHoooowW.',
    '...WoooHHHooW...',
    '....WWWWWWWW....',
    '................',
    '................',
  ];
  const ICON_PHONE_K = { W:PAL.DIM, o:PAL.MUTED, H:PAL.DARK, I:PAL.SIGNAL, h:PAL.HATCH };

  const ICON_CODE = [
    '................',
    '..WWWWWWWWWWWW..',
    '..WgggggggggW..',
    '..WgOOOOOOOOgW..',
    '..WgoooSooogW..',
    '..WgoSSooSSogW..',
    '..WgSooSSooSgW..',
    '..WgoSSooSSogW..',
    '..WgooSooSoogW..',
    '..WgooooooogW..',
    '..WggggggggggW..',
    '..WgPgggggggW..',
    '..WWWWWWWWWWWW..',
    '...WWWWWWWWW....',
    '................',
    '................',
  ];
  const ICON_CODE_K = { W:PAL.DIM, g:PAL.DARK, O:PAL.SIG_D, o:PAL.SIGNAL, S:PAL.HATCH, P:PAL.HATCH };

  Object.assign(SPRITE_MAP, {
    'case-founder':  [ICON_BRIEFCASE, ICON_BRIEFCASE_K],
    'case-dev':      [ICON_TERMINAL,  ICON_TERMINAL_K],
    'case-creator':  [ICON_PEN,       ICON_PEN_K],
    'case-free':     [ICON_PHONE,     ICON_PHONE_K],
    'case-indie':    [ICON_CODE,      ICON_CODE_K],
  });

  // Render all data-sprite canvases
  document.querySelectorAll('canvas[data-sprite]').forEach(c => {
    const name = c.getAttribute('data-sprite');
    if (name === 'banner') {
      drawBanner(c);
    } else if (SPRITE_MAP[name]) {
      const [lines, key] = SPRITE_MAP[name];
      paint(c, lines, key);
    }
  });

  window.__airlockPaintAvatars = function(root) {
    const scope = root || document;
    scope.querySelectorAll('canvas[data-avatar]').forEach(c => {
      const idx = parseInt(c.getAttribute('data-avatar'), 10) || 0;
      const a = AVATARS[idx % AVATARS.length];
      paint(c, a.lines, a.key);
    });
  };

  // Render avatars
  window.__airlockPaintAvatars(document);
})();
