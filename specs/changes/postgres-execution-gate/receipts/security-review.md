# Security and Governance Review: Database Schema Gate

- **Change ID**: `postgres-execution-gate`
- **Revision**: 2
- **Date**: 2026-08-14
- **Auditor**: Antigravity Implementation Agent

## 1. Governance Boundary Evaluation

- **Surface Area**: `db/` directory, encompassing all database schema definitions, SQLite migrations (`db/migrations/sqlite/`), and PostgreSQL migrations (`db/migrations/postgres/`).
- **Previous Vulnerability**: `server/tools/speccheck/main.go` omitted `db/` from `requiresControlledSpec`, allowing untracked or arbitrary SQL schema alterations without an approved controlled change specification, creating a discrepancy with `AGENTS.md:13`.
- **Enforcement Remediation**: Added `"db/"` to `requiresControlledSpec` in `server/tools/speccheck/main.go`. Unit testing in `server/tools/speccheck/main_test.go` verifies that any file modification under `db/` requires an authorized controlled spec.

## 2. Secrets and Ephemeral Infrastructure Review

- **CI Database Credentials**: The PostgreSQL service container defined in `.github/workflows/ci.yml` uses non-secret dummy credentials (`POSTGRES_DB: test_db`, `POSTGRES_USER: test_user`, `POSTGRES_PASSWORD: test_password`).
- **Production Isolation**: No production connection strings, database secrets, or external network services are accessed or referenced during local testing or CI verification.
- **Data Integrity & Consistency**: Mechanical migration parity (`server/tools/migration-parity`) prevents schema divergence between development and production databases.
