# OmnethDB V1 Sprint Plan: Milestone 1

This document turns Milestone 1 from [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md) into an execution-ready sprint plan.

Milestone 1 focus:

- domain foundation
- space bootstrap
- storage architecture baseline

Use this together with:

- [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
- [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)

## Sprint Goal

Establish a stable and verified foundation for `v1` so subsequent milestones can build on:

- canonical domain types and errors
- deterministic first-write space creation
- a concrete persisted storage model with hot/cold separation

At the end of this sprint, the codebase does not need to support the full product yet, but it must have a trustworthy substrate for the rest of the system.

## In Scope

Backlog items:

- `BL-001` Define Domain Types And Error Surface
- `BL-002` Implement Core Domain Model And Validation Helpers
- `BL-003` Verify Domain Reject Cases
- `BL-004` Specify Space Bootstrap And Config Lock Semantics
- `BL-005` Implement Space Config Storage And First-Write Bootstrap
- `BL-006` Verify Space Bootstrap, Locking, And Override Behavior
- `BL-007` Specify Bucket Layout And Hot/Cold Path Boundaries
- `BL-008` Implement Core Buckets And Persistence Primitives
- `BL-009` Verify Storage Invariants And Persistence Boundaries

Primary milestone capabilities:

- `CAP-01` Create And Lock A Space On First Write
- `CAP-02` Apply Per-Space Runtime Behavior
- `CAP-24` Operate As A Single-File Embedded Store

Primary UAT alignment:

- `UAT-01`
- `UAT-02`
- `UAT-03`
- `UAT-28`

## Out Of Scope

The following are explicitly deferred to later milestones:

- lineage transitions
- `Remember` end-to-end write semantics
- `Forget`, `Revive`, derived memories
- retrieval ranking and profile assembly
- audit APIs
- migration workflow

## Workstreams

### Workstream A. Domain Contract

Objective:
Freeze the core language of the system before storage and APIs grow around it.

Tasks:

- `BL-001`
- `BL-002`
- `BL-003`

Expected outputs:

- canonical Go types for domain entities and config entities
- error surface for domain and validation failures
- reusable validation helpers
- unit tests for reject cases

### Workstream B. Space Bootstrap

Objective:
Make spaces real runtime entities with deterministic initialization and embedder locking.

Tasks:

- `BL-004`
- `BL-005`
- `BL-006`

Expected outputs:

- persisted space config model
- first-write bootstrap path
- embedder mismatch rejection path
- tests for initial space creation and mismatch behavior

### Workstream C. Storage Baseline

Objective:
Create the physical persistence primitives needed for all later milestones.

Tasks:

- `BL-007`
- `BL-008`
- `BL-009`

Expected outputs:

- bucket layout implementation
- persistence helpers and transaction scaffolding
- tests proving hot/cold storage expectations

## Suggested Execution Order

Recommended order inside the sprint:

1. `BL-001`
2. `BL-004`
3. `BL-007`
4. `BL-002`
5. `BL-005`
6. `BL-008`
7. `BL-003`
8. `BL-006`
9. `BL-009`

Rationale:

- start by freezing contracts before implementation
- specify space bootstrap before persisting it
- specify storage layout before encoding data into it
- finish each area with verification before calling the sprint done

## Suggested Ticket Breakdown

If this sprint is turned into tickets, a good minimum split is:

1. domain types and validation
2. domain error model and reject cases
3. space config model and bootstrap semantics
4. embedder lock enforcement
5. bucket layout and transaction primitives
6. persistence invariant tests

This is small enough to execute, but large enough to avoid ticket fragmentation.

## Definition Of Done

Milestone 1 sprint is done only when all of the following are true:

1. Domain model and error types are committed and used consistently.
2. Spaces can be bootstrapped on first write without manual pre-creation.
3. Embedder model and dimension locking are enforced in code and tests.
4. Storage primitives exist for hot-path and cold-path buckets.
5. Storage tests prove the intended separation between live retrieval data and inspection data.
6. Documentation and code do not disagree on the shape of space config or storage buckets.

## Sprint Risks

### Risk 1. Overdesigning The Domain Layer

Why it matters:
It is easy to introduce abstractions that look clean now but constrain later milestones.

Mitigation:

- keep domain types close to the architecture
- avoid speculative abstractions for APIs not yet implemented

### Risk 2. Letting Storage Shape The Domain

Why it matters:
If bucket layout decisions leak into the domain model too early, later semantics get distorted.

Mitigation:

- validate domain contract first
- treat storage as an implementation of the contract, not the source of it

### Risk 3. Fake Bootstrap Behavior

Why it matters:
A bootstrap path that works only in test scaffolding will create expensive rework in Milestone 2.

Mitigation:

- test through real persistence primitives
- verify persisted space config, not just returned values

## Review Checklist

Before closing the sprint, review:

- are domain errors specific enough to support later UAT assertions?
- does space creation happen exactly once per new space?
- can the code reject mismatched embedders without partial writes?
- is the bucket layout future-compatible with inspection, retrieval, and migration needs?

## Next Step After This Sprint

If Milestone 1 closes successfully, the next planning artifact should be:

- `SPRINT_PLAN_M2.md`

That sprint should focus on lineage semantics, governance, and the main `Remember` write path.
