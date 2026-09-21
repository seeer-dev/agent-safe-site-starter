# Curatory curtain motion walkthrough

Revision 3 verification — curtain lockup now matches the supplied reference
(`http://127.0.0.1:4187/curatory-logo-motion.html` curtain preview).

- Route: `/about/` to `/shop/` via the primary header navigation on rendered
  `dist/` output (static serve on :8090).
- Curtain markup: `#page-curtain` contains `img.curtain-logo` +
  `p.curtain-caption`; the logo is the single baked lockup asset
  `curatory-bloom-curtain-motion.svg` (viewBox `260 85 730 1070` — mark plus
  質選所/CURATORY wordmark, cream `#EEE7DB` with terracotta `#CC9782` petal
  accent), sized `min(47vw,280px)` / `61vw` under 760px, `max-height:64dvh`.
- Cover: adding `curtain-cover` plays `curtain-cover` 0.52s
  cubic-bezier(0.22,1,0.36,1); screenshot at full cover shows the lockup and
  brand line 慢一點，遇見好物。 matching the reference composition.
- Mark motion: two `.curtain-logo` screenshots 800ms apart differ — the
  embedded wing-flap/flower-sway keyframes run inside the `<img>` document.
- Reveal: after real navigation the new page carries `curtain-reveal`
  (0.68s cubic-bezier(0.65,0,0.25,1)) which clears itself on `animationend`;
  `/shop/` settled with the curtain hidden and the static header mark intact.
- Shared chrome check: header and footer continue to use only
  `curatory-bloom-mark.svg`.
- Reduced motion: navigation bypasses the extended curtain; the lockup SVG's
  internal `prefers-reduced-motion` rule freezes wing/flower motion, and the
  curtain falls back to the 0.15s `curtain-fade` (reversed on reveal).
