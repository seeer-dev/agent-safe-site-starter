# Data and forms

For a form or persistent model, trace one vertical path:

```text
browser -> Go handler -> service -> store -> database
```

Use plain JSON API endpoints. Keep validation in the service when it is business validation; keep request decoding in the handler. A module must not import another module directly.

Schema changes need both SQLite and PostgreSQL migrations unless the requested feature is explicitly database-specific.
