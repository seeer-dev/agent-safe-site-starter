from pathlib import Path

p = Path('site/themes/minimal-cart/islands/CheckoutDialog/CheckoutDialog.vue')
text = p.read_text()

def replace_once(old, new):
    global text
    if new in text:
        return
    if old not in text:
        raise SystemExit(f'anchor not found: {old[:120]!r}')
    text = text.replace(old, new, 1)

replace_once("import { ref, computed, watch, reactive } from 'vue'", "import { ref, computed, watch, reactive, onMounted } from 'vue'")
replace_once(
    '  prepareECPayPayment, submitHostedPayment,\n',
    '  prepareECPayPayment, submitHostedPayment, getGuestOrder,\n',
)
anchor = '''watch(checkoutFingerprint, () => {\n  idempotencyKey.value = null\n})\n'''
resume = anchor + '''\nconst sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))\n\nasync function resumeECPayBrowserReturn() {\n  const params = new URLSearchParams(window.location.search)\n  if (params.get('payment') !== 'returned') return\n\n  params.delete('payment')\n  const nextQuery = params.toString()\n  history.replaceState(null, '', `${window.location.pathname}${nextQuery ? `?${nextQuery}` : ''}${window.location.hash}`)\n\n  const raw = sessionStorage.getItem('ecpay.pendingOrder')\n  if (!raw) {\n    toast.error('已返回商店，但找不到此分頁的付款確認資料；請至訂單查詢確認付款狀態')\n    return\n  }\n\n  let pending: { orderId?: string; accessToken?: string }\n  try {\n    pending = JSON.parse(raw)\n  } catch {\n    sessionStorage.removeItem('ecpay.pendingOrder')\n    toast.error('付款確認資料已失效，請至訂單查詢確認付款狀態')\n    return\n  }\n  if (!pending.orderId || !pending.accessToken) {\n    sessionStorage.removeItem('ecpay.pendingOrder')\n    toast.error('付款確認資料不完整，請至訂單查詢確認付款狀態')\n    return\n  }\n\n  // Provider ReturnURL and browser navigation can race. The browser only\n  // re-queries durable server truth; it never infers paid from navigation.\n  for (let attempt = 0; attempt < 5; attempt++) {\n    try {\n      const order = await getGuestOrder(pending.orderId, pending.accessToken)\n      if (order?.payment_status === 'paid') {\n        sessionStorage.removeItem('ecpay.pendingOrder')\n        cart.clear()\n        toast.success(`付款完成，訂單 ${pending.orderId} 已確認`)\n        return\n      }\n    } catch {\n      // Preserve the existing credential for manual lookup on transient reads.\n    }\n    if (attempt < 4) await sleep(800)\n  }\n  toast.error('付款結果仍在確認中，請稍後至訂單查詢確認狀態')\n}\n\nonMounted(() => {\n  void resumeECPayBrowserReturn()\n})\n'''
replace_once(anchor, resume)
replace_once(
    '''      submitHostedPayment(launch)\n      return\n''',
    '''      sessionStorage.setItem('ecpay.pendingOrder', JSON.stringify({\n        orderId: o.id,\n        accessToken: o.access_token,\n      }))\n      submitHostedPayment(launch)\n      return\n''',
)
p.write_text(text)
