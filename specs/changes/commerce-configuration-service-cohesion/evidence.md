# Commerce Configuration Service Cohesion Evidence

Change ID: commerce-configuration-service-cohesion
Revision: 1
Status: Accepted

| ID | Status | Proof |
| --- | --- | --- |
| REQ-001 | passed | Promotion CRUD and discount validation/calculation are colocated in promotions.go while retaining the existing Service method names and package commerce. |
| AC-001 | passed | ListPromos, CreatePromo, UpdatePromo, DeletePromo, and calculateDiscount retain their existing method bodies and call the same Store operations and shared arithmetic helpers. |
| REQ-002 | passed | Payment public discovery, validation, listing, and update behavior are colocated in payment_methods.go inside package commerce. |
| AC-002 | passed | PublicPaymentMethod, ListPublicPaymentMethods, validatePaymentMethod, ListPaymentMethods, and UpdatePaymentMethod retain their existing exported or package-local contracts and error behavior. |
| REQ-003 | passed | Shipping public discovery and authoritative fee calculation are colocated with shipping admin behavior in shipping_methods.go without changing checkout orchestration. |
| AC-003 | passed | ListPublicShippingMethods and computeShipping retain their existing logic; Quote and CreateOrder continue calling the same Service helpers with no route, schema, storage, or API change. |
