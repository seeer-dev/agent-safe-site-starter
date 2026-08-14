/**
 * Source assertion: shared Dialog is an accessible modal and every
 * consumer supplies a truthful accessible name.
 *
 * Run: node scripts/check-dialog-a11y.mjs
 */
import { readFileSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const root = resolve(__dirname, '..')
const errors = []

function read(rel) {
  return readFileSync(resolve(root, rel), 'utf-8')
}

const dialog = read('shared/components/ui/Dialog.vue')
const consumers = {
  'CheckoutDialog.vue': read('islands/CheckoutDialog/CheckoutDialog.vue'),
  'AccountDialog.vue': read('islands/AccountDialog/AccountDialog.vue'),
  'FooterPageDialog.vue': read('islands/FooterPageDialog/FooterPageDialog.vue'),
  'ProductDetailDialog.vue': read('islands/ProductDetailDialog/ProductDetailDialog.vue'),
  'TrackOrderDialog.vue': read('islands/TrackOrderDialog/TrackOrderDialog.vue'),
}

if (!/role="dialog"/.test(dialog) || !/aria-modal="true"/.test(dialog)) {
  errors.push('Dialog.vue must render role=dialog and aria-modal=true')
}
if (!/aria-labelledby/.test(dialog) || !/aria-label/.test(dialog)) {
  errors.push('Dialog.vue must support aria-labelledby or aria-label')
}
if (!/tabindex="-1"/.test(dialog)) {
  errors.push('Dialog.vue must be focusable when it has no enabled controls')
}
if (!/e\.key === 'Escape'/.test(dialog) || !/e\.key !== 'Tab'/.test(dialog)) {
  errors.push('Dialog.vue must handle Escape and trap Tab/Shift+Tab')
}
if (!/restoreTrigger/.test(dialog) || !/document\.contains\(triggerEl\)/.test(dialog)) {
  errors.push('Dialog.vue must restore focus to a still-connected trigger')
}
if (!/keydownAttached/.test(dialog) || !/onUnmounted/.test(dialog)) {
  errors.push('Dialog.vue must detach listeners and restore body overflow on close/unmount')
}
if (!/aria-label="關閉"/.test(dialog)) {
  errors.push('Dialog.vue built-in close button must have an accessible name')
}
if (!/focusGeneration/.test(dialog) || !/cancelPendingFocus/.test(dialog) || !/scheduleInitialFocus/.test(dialog)) {
  errors.push('Dialog.vue must schedule initial focus with a generation token and cancelPendingFocus')
}
if (!/requestAnimationFrame/.test(dialog) || !/cancelAnimationFrame/.test(dialog)) {
  errors.push('Dialog.vue must use requestAnimationFrame (not a long timeout) for Teleport/Transition insertion')
}
if (/setTimeout\s*\(/.test(dialog)) {
  errors.push('Dialog.vue must not use setTimeout for initial focus')
}
if (!/flush:\s*['"]post['"]/.test(dialog) || !/watch\(panelRef/.test(dialog)) {
  errors.push('Dialog.vue must watch the panel ref with flush:post so nested Transition/Teleport can insert first')
}
if (!/contains\(document\.activeElement\)/.test(dialog)) {
  errors.push('Dialog.vue must confirm focus moved into the dialog and fall back to the container')
}
if (!/checkVisibility/.test(dialog) && !/display === 'none'/.test(dialog)) {
  errors.push('Dialog.vue must ignore display:none ancestors when choosing the first focusable')
}
const unmountBlock = dialog.match(/onUnmounted\(\(\) => \{[\s\S]*?\n\}\)/)
if (!unmountBlock || !/cancelPendingFocus/.test(unmountBlock[0])) {
  errors.push('Dialog.vue must cancel pending focus on unmount so a delayed rAF cannot steal focus')
}
if (!/else \{\s*cancelPendingFocus/.test(dialog)) {
  errors.push('Dialog.vue must cancel pending focus on close before restoring the trigger')
}

for (const [name, src] of Object.entries(consumers)) {
  if (!/<Dialog[\s\S]*?(aria-label|:aria-label|labelled-by|:labelled-by|title=)/.test(src)) {
    errors.push(`${name} must supply a truthful Dialog accessible name`)
  }
  const closeButtons = src.match(/<button[\s\S]*?關閉[\s\S]*?<\/button>/g) || []
  for (const btn of closeButtons) {
    if (!/aria-label=/.test(btn) && !/sr-only/.test(btn)) {
      errors.push(`${name} has a close button without an accessible name`)
      break
    }
  }
}

if (errors.length > 0) {
  console.error('Dialog a11y check FAILED:')
  for (const e of errors) console.error(`  - ${e}`)
  process.exit(1)
}

console.log('Dialog a11y check PASSED: shared primitive and consumers have names, trap, and restore.')
