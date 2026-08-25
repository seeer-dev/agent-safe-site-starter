# Beginner single-site architecture boundary

This repository is a starter for a non-technical owner who asks an AI agent to build and evolve one site. Architecture exists to keep that workflow reliable; it is not a goal to turn the starter into a general application platform.

## Product shape

Preserve these defaults unless the user explicitly changes the product architecture:

- one site and one deployment composition;
- one Go application backend;
- static-first public publishing to Cloudflare Pages;
- a separate Vue admin SPA;
- scoped Vue islands only where interaction needs them;
- SQLite for local use and PostgreSQL as the production default.

Do not introduce multi-site runtime selection, a site/module provider registry, Composer/`ResolvedPlan`-style planning, dynamic plugins, a service locator, or a runtime DI container to prepare for hypothetical future sites.

## Abstraction budget

Prefer the concrete local solution. Add a reusable seam only when at least one is true:

1. two real implementations or consumers already need the same stable contract;
2. a test needs a seam around an external dependency or nondeterministic boundary;
3. the seam protects a cross-cutting correctness/security boundary such as identity, authorization, transactionality, idempotency, storage, mail, or time.

A future provider, future second site, or cleaner-looking diagram is not sufficient justification.

## Cross-module synchronous collaboration

A business module must not import another business module directly. When one module needs behavior owned by another:

1. the consuming module defines the smallest interface it needs;
2. bootstrap owns the adapter/wiring;
3. the provider module keeps its own model and storage ownership;
4. do not create a registry or package-global lookup.

This is a typed port, not a plugin system.

## Module growth and cohesion

A large module is not automatically evidence for a new top-level module. Split cohesive internals first while preserving one business boundary and its tests. Promote a subdomain only when independent ownership, lifecycle, external contract, or multiple real consumers justify it.

For the current `commerce` module, catalog/products, ordering, returns/restock, promotions, payment-method configuration, and shipping-method configuration are valid cohesion seams to evaluate. A later cleanup should move behavior in small test-preserving slices rather than perform a single structural rewrite.

## UI-first integration

When a checked-in UI, mockup, route, or reference flow already expresses the product behavior, treat it as the user-facing acceptance contract for information architecture, visible fields, actions, states, and navigation unless a reviewed product/design change supersedes it.

Backend ownership answers where truth lives; it does not by itself authorize a new page, route, navigation item, or substitute metric. If canonical data is not ready, preserve the UI position and show an explicit unavailable/error state rather than inventing fixture or browser-local production truth.

## Provider and framework changes

Use the existing provider and framework path first. A new provider seam or framework needs a real current requirement and must preserve the beginner workflow. Do not expose architecture choices, change IDs, verification commands, or provider jargon to the user when the agent can safely choose the repository default.
