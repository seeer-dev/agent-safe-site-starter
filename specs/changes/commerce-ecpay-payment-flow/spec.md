# Commerce ECPay Payment Flow

## Goal

Connect durable starter orders to ECPay AIO v5 credit-card checkout without making browser navigation authoritative for payment state.

## Requirements

### REQ-001 Server-owned launch
The server MUST derive the ECPay endpoint and callback URLs from finite environment configuration, keep HashKey/HashIV server-only, and sign the hosted form.

### REQ-002 Durable payment truth
Each order MUST have at most one starter-owned ECPay payment attempt. The provider ReturnURL MUST verify CheckMacValue, MerchantID, merchant trade identity, and amount against durable state before payment_status can become paid.

### REQ-003 Replay and conflict safety
An identical verified callback MAY be acknowledged repeatedly with one durable effect. A conflicting terminal callback MUST fail closed and MUST NOT overwrite the durable result.

### REQ-004 Browser return is not payment truth
OrderResultURL/browser return MUST verify the provider form and redirect only. It MUST NOT transition payment state.

### REQ-005 Storefront handoff
After a durable order is created with an enabled ready ECPay method, the minimal-cart storefront MUST request the signed launch form from the server and POST it to ECPay.
