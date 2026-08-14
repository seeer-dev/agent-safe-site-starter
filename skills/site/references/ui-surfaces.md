# UI surfaces

Use this reference during proposal and acceptance work whenever a change affects a user-visible page, panel, dialog, form, state, or interaction. A surface contract connects user intent to data ownership and testable experience behavior.

## Build the surface inventory

Inspect real routes, mount points, layouts, components, forms, API clients, renderer templates, fixtures, permission gates, and tests. Inventory current and proposed surfaces; do not derive the list only from the plan.

For each surface, record:

```text
Surface ID and name:
Requirement and acceptance IDs:
Route or mount point:
Static render or runtime owner:
Personas and permissions:
Primary task:
Risk if wrong:
Read operations and evidence source:
Write operations and authoritative owner:
States: empty / loading / error / forbidden / success
Primary and secondary actions:
Consequence, feedback, and recovery:
Critical journeys:
Acceptance evidence:
```

For static output, identify the store-to-render-to-`dist/` path. For runtime behavior, identify the browser-to-Go-API path. A label such as "products API" is not a data contract: name the route or operation, fields needed, authoritative owner, permission, failure behavior, and repository evidence.

## Universal experience obligations

Every important surface must answer five questions:

1. **Orientation:** Where am I, and what object or scope am I acting on?
2. **Attention:** What needs action now, including risk, errors, or pending state?
3. **Primary action:** What is the next meaningful action, and is it keyboard and touch accessible?
4. **Consequence:** What will change, who or what is affected, and is confirmation needed?
5. **Feedback and recovery:** Did it work, what is the resulting state, and how can failure be corrected or retried?

Apply these obligations to the actual task and risk. Do not copy a role vocabulary, dashboard pattern, or owner-operations rule from another product unless repository and product evidence requires it.

## State contract

| State | Contract question |
|---|---|
| Empty | Is absence distinguished from failure, and is the next action useful? |
| Loading | Is progress visible without presenting stale data as current? |
| Error | Is the failure specific, safe, and recoverable? |
| Forbidden | Is unauthorized access blocked without leaking protected data? |
| Success | Is real data shown with the correct actions and resulting state? |

Include only states that can occur, but never omit a relevant state merely because fixtures do not demonstrate it. Forms also need initial, dirty, invalid, submitting, succeeded, and failed behavior where applicable.

## Interaction and accessibility contract

- Define triggers, validation timing, confirmation, optimistic behavior, cancellation, retry, and post-mutation refresh or navigation.
- Preserve focus, labeling, error association, keyboard operation, readable status changes, and responsive behavior.
- State whether an action is reversible and what evidence proves the final authoritative state.
- Keep fixture and mock behavior explicitly separated from production behavior.

## Coverage matrix

Use a compact matrix for non-trivial UI work:

| Surface | REQ/AC | Persona | Primary task | Data owner | Relevant states | Critical journey | Evidence |
|---|---|---|---|---|---|---|---|

Reverse-check the matrix against actual routes and components. An unlisted real surface, persona, write path, or state is a planning gap; a planned surface with no mount point or owner is an unverified claim.

## Completion gate

The UI contract is ready when every in-scope surface maps to controlled-spec REQ/AC IDs and has a real location, allowed personas, task and risk, authoritative data path, relevant states, interaction consequences, feedback/recovery, critical journey, and observable acceptance evidence. Visual polish cannot substitute for missing truth, permissions, or failure behavior.
