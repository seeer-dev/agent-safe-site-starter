# Evidence

## Delivery status

Revision 3 is accepted. The curtain was realigned to the supplied motion
reference (`curatory-logo-motion.html`): the mark-only asset plus separately
typeset wordmark was replaced by the baked cream-and-terracotta lockup
(mark + 質選所 + CURATORY in one animated SVG) with the reference brand line.

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | Shared header, footer, loader, checkout, and rendered fallback keep using the static flower-and-butterfly mark. Receipts: receipts/curatory-curtain-motion-walkthrough.md. |
| REQ-002 | passed | An eligible header navigation activates the curtain-only animated lockup matching the supplied motion reference — cream-and-terracotta mark with independently folding wings and swaying flower, baked 質選所/CURATORY wordmark, and the reference brand line. Receipts: receipts/curatory-curtain-motion-walkthrough.md. |
| AC-001 | passed | Shared chrome and its rendered fallback use only the static mark asset. Receipts: receipts/curatory-curtain-motion-walkthrough.md. |
| AC-002 | passed | A real /about/ to /shop/ header navigation set curtain-cover, lifted on arrival via curtain-reveal, and displayed the reference lockup (0.52s cover, 0.68s reveal, live wing/flower keyframes verified across frames) with brand line 慢一點，遇見好物。 Receipts: receipts/curatory-curtain-motion-walkthrough.md. |
| AC-003 | passed | Reduced motion still bypasses the extended transition; the lockup asset freezes its internal wing/flower animation via prefers-reduced-motion and the curtain falls back to curtain-fade. Receipts: receipts/curatory-curtain-motion-walkthrough.md. |
