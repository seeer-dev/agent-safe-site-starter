# Commerce Configuration Model Cohesion Evidence

Change ID: commerce-configuration-model-cohesion
Revision: 1
Status: Accepted

| ID | Status | Proof |
| --- | --- | --- |
| REQ-001 | passed | Promo and PaymentMethod declarations are colocated in promotions.go and payment_methods.go while retaining package commerce and their existing exported names/JSON tags. |
| AC-001 | passed | The moved Promo, PromoInput, PaymentMethod, and PaymentMethodInput declarations preserve their field names, types, and JSON tags from the baseline. |
| REQ-002 | passed | ShippingMethod declarations are colocated in shipping_methods.go with the existing shipping validation and service behavior, still inside package commerce. |
| AC-002 | passed | ShippingMethod, ShippingMethodInput, and ShippingMethodUpdateInput preserve exported names, field types, and JSON tags; no route, storage, schema, or service contract is changed. |
