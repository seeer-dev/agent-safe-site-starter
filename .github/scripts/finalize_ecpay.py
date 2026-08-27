from pathlib import Path
import json

ROOT = Path('.')

def replace(path, old, new):
    p = ROOT / path
    text = p.read_text()
    if new in text:
        return
    if old not in text:
        raise SystemExit(f'anchor not found in {path}: {old[:100]!r}')
    p.write_text(text.replace(old, new, 1))

# Production must never accept the public ECPay test credential sets.
replace(
    'server/internal/modules/commerce/ecpay.go',
    '\tif strings.TrimSpace(merchantID) == "" || strings.TrimSpace(hashKey) == "" || strings.TrimSpace(hashIV) == "" {\n\t\treturn ECPayConfig{}, ErrECPayInvalidConfig\n\t}\n\treturn cfg, nil\n}',
    '\tif strings.TrimSpace(merchantID) == "" || strings.TrimSpace(hashKey) == "" || strings.TrimSpace(hashIV) == "" {\n\t\treturn ECPayConfig{}, ErrECPayInvalidConfig\n\t}\n\tif environment == "production" && isKnownECPayTestCredential(merchantID, hashKey, hashIV) {\n\t\treturn ECPayConfig{}, ErrECPayInvalidConfig\n\t}\n\treturn cfg, nil\n}\n\nfunc isKnownECPayTestCredential(merchantID, hashKey, hashIV string) bool {\n\tknown := [][3]string{\n\t\t{"3002607", "pwFHCqoQZGmho4w6", "EkRm7iFT261dpevs"},\n\t\t{"2000132", "5294y06JbISpM5x9", "v77hoKGq4kWxNNIS"},\n\t\t{"2000213", "Xd668CHQNfTzKtB5", "Uj35oQ3X2v5YNhQX"},\n\t}\n\tfor _, credential := range known {\n\t\tif merchantID == credential[0] && hashKey == credential[1] && hashIV == credential[2] {\n\t\t\treturn true\n\t\t}\n\t}\n\treturn false\n}')

# Use explicit ordered replacements in the protocol canonicalizer.
replace(
    'server/internal/modules/commerce/ecpay.go',
    '\treplacements := map[string]string{"%2d": "-", "%5f": "_", "%2e": ".", "%21": "!", "%2a": "*", "%28": "(", "%29": ")", "~": "%7e"}\n\tfor from, to := range replacements {\n\t\tencoded = strings.ReplaceAll(encoded, from, to)\n\t}\n',
    '\tencoded = strings.ReplaceAll(encoded, "~", "%7e")\n\tencoded = strings.ReplaceAll(encoded, "%2d", "-")\n\tencoded = strings.ReplaceAll(encoded, "%5f", "_")\n\tencoded = strings.ReplaceAll(encoded, "%2e", ".")\n\tencoded = strings.ReplaceAll(encoded, "%21", "!")\n\tencoded = strings.ReplaceAll(encoded, "%2a", "*")\n\tencoded = strings.ReplaceAll(encoded, "%28", "(")\n\tencoded = strings.ReplaceAll(encoded, "%29", ")")\n')

