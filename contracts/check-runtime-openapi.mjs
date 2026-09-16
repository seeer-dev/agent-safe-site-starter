import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const bootstrap = readFileSync(resolve(root, 'server/internal/bootstrap/app.go'), 'utf8')
const openapi = readFileSync(resolve(root, 'contracts/openapi.yaml'), 'utf8')
const errors = []

const httpMethods = new Set(['get', 'post', 'put', 'patch', 'delete', 'head', 'options'])

function runtimeOperations(source) {
  const operations = new Map()
  const re = /mux\.HandleFunc\("([A-Z]+) ([^"]+)"/g
  for (const match of source.matchAll(re)) {
    const method = match[1].toLowerCase()
    const path = match[2]
    const key = `${method.toUpperCase()} ${path}`
    if (operations.has(key)) {
      errors.push(`runtime registers duplicate operation ${key}`)
    }
    operations.set(key, { method, path })
  }
  return operations
}

function openapiOperations(source) {
  const lines = source.split('\n')
  const operations = new Map()
  let inPaths = false
  let currentPath = ''

  for (const line of lines) {
    if (line === 'paths:') {
      inPaths = true
      currentPath = ''
      continue
    }
    if (!inPaths) continue
    if (line === 'components:') break

    const pathMatch = line.match(/^  (\/[^:]+):\s*$/)
    if (pathMatch) {
      currentPath = pathMatch[1]
      continue
    }
    const methodMatch = line.match(/^    (get|post|put|patch|delete|head|options):\s*$/)
    if (currentPath && methodMatch && httpMethods.has(methodMatch[1])) {
      const method = methodMatch[1]
      const key = `${method.toUpperCase()} ${currentPath}`
      if (operations.has(key)) {
        errors.push(`OpenAPI declares duplicate operation ${key}`)
      }
      operations.set(key, { method, path: currentPath })
    }
  }
  return operations
}

function operationBlock(path, method) {
  const lines = openapi.split('\n')
  const pathLine = `  ${path}:`
  const methodLine = `    ${method.toLowerCase()}:`
  const pathIndex = lines.findIndex((line) => line === pathLine)
  if (pathIndex < 0) return ''

  let methodIndex = -1
  for (let i = pathIndex + 1; i < lines.length; i++) {
    if (/^  \/[^:]+:\s*$/.test(lines[i]) || lines[i] === 'components:') break
    if (lines[i] === methodLine) {
      methodIndex = i
      break
    }
  }
  if (methodIndex < 0) return ''

  let end = lines.length
  for (let i = methodIndex + 1; i < lines.length; i++) {
    if (/^    (get|post|put|patch|delete|head|options):\s*$/.test(lines[i]) || /^  \/[^:]+:\s*$/.test(lines[i]) || lines[i] === 'components:') {
      end = i
      break
    }
  }
  return lines.slice(methodIndex, end).join('\n')
}

function componentSchemaBlock(name) {
  const lines = openapi.split('\n')
  const schemaLine = `    ${name}:`
  const schemaIndex = lines.findIndex((line) => line === schemaLine)
  if (schemaIndex < 0) return ''

  let end = lines.length
  for (let i = schemaIndex + 1; i < lines.length; i++) {
    if (/^    [A-Za-z0-9_]+:\s*$/.test(lines[i])) {
      end = i
      break
    }
  }
  return lines.slice(schemaIndex, end).join('\n')
}

function requireOperationContains(path, method, needles) {
  const block = operationBlock(path, method)
  if (!block) {
    errors.push(`cannot inspect missing OpenAPI operation ${method.toUpperCase()} ${path}`)
    return
  }
  for (const needle of needles) {
    if (!block.includes(needle)) {
      errors.push(`${method.toUpperCase()} ${path} contract is missing ${JSON.stringify(needle)}`)
    }
  }
}

function requireComponentContains(name, needles) {
  const block = componentSchemaBlock(name)
  if (!block) {
    errors.push(`cannot inspect missing OpenAPI component ${name}`)
    return
  }
  for (const needle of needles) {
    if (!block.includes(needle)) {
      errors.push(`OpenAPI component ${name} is missing ${JSON.stringify(needle)}`)
    }
  }
}

const runtime = runtimeOperations(bootstrap)
const contract = openapiOperations(openapi)

for (const key of runtime.keys()) {
  if (!contract.has(key)) errors.push(`runtime operation missing from OpenAPI: ${key}`)
}
for (const key of contract.keys()) {
  if (!runtime.has(key)) errors.push(`OpenAPI operation is not registered by runtime: ${key}`)
}

// Representative observable-contract guards. Route parity above is exhaustive;
// these guards make status/schema drift at the known high-risk boundaries fail
// mechanically without pretending this lightweight checker is a YAML schema
// validator or generated-client toolchain.
requireOperationContains('/api/admin/products', 'get', [
  "items: {$ref: '#/components/schemas/AdminProductResponse'}",
])
requireOperationContains('/api/admin/products', 'post', [
  "'201':",
  "$ref: '#/components/schemas/AdminProductResponse'",
])
for (const method of ['get', 'put']) {
  requireOperationContains('/api/admin/products/{id}', method, [
    "$ref: '#/components/schemas/AdminProductResponse'",
  ])
}
requireOperationContains('/api/admin/products/{id}', 'delete', [
  "'200':",
  'required: [id]',
])
requireOperationContains('/api/admin/products/{id}/status', 'patch', [
  "'200':",
  "$ref: '#/components/schemas/AdminProductResponse'",
])
requireOperationContains('/api/admin/products/bulk', 'post', [
  "'200':",
  'required: [updated]',
])

requireOperationContains('/api/media/presign', 'post', [
  'required: [filename, content_type, purpose]',
])

requireOperationContains('/api/admin/staff', 'get', [
  'required: [members]',
  "items: {$ref: '#/components/schemas/StaffMember'}",
])
requireOperationContains('/api/admin/staff', 'post', ["'201':"])
requireOperationContains('/api/admin/staff/{id}', 'delete', [
  "'200':",
  'required: [status]',
])

requireOperationContains('/api/orders/{id}/payments/ecpay', 'post', [
  'X-Order-Access-Token',
  "'200':",
  "$ref: '#/components/schemas/ECPayLaunchForm'",
])
requireOperationContains('/api/payments/ecpay/return', 'post', [
  'application/x-www-form-urlencoded',
  "'200':",
  'text/plain:',
  'example: 1|OK',
])
requireOperationContains('/api/payments/ecpay/browser-return', 'post', [
  'application/x-www-form-urlencoded',
  "'303':",
  'Location:',
])
requireComponentContains('ECPayCallbackForm', [
  'required: [CheckMacValue, MerchantID, MerchantTradeNo, TradeAmt, RtnCode, TradeNo]',
  'TradeAmt: {type: string}',
  "SimulatePaid: {type: string, enum: ['0', '1']}",
])

if (errors.length > 0) {
  console.error('Runtime/OpenAPI parity check FAILED:')
  for (const error of errors) console.error(`  - ${error}`)
  process.exit(1)
}

console.log(`Runtime/OpenAPI parity check PASSED: ${runtime.size} registered operations match the OpenAPI surface.`)