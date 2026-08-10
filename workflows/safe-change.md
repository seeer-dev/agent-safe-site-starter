# Safe change workflow

Internal agent SOP; users should not have to invoke it by name.

```text
user intent
  -> read site skill
  -> inspect owner module
  -> impact discovery (CodeGraph if available)
  -> write narrow .ai/scope.json
  -> implement
  -> scopecheck
  -> archcheck/tests/vet
  -> render when public output changed
  -> explain what changed in user terms
```
