(function () {
  function paint(canvas) {
    const cruiser = window.AirlockCruiser;
    if (!cruiser) return;

    const rect = canvas.getBoundingClientRect();
    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    const cssW = Math.max(1, Math.round(rect.width || canvas.width));
    const cssH = Math.max(1, Math.round(rect.height || canvas.height));
    const pixelW = Math.round(cssW * dpr);
    const pixelH = Math.round(cssH * dpr);
    if (canvas.width !== pixelW || canvas.height !== pixelH) {
      canvas.width = pixelW;
      canvas.height = pixelH;
    }

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const w = cssW;
    const h = cssH;
    ctx.imageSmoothingEnabled = false;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, w, h);

    ctx.save();
    ctx.translate(w / 2, h / 2 + 36);
    ctx.rotate(Math.PI / 2);
    ctx.scale(1.48, 1.48);
    cruiser.draw(ctx, {
      x: -130,
      y: -42,
      stormIntensity: 0,
      enginesOn: false,
      motion: false,
      airlockProgress: 0,
    });
    ctx.restore();
  }

  function renderAll() {
    document.querySelectorAll('canvas[data-manifest-cruiser]').forEach(paint);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', renderAll);
  } else {
    renderAll();
  }

  window.addEventListener('resize', renderAll);
})();
