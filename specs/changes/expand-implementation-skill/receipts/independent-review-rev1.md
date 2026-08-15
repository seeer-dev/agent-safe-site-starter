# Independent forward-test review

Change ID: expand-implementation-skill
Revision: 1
Receipt kind: independent-review
Reviewers: two isolated Codex-native forward-test contexts that did not implement this change
Method: read-only replay from repository artifacts; no files, commits, or controlled evidence were modified by the reviewers

## Replay 1: Admin configurable API base

The reviewer read the new skill and reference, inspected the Accepted `admin-configurable-api-base` change and its real admin consumers, and produced a representative packet for shared API-prefix consumption. The packet traced the Vite build producer through `getApiBase`, generic/media clients, auth token state, and UI consumers; it separated direct R2 upload authority from Go API prefixing and named exact tests and mutations.

The first replay exposed seven format defects:

- no fixed baseline, observed HEAD, or dirty-path context;
- no stable mapping from existing slice IDs to packet IDs;
- no working-directory field for exact commands;
- no status for behavior already present in the inspected tree;
- no observed-implementation format for retrospective replay;
- no retrospective evidence gate; and
- no safe output rule for Accepted/Superseded artifacts that are immutable in the comparison base.

The skill and reference were revised to cover all seven. The same reviewer re-read the revision and returned `PASS` with no remaining essential omission.

## Replay 2: PostgreSQL lock semantics and evidence

A separate reviewer read the revised skill and inspected the Applying `postgres-lock-semantics-and-evidence` change. It produced a representative exact-test-inventory packet with:

- proposal baseline, observed HEAD, and pre-existing dirty paths;
- stable slice-to-packet mapping;
- exact owner, contract, CI consumer, and test anchors;
- read, modify, and must-not-modify boundaries;
- before/after behavior and authority;
- an end-to-end CI-to-`go test -json` trace;
- working directory, exact argv, named assertions, negative cases, mutations, and restoration checks; and
- explicit environment classification for live PostgreSQL and CI-only proof without substituting SQLite or mocks.

The replay also detected that the source proposal's broad slices and role-labelled tasks would need expansion or removal before serving as low-inference packets. This is the intended skill behavior: repository gaps remained visible instead of being inferred away. The reviewer found no new product decision and did not identify an additional essential omission in the revised skill.

## Independence and conclusion

Both reviewers received the skill and raw repository artifacts rather than the implementer's intended packet output. They performed no writes. The two surfaces were materially different: browser build/runtime configuration and backend/CI concurrency evidence. Revision 1 transfers across both, preserves identity-neutral semantics, and emits explicit drift/environment stops when exact implementation or proof cannot proceed.
