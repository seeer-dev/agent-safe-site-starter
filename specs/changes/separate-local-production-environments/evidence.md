# Evidence

## Delivery status

Revision 1 implemented and verified on 2026-08-14 under a plain `apply` instruction from the repository owner. All three slices are delivered.

- Repository baseline: `3c3a475c1bdbd27dc13f4c19e91b4b37dc4638f0`.
- Receipts: [`receipts/security-review.md`](receipts/security-review.md), [`receipts/consumer-reachability.md`](receipts/consumer-reachability.md).

## Observed evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | config.Load reads APP_ENV from the process environment via isProductionEnv() before opening any dotenv file, then loads .env.development.local and .env only outside production. Verified by go test ./server/internal/config -count=5. |
| REQ-002 | passed | Both admin/vite.config.ts and site/themes/minimal-cart/vite.config.ts select process.env for production mode and loadEnv for development, and build their define map from a literal BROWSER_SAFE_KEYS allowlist. |
| REQ-003 | passed | Placeholder-only .env.development.example and .env.production.example added; .gitignore covers all six real dotenv profiles with negations for the three examples; docs/environment-configuration.md documents provider ownership and the browser allowlist. |
| REQ-004 | passed | No API route, authentication semantic, schema, rendering behavior, or deployment contract changed. gofmt, go vet, and go test ./... are clean; admin runs 194 passing tests; theme build:check passes; check:resource-contracts PASSED. |
| AC-001 | passed | TestLoadDevelopmentPrecedence shows the process value winning over both files, .env.development.local beating legacy .env, and a name present only in .env still resolving. TestLoadDevelopmentLegacyOnly and TestLoadDevelopmentWithoutDotEnvFiles cover the legacy-only and no-file cases. |
| AC-002 | passed | TestLoadProductionIgnoresRepositoryDotEnvFiles, TestLoadProductionFailsClosedWithoutProcessConfig, and TestDotEnvCannotDeclareProduction show production ignoring both dotenv files, failing closed on missing process configuration, and a dotenv APP_ENV=production being unable to move the boundary. |
| AC-003 | passed | A development-mode admin build emitted the .env.development.local sentinels, and with ADMIN_API_BASE unset everywhere the bundle carries an empty ADMIN_API_BASE that resolveApiBase maps to the /api default. Detail recorded in receipts/consumer-reachability.md. |
| AC-004 | passed | Fresh production builds of both consumers, with conflicting dotenv sentinels present on disk, emitted only the approved process values; all eight sentinels were absent from both outputs. Table recorded in receipts/security-review.md. |
| AC-005 | passed | git check-ignore confirms all six real dotenv profiles are ignored and all three example files remain trackable; a credential-pattern scan over the examples returned no matches. |
| AC-006 | passed | Four server-secret sentinels present in the build process environment produced zero occurrences in admin/dist/ and site/themes/minimal-cart/dist/, and only the named allowlist crosses the boundary with ADMIN_API_BASE remaining admin-only; recorded in receipts/security-review.md. |
| AC-007 | passed | Five consumers were exercised through real entry points: Go development config, Go production config, admin Vite in both modes, public-theme Vite production, and the Git ignore and template inventory; recorded in receipts/consumer-reachability.md. |
| AC-008 | passed | No migration of the developer .env was performed because the precedence chain keeps it working. Two temporary files were confirmed absent beforehand and deleted afterwards, and the existing .env was never opened, read, printed, or written; recorded in receipts/consumer-reachability.md. |

## Commands replayed

```text
gofmt -l server                                        (no output)
go vet ./...                                           ok
go test ./... -count=1                                 ok
go test ./server/internal/config -count=5              ok
npm --prefix admin run typecheck                       ok
npm --prefix admin test -- --run                       16 files, 194 tests passed
npm --prefix admin run build:only                      built
npm --prefix admin run check:resource-contracts        PASSED
npm --prefix site/themes/minimal-cart run build:check  built (a11y + hydration checks passed)
git check-ignore -q <six real dotenv profiles>         all ignored
```

## Residual notes

- The two `BROWSER_SAFE_KEYS` allowlists are maintained independently in the two Vite configs. The divergence is intended (admin has `ADMIN_API_BASE`), but nothing mechanically compares them; an accidental future divergence would not be caught automatically.
- `.env.production.example` is an inventory checklist. It is not deployable, and the documentation states that no real `.env.production` should be created.
- The sentinel scans prove absence in freshly built output. They are a point-in-time check, not a standing gate; a future change to either Vite config could regress the boundary without failing an existing test.
