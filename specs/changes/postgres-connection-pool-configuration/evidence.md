# Evidence

## Delivery status

Revision 1 implemented and verified on 2026-08-18. Both behavioural gates were mutation-verified: each was observed failing for a named trigger and green after restoration, with no mutation left in the diff.

## Observed evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | database.PoolConfig plus OpenWithPool apply MaxOpenConns, MaxIdleConns, ConnMaxLifetime, and ConnMaxIdleTime for PostgreSQL, sourced from DB_MAX_OPEN_CONNS, DB_MAX_IDLE_CONNS, DB_CONN_MAX_LIFETIME, and DB_CONN_MAX_IDLE_TIME through config.Load and wired in bootstrap. Open retains its previous signature and delegates with an empty PoolConfig. |
| REQ-002 | passed | applyPool keeps SQLite at SetMaxOpenConns(1) in a separate branch, and all four settings appear in the ownership table, the production inventory, and the development example, with an explicit statement that sizing depends on the pooler mode. |
| AC-001 | passed | TestPoolConfigWithDefaults covers empty, supplied, zero, negative, and idle-above-open input; envInt and envDuration return 0 for absent, unparseable, zero, or negative values so the platform default applies rather than an unbounded pool. TestApplyPoolPerDialect asserts a configured bound of 7 is applied and that an empty config applies DefaultMaxOpenConns. Mutation-verified: removing db.SetMaxOpenConns from the PostgreSQL branch turned the assertion red (got 0), and restoration returned it to green. |
| AC-002 | passed | TestApplyPoolPerDialect opens SQLite with a 50-connection PoolConfig and asserts Stats().MaxOpenConnections is still 1. Mutation-verified: replacing the SQLite branch condition with a constant false turned the assertion red, and restoration returned it to green. |
| AC-003 | passed | All four names appear in docs/environment-configuration.md (ownership table plus a Connection pool bounds section), .env.production.example with Railway ownership comments, and .env.development.example as commented defaults. The documentation states the defaults are a conservative floor, distinguishes session from transaction pooler sizing, notes the pgx simple_protocol requirement under transaction pooling, and records that choosing the pooler mode is an operator decision this repository does not make. |