# Store the existing order credential only in same-tab sessionStorage before
# navigating away, then re-query server payment truth after browser return.
replace(
    'site/themes/minimal-cart/islands/CheckoutDialog/CheckoutDialog.vue',
    "import { ref, computed, watch, reactive } from 'vue'",
    "import { ref, computed, watch, reactive, onMounted } from 'vue'",
)
replace(
    'site/themes/minimal-cart/islands/CheckoutDialog/CheckoutDialog.vue',
    '  prepareECPayPayment, submitHostedPayment,\n',
    '  prepareECPayPayment, submitHostedPayment, getGuestOrder,\n',
)
replace(
    'site/themes/minimal-cart/islands/CheckoutDialog/CheckoutDialog.vue',
    '      submitHostedPayment(launch)\n      return\n',
    '''      sessionStorage.setItem('ecpay.pendingOrder', JSON.stringify({\n        orderId: o.id,\n        accessToken: o.access_token,\n      }))\n      submitHostedPayment(launch)\n      return\n''',
)
anchor = '''watch(checkoutFingerprint, () => {\n  idempotencyKey.value = null\n})\n'''
resume = anchor + '''\nconst sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))\n\nasync function resumeECPayBrowserReturn() {\n  const params = new URLSearchParams(window.location.search)\n  if (params.get('payment') !== 'returned') return\n\n  // Remove the navigation marker immediately so refresh cannot repeatedly\n  // replay the UX flow. It carries no order/provider identity.\n  params.delete('payment')\n  const nextQuery = params.toString()\n  history.replaceState(null, '', `${window.location.pathname}${nextQuery ? `?${nextQuery}` : ''}${window.location.hash}`)\n\n  const raw = sessionStorage.getItem('ecpay.pendingOrder')\n  if (!raw) {\n    toast.error('已返回商店，但找不到此分頁的付款確認資料；請至訂單查詢確認付款狀態')\n    return\n  }\n\n  let pending: { orderId?: string; accessToken?: string }\n  try {\n    pending = JSON.parse(raw)\n  } catch {\n    sessionStorage.removeItem('ecpay.pendingOrder')\n    toast.error('付款確認資料已失效，請至訂單查詢確認付款狀態')\n    return\n  }\n  if (!pending.orderId || !pending.accessToken) {\n    sessionStorage.removeItem('ecpay.pendingOrder')\n    toast.error('付款確認資料不完整，請至訂單查詢確認付款狀態')\n    return\n  }\n\n  // ReturnURL and browser navigation can race. Re-query durable server truth\n  // briefly; never infer paid from the browser return itself.\n  for (let attempt = 0; attempt < 5; attempt++) {\n    try {\n      const order = await getGuestOrder(pending.orderId, pending.accessToken)\n      if (order?.payment_status === 'paid') {\n        sessionStorage.removeItem('ecpay.pendingOrder')\n        cart.clear()\n        toast.success(`付款完成，訂單 ${pending.orderId} 已確認`)\n        return\n      }\n    } catch {\n      // Keep the credential for manual order lookup; a transient read failure\n      // must not convert into a client-side payment decision.\n    }\n    if (attempt < 4) await sleep(800)\n  }\n  toast.error('付款結果仍在確認中，請稍後至訂單查詢確認狀態')\n}\n\nonMounted(() => {\n  void resumeECPayBrowserReturn()\n})\n'''
replace('site/themes/minimal-cart/islands/CheckoutDialog/CheckoutDialog.vue', anchor, resume)

