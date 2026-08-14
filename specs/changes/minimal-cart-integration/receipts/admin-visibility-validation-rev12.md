# Revision 12 admin visibility validation

Date: 2026-08-14  
Reviewer: Codex (independent of bounded Grok implementation)

## Scope

- `admin/src/pages/DashboardPage.vue`: remove the nested native `template`
  element that made loaded dashboard content inert.
- `server/internal/modules/staff/http.go`: require `staff.read` for the staff
  list only. Staff create, update, status update, and delete retain their
  `staff.update` service authorization.

## Automated checks

| Command | Result |
| --- | --- |
| `go test ./server/internal/modules/staff -count=1` | passed |
| `npm test` in `admin/` | passed: 13 files, 153 tests |
| `npm run build` in `admin/` | passed: Vue typecheck and Vite production build |
| `go run ./server/tools/speccheck` | passed before evidence update; replayed after this receipt |

The focused regression coverage proves all of the following:

- dashboard KPIs and module/task DOM render after the orders and products reads
  resolve, and no nested `template` remains;
- staff list with no credential returns 401;
- a principal with only `staff.update` returns 403 for the list;
- a principal with only `staff.read` receives a members response.

## Linked-manager browser replay

After restarting the local Go development server, the existing linked manager
session at `http://127.0.0.1:5174` was observed without data mutation.

- Dashboard: four visible, non-zero-height KPI cards reported `0`, `0`, `0`,
  and `3`; the low-stock task and both operational panels were visible. No
  inert dashboard `template` remained and the browser reported no warning/error
  log entries.
- Staff: the list loaded three staff rows with no loading or error state. The
  manager saw no enabled mutation control. The New, Edit, and Disable controls
  were rendered disabled, preserving the `staff.update` boundary.
- Products, content, and payment methods had existing records. Orders,
  members, promos, and shipping methods remained honest zero-record empty
  states in the local database.

## Remaining limits

This receipt validates the revision-12 local repair only. It does not complete
the broader controlled change, including live OAuth provider, PostgreSQL, R2,
and Cloudflare acceptance work.
