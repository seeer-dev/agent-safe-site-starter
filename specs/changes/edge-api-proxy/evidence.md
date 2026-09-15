# Edge API Proxy — Evidence

Change ID: edge-api-proxy
Revision: 1
Status: Accepted

| Gate | Status | Proof |
|---|---|---|
| REQ-001 | passed | wrangler pages dev dist --binding API_ORIGIN=http://localhost:8080: GET /api/products through :8788 returned identical product JSON as direct :8080; GET / returned static 200 — function intercepts /api/* only. |
| REQ-002 | passed | Echo upstream (:8099) observed: x-edge-secret present on GET+POST, cookie:null despite sent Cookie, Authorization passed, path+query preserved, POST body+content-type intact. |
| REQ-003 | passed | Guide step 6 no longer instructs api. CNAME or Transform Rule; diagram shows Pages Function hop; checklist has Pages-Functions-env + proxy-verification items; env.example + ownership table list API_ORIGIN/EDGE_SECRET as pages-fn vars. |
| REQ-004 | passed | AGENTS.md hard-boundary bullet now names specs/changes/edge-api-proxy as the sole approved Pages Functions use. |
| AC-001 | passed | /api/products via :8788 == direct :8080 body; static / unaffected (200). |
| AC-002 | passed | Upstream saw X-Edge-Secret with bound value; Cookie header absent upstream. |
| AC-003 | passed | grep of guide/README/env docs: no remaining api. subdomain or Transform Rule instructions; ECPay section points ReturnURL at site origin. |
| AC-004 | passed | AGENTS.md: "Approved exception: functions/api/[[path]].js ... specs/changes/edge-api-proxy/ — no other edge compute is authorized." |