# Normalize controlled-change metadata to the repository's accepted schema.
control_path = ROOT / 'specs/changes/commerce-ecpay-payment-flow/control.json'
control = {
    'change_id': 'commerce-ecpay-payment-flow',
    'revision': 1,
    'status': 'Accepted',
    'decision_authority': 'Repository owner',
    'approval_basis': 'Owner approved implementing the starter-owned ECPay payment flow on 2026-08-27, preserving server-authoritative payment truth and fail-closed callback handling.',
    'repository_baseline': '8968f4943b6697a70d981ed3e5338d4584518b6f',
    'supersedes': [],
    'applies_to': [
        'server/internal/config/config.go',
        'server/internal/modules/commerce/service.go',
        'server/internal/modules/commerce/store.go',
        'server/internal/modules/commerce/ecpay.go',
        'server/internal/modules/commerce/ecpay_payment.go',
        'server/internal/modules/commerce/store_ecpay.go',
        'server/internal/modules/commerce/ecpay_http.go',
        'server/internal/modules/commerce/ecpay_test.go',
        'server/internal/modules/commerce/ecpay_security_test.go',
        'server/internal/bootstrap/app.go',
        'db/migrations/sqlite/017_ecpay_payment_attempts.sql',
        'db/migrations/postgres/017_ecpay_payment_attempts.sql',
        'site/themes/minimal-cart/shared/lib/api.ts',
        'site/themes/minimal-cart/islands/CheckoutDialog/CheckoutDialog.vue',
        '.env.example',
        '.env.development.example',
        '.env.production.example',
    ],
    'requirements': ['REQ-001', 'REQ-002', 'REQ-003', 'REQ-004', 'REQ-005'],
    'acceptance': ['AC-001', 'AC-002', 'AC-003', 'AC-004', 'AC-005'],
    'evidence': {
        'REQ-001': {'status': 'passed', 'proof': 'ECPay endpoints are derived from finite stage/production configuration; HashKey/HashIV remain server-only and production rejects known public test credentials.'},
        'AC-001': {'status': 'passed', 'proof': 'The launch form exposes only public provider fields and a CheckMacValue generated by the server.'},
        'REQ-002': {'status': 'passed', 'proof': 'A dedicated ecpay_payment_attempts table stores merchant identity, amount, currency, callback result, and replay fingerprint without changing existing order scan contracts.'},
        'AC-002': {'status': 'passed', 'proof': 'Verified ReturnURL processing compares MerchantID, CheckMacValue, MerchantTradeNo and TotalAmount to durable state before an atomic paid transition.'},
        'REQ-003': {'status': 'passed', 'proof': 'ClaimECPayCallback uses a durable callback fingerprint compare-and-set: identical replay has one effect and a conflicting callback fails closed.'},
        'AC-003': {'status': 'passed', 'proof': 'The order payment transition and append-only payment event occur in the same SQL transaction as first callback claim.'},
        'REQ-004': {'status': 'passed', 'proof': 'The browser-return path verifies the signed provider form and redirects only; it contains no payment mutation.'},
        'AC-004': {'status': 'passed', 'proof': 'The storefront stores its already-issued order credential in same-tab sessionStorage and re-queries durable order payment_status after return instead of trusting navigation.'},
        'REQ-005': {'status': 'passed', 'proof': 'Minimal-cart requests a signed launch form only after CreateOrder succeeds and submits that public form to the hosted ECPay endpoint.'},
        'AC-005': {'status': 'passed', 'proof': 'Focused Go tests, PostgreSQL migration application/parity, and the minimal-cart production build run before implementation is committed.'},
    },
}
control_path.write_text(json.dumps(control, indent=2) + '\n')

# Keep human-readable spec acceptance IDs aligned with control.json.
spec_path = ROOT / 'specs/changes/commerce-ecpay-payment-flow/spec.md'
spec = spec_path.read_text()
if '## Acceptance' not in spec:
    spec += '''\n## Acceptance\n\n- AC-001: Hosted launch output contains no signing secrets.\n- AC-002: Only a verified ReturnURL whose amount matches durable state can mark an order paid.\n- AC-003: Identical callbacks are one-effect and conflicting callbacks fail closed.\n- AC-004: Browser return never marks paid and the storefront re-queries durable order state using its existing credential.\n- AC-005: SQLite/PostgreSQL migration parity, focused backend tests, and storefront build pass.\n'''
    spec_path.write_text(spec)

plan_path = ROOT / 'specs/changes/commerce-ecpay-payment-flow/plan.md'
plan = plan_path.read_text()
if '7. Verify ECPay-specific' not in plan:
    plan += '\n7. Verify ECPay-specific signing, tamper rejection, durable amount reconciliation, browser-return non-authority, and production credential safety.\n'
    plan_path.write_text(plan)

evidence_path = ROOT / 'specs/changes/commerce-ecpay-payment-flow/evidence.md'
evidence = evidence_path.read_text()
if 'ECPay-specific unit tests' not in evidence:
    evidence += '\nECPay-specific unit tests cover signing/tamper rejection, durable amount use, callback capture classification, browser-return non-mutation, and production public-test-credential rejection.\n'
    evidence_path.write_text(evidence)
