// Scroll-triggered reveals for content sections
(function () {
  const io = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (e.isIntersecting) {
        e.target.classList.add('is-visible');
        io.unobserve(e.target);
      }
    }
  }, { threshold: 0.15, rootMargin: '0px 0px -10% 0px' });

  document.querySelectorAll('[data-reveal]').forEach(el => io.observe(el));

  // Stagger children
  document.querySelectorAll('[data-stagger]').forEach(parent => {
    const kids = parent.querySelectorAll('[data-stagger-child]');
    kids.forEach((k, i) => {
      k.style.transitionDelay = (i * 60) + 'ms';
    });
  });

  // Live countdown
  (function(){
    const d = document.querySelector('[data-cd="d"]');
    const h = document.querySelector('[data-cd="h"]');
    const m = document.querySelector('[data-cd="m"]');
    const s = document.querySelector('[data-cd="s"]');
    if (!s) return;
    let total = 2*86400 + 6*3600 + 42*60 + 11;
    function pad(n){ return String(n).padStart(2,'0'); }
    function tick(){
      if (total <= 0) total = 7*86400;
      const dd = Math.floor(total / 86400);
      const hh = Math.floor((total % 86400) / 3600);
      const mm = Math.floor((total % 3600) / 60);
      const ss = total % 60;
      d.textContent = pad(dd);
      h.textContent = pad(hh);
      m.textContent = pad(mm);
      s.textContent = pad(ss);
      total -= 1;
    }
    tick();
    setInterval(tick, 1000);
  })();

  // Skip-cinematic shortcut: pressing Space or scrolling rapidly jumps past
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' || e.key === 'Enter') {
      const after = document.getElementById('after-cinematic');
      if (after) after.scrollIntoView({ behavior: 'smooth' });
    }
  });

  // Scroll hint - hide after first scroll
  let hidHint = false;
  window.addEventListener('scroll', () => {
    if (hidHint) return;
    if (window.scrollY > 80) {
      document.body.classList.add('scrolled');
      hidHint = true;
    }
  }, { passive: true });
})();
