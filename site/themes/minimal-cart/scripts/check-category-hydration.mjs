/**
 * Source assertion: category pages pass a safe category token into
 * ProductGrid and keep a replaceable no-JS static baseline.
 *
 * Run: node scripts/check-category-hydration.mjs
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

const categoryTpl = read('templates/category.html')
const bootstrap = read('islands/bootstrap.ts')
const grid = read('islands/ProductGrid/ProductGrid.vue')

if (!/id="shop-static"/.test(categoryTpl)) {
  errors.push('category.html must emit #shop-static so hydration can hide the baseline')
}
if (!/data-vue-island="ProductGrid"/.test(categoryTpl) || !/data-category="\{\{\.Category\}\}"/.test(categoryTpl)) {
  errors.push('category.html must emit ProductGrid with data-category token')
}
if (/data-props=/.test(categoryTpl)) {
  errors.push('category.html must not construct data-props JSON for category hydration')
}
if (!/data-category/.test(bootstrap) || !/initialCategory/.test(bootstrap) || !/SAFE_CATEGORY/.test(bootstrap)) {
  errors.push('bootstrap must map a validated data-category token to initialCategory')
}
if (!/initialCategory\?:/.test(grid) && !/initialCategory\?: string/.test(grid)) {
  errors.push('ProductGrid must accept initialCategory')
}
if (!/normalizeCategory\(props\.initialCategory\)/.test(grid)) {
  errors.push('ProductGrid must initialize the filter from initialCategory before fetch')
}
if (!/heading/.test(grid) || !/全部商品/.test(grid)) {
  errors.push('ProductGrid must label the current category truthfully')
}
if (!/hideStaticSection/.test(grid) || !/shop-static/.test(grid)) {
  errors.push('ProductGrid must hide #shop-static only after a successful load')
}

const homeTpl = read('templates/home.html')
if (!/<title>\{\{\.SiteName\}\}<\/title>/.test(homeTpl)) {
  errors.push('home.html title must be SiteName only so the brand is not duplicated')
}
if (!/質物選物/.test(homeTpl)) {
  errors.push('home.html must emit the visible 質物選物 brand in static HTML')
}

if (errors.length > 0) {
  console.error('Category hydration check FAILED:')
  for (const e of errors) console.error(`  - ${e}`)
  process.exit(1)
}

console.log('Category hydration check PASSED: static baseline + safe data-category contract.')
