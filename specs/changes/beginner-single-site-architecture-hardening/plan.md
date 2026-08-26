# Beginner Single-Site Architecture Hardening Delivery Plan

Change ID: beginner-single-site-architecture-hardening
Revision: 1
Status: Verifying

Repository baseline: `ade97a8d58dfffe1d2b61633d552dfffcb7ba3f6`

## Slices

### Slice 1: Versioned manifest-driven import enforcement

- add `enforcement.version: 1` and explicit deny rules to `architecture.yaml`;
- load the policy from the manifest;
- reject unsupported versions and any attempt to weaken hard deny rules;
- validate roots are repository-relative, strict descendants, distinct, present directories;
- require non-zero scan/module/platform non-test Go coverage.

Verification:

```text
go test ./server/tools/archcheck -count=1
go run ./server/tools/archcheck
```

### Slice 2: Starter-owned architecture guidance

- preserve one site, one Go backend, static-first public delivery, separate Vue admin, scoped islands;
- use consumer-owned typed ports plus bootstrap wiring for synchronous cross-module behavior;
- split large modules internally by cohesion before creating top-level modules;
- preserve checked-in UI/reference surfaces as acceptance contracts;
- keep speculative platform abstractions out of the starter.

### Slice 3: Repository verification and merge

- run architecture tests and real scan in PR CI;
- reconcile controlled evidence;
- move the change to Accepted only after the targeted architecture gate has passed;
- rerun the full CI chain, including migrations, unit tests, live PostgreSQL gate, concurrency stress tests, and `go vet`;
- merge only after the full CI is green.
