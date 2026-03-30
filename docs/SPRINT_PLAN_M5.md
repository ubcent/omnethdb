# OmnethDB V1 Sprint Plan: Milestone 5

This document turns Milestone 5 from [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md) into an execution-ready sprint plan.

Milestone 5 focus:

- inspection APIs
- auditability
- optimistic locking
- concurrent read/write consistency

Use this together with:

- [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
- [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
- [SPRINT_PLAN_M4.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M4.md)

## Sprint Goal

Make OmnethDB explainable after the fact and safe under contested writes.

At the end of this sprint, the system should support:

- full lineage inspection
- explicit relation traversal
- audit history for lifecycle operations
- targeted forget-history lookup
- optimistic locking for contended updates
- snapshot-safe reads during concurrent writes

## In Scope

Backlog items:

- `BL-035` Specify Inspection And Audit Contracts
- `BL-036` Implement Audit Log And Forget Record Storage
- `BL-037` Implement MemoryInspector APIs
- `BL-038` Verify Inspection And Audit Explainability
- `BL-039` Specify Concurrency And Optimistic Lock Semantics
- `BL-040` Implement Optimistic Locking And Conflict Detection
- `BL-041` Verify Concurrent Write And Read Consistency

Primary milestone capabilities:

- `CAP-19` Inspect Full Lineage History
- `CAP-20` Traverse Explicit Memory Relations
- `CAP-21` Audit Why The Corpus Looks The Way It Does
- `CAP-22` Stay Correct Under Concurrent Writers

Primary UAT alignment:

- `UAT-22`
- `UAT-23`
- `UAT-24`
- `UAT-25`
- `UAT-26`

## Out Of Scope

The following remain outside Milestone 5:

- embedding migration workflow
- operator backup/restore validation
- release-level UAT harness and final sign-off

## Workstreams

### Workstream A. Inspection And Audit

Objective:
Provide the cold-path visibility needed to explain state, debug lineage history, and inspect graph structure explicitly.

Tasks:

- `BL-035`
- `BL-036`
- `BL-037`
- `BL-038`

Expected outputs:

- `GetLineage`
- `GetRelated`
- `GetAuditLog`
- indexed forget-history access
- verification that inspection explains live and historical state

### Workstream B. Concurrency Safety

Objective:
Prevent silent corruption of latest-version state under contested writes while preserving read consistency.

Tasks:

- `BL-039`
- `BL-040`
- `BL-041`

Expected outputs:

- `IfLatestID` optimistic lock behavior
- conflict detection for contested updates
- tests proving one-winner behavior and snapshot-safe reads

## Suggested Execution Order

Recommended order inside the sprint:

1. `BL-035`
2. `BL-039`
3. `BL-036`
4. `BL-037`
5. `BL-040`
6. `BL-038`
7. `BL-041`

Rationale:

- freeze cold-path contracts before implementing inspector APIs
- define concurrency behavior before adding lock checks
- complete explainability and concurrency verification before moving to operations work

## Suggested Ticket Breakdown

If this sprint is turned into tickets, a good minimum split is:

1. audit log and forget-record model
2. lineage inspection API
3. relation traversal API
4. audit retrieval and explainability tests
5. optimistic locking and conflict detection
6. concurrency verification suite

## Definition Of Done

Milestone 5 sprint is done only when all of the following are true:

1. Full lineage history is retrievable in chronological order.
2. Explicit relation traversal works independently from retrieval.
3. Audit history includes writes, forgets, revives, and other lifecycle events implemented so far.
4. Forget actor and reason are inspectable through targeted history.
5. Contested updates can be rejected through optimistic locking.
6. Concurrent reads see coherent snapshots rather than partial write state.
7. Inspection and audit data can explain why a memory is live, superseded, forgotten, or revived.

## Sprint Risks

### Risk 1. Letting Inspector Semantics Bleed Into Retrieval

Why it matters:
If graph traversal logic leaks into hot-path reads, the retrieval contract becomes unpredictable.

Mitigation:

- keep inspector APIs physically and logically separate from retrieval code
- verify retrieval behavior remains unchanged after inspector implementation

### Risk 2. Incomplete Audit Coverage

Why it matters:
If some lifecycle operations are not logged consistently, explainability breaks in subtle ways.

Mitigation:

- audit every state-changing operation through a single disciplined write path
- verify history completeness using scenario-based tests

### Risk 3. Conflict Handling That Only Works In Unit Tests

Why it matters:
Optimistic locking is easy to simulate incorrectly if concurrency tests are too synthetic.

Mitigation:

- test with real contention against persisted state
- verify final lineage state, not only returned errors

### Risk 4. Forget History Becoming Detached From Audit History

Why it matters:
If forget-specific indexing diverges from general audit semantics, debugging becomes confusing.

Mitigation:

- model forget records as a targeted index over the same lifecycle truth
- test cross-consistency between forget lookup and audit log entries

## Review Checklist

Before closing the sprint, review:

- can every current lineage state be explained through lineage plus audit data?
- do `GetRelated` results include historical and forgotten records when they should?
- does `IfLatestID` reject exactly the contested writes it is meant to reject?
- can a read ever observe half-applied latest switching?
- are audit and forget-history outputs consistent with one another?

## Next Step After This Sprint

If Milestone 5 closes successfully, the next planning artifact should be:

- `SPRINT_PLAN_M6.md`

That sprint should focus on embedding migration, config completion, and operator-facing readiness.
