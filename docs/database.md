# Database

`DB_DRIVER` accepts only `sqlite` or `postgres`.

- Local default: SQLite.
- Production default: PostgreSQL (Supabase-hosted in the intended stack).
- Production SQLite remains technically possible when deployment persistence is explicitly handled.

Business modules use `database/sql`. The tiny `database.Bind` helper converts the starter's simple `?` placeholders to PostgreSQL `$1...` placeholders. Do not expand it into a SQL parser. If query complexity grows, adopt a query builder deliberately.

Every schema change should have matching migrations under:

```text
db/migrations/sqlite/
db/migrations/postgres/
```

Prefer the portable profile: tables, indexes, foreign keys, transactions, simple joins, pagination, and ordinary constraints. If a feature truly requires a PostgreSQL-only capability, document that capability and stop pretending the feature supports SQLite.
