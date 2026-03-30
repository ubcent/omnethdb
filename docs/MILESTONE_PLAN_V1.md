# OmnethDB Full-Spec V1 Milestone Plan

This document groups the `v1` backlog into delivery milestones.

Its purpose is to turn the backlog in [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md) into an execution plan with:

- milestone goals
- scoped backlog items
- dependency-aware sequencing
- acceptance focus per wave

Use this together with:

- [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
- [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
- [CAPABILITY_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/CAPABILITY_MAP_V1.md)
- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
- [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)

## Planning Approach

The milestones below are not arbitrary delivery buckets.

Each one is intended to leave the system in a coherent state with a meaningful capability gain:

- foundation first
- then core write-path correctness
- then retrieval correctness
- then inspectability and operational safety
- then release-grade UAT closure

## Milestone 0. Planning Baseline

Goal:
Freeze the planning stack so implementation work starts from a stable contract.

Deliverable:

- architecture source of truth
- spec map
- capability map
- UAT map
- spec-driven backlog
- milestone plan

Included docs:

- [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
- [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
- [CAPABILITY_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/CAPABILITY_MAP_V1.md)
- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
- [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
- [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md)

Exit criteria:

- planning artifacts are internally consistent
- traceability chain is established from spec to UAT

## Milestone 1. Domain, Space, And Storage Foundation

Goal:
Establish the domain contract, first-write space bootstrap, and physical persistence model.

Why this milestone matters:
Everything else depends on having a stable model, deterministic space creation, and a storage layout that cleanly separates hot and cold paths.

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

Primary capabilities unlocked:

- `CAP-01`
- `CAP-02`
- `CAP-24`

Primary UAT coverage target:

- `UAT-01`
- `UAT-02`
- `UAT-03`
- `UAT-28`

Exit criteria:

- space bootstrap is real, not mocked
- embedder locking is enforced
- persisted storage primitives exist for hot and cold paths
- storage invariants are verified

## Milestone 2. Lineage, Governance, And Core Write Path

Goal:
Make write operations semantically correct and policy-governed.

Why this milestone matters:
This is where OmnethDB starts behaving like versioned memory instead of plain record storage.

Backlog items:

- `BL-010` Specify Lineage, Versioning, And Relation Rules
- `BL-011` Implement Lineage State Transitions And Relation Validation
- `BL-012` Verify Lineage And Relation Invariants
- `BL-013` Specify Writer Policies, Promotion, Trust, And Limits
- `BL-014` Implement Policy Enforcement And Limit Checks
- `BL-015` Verify Governance, Trust, And Limit Semantics
- `BL-016` Specify Remember API Contract
- `BL-017` Implement Remember End-To-End
- `BL-018` Verify Remember Atomicity And Reject Behavior

Primary capabilities unlocked:

- `CAP-03`
- `CAP-04`
- `CAP-05`
- `CAP-07`
- `CAP-16`
- `CAP-17`
- `CAP-18`

Primary UAT coverage target:

- `UAT-04`
- `UAT-05`
- `UAT-06`
- `UAT-09`
- `UAT-19`
- `UAT-20`
- `UAT-21`

Exit criteria:

- root writes and updates are correct
- `Extends` and `Updates` are validated
- `Remember` is transactional
- policy and limit enforcement are active

## Milestone 3. Derived Memory And Lifecycle Semantics

Goal:
Complete the advanced write-path lifecycle with derived knowledge, forgetting, inactive lineages, revive, and orphaned-source handling.

Why this milestone matters:
This milestone turns the system from a basic versioned store into the full-spec memory model promised by the architecture.

Backlog items:

- `BL-019` Specify Derived Memory And Provenance Contract
- `BL-020` Implement Derived Memory Validation And Persistence
- `BL-021` Verify Derived Write Validity And Failure Paths
- `BL-022` Specify Forget, Inactive Lineage, And Revive Semantics
- `BL-023` Implement Forget And Inactive Lineage Handling
- `BL-024` Implement Revive For Inactive Lineages
- `BL-025` Verify Forget, Inactive, And Revive Behavior
- `BL-032` Specify Orphaned Source Propagation
- `BL-033` Implement Source-Forget Propagation To Derived Memories
- `BL-034` Verify Orphaned Source Behavior

Primary capabilities unlocked:

- `CAP-06`
- `CAP-08`
- `CAP-09`
- `CAP-10`
- `CAP-11`

Primary UAT coverage target:

- `UAT-07`
- `UAT-08`
- `UAT-10`
- `UAT-11`
- `UAT-12`
- `UAT-13`

Exit criteria:

- derived writes are fully governed and validated
- forgetting affects live corpus correctly
- inactive lineages and revive work end-to-end
- orphaned-source semantics are implemented and verified

## Milestone 4. Retrieval, Scoring, And Agent Context Assembly

Goal:
Deliver the full read-path contract for live recall, layered profile assembly, candidate search, and multi-space ranking.

Why this milestone matters:
This is the milestone where the system becomes useful to agents in real execution flows.

Backlog items:

- `BL-026` Specify Recall, Profile, And Candidate Search Contracts
- `BL-027` Implement Recall Over Live Corpus
- `BL-028` Implement Scoring Engine
- `BL-029` Implement GetProfile Layered Retrieval
- `BL-030` Implement FindCandidates
- `BL-031` Verify Retrieval, Scoring, Profile, And Candidate Search

Primary capabilities unlocked:

- `CAP-12`
- `CAP-13`
- `CAP-14`
- `CAP-15`

Primary UAT coverage target:

- `UAT-14`
- `UAT-15`
- `UAT-16`
- `UAT-17`
- `UAT-18`

Exit criteria:

- `Recall` returns only live knowledge
- full scoring formula is implemented
- `GetProfile` delivers layered context
- `FindCandidates` remains distinct from retrieval

## Milestone 5. Inspection, Auditability, And Concurrency Safety

Goal:
Make the system explainable and safe under concurrent writers.

Why this milestone matters:
Without this milestone, the system may be useful, but it is still not complete as a trustworthy memory primitive.

Backlog items:

- `BL-035` Specify Inspection And Audit Contracts
- `BL-036` Implement Audit Log And Forget Record Storage
- `BL-037` Implement MemoryInspector APIs
- `BL-038` Verify Inspection And Audit Explainability
- `BL-039` Specify Concurrency And Optimistic Lock Semantics
- `BL-040` Implement Optimistic Locking And Conflict Detection
- `BL-041` Verify Concurrent Write And Read Consistency

Primary capabilities unlocked:

- `CAP-19`
- `CAP-20`
- `CAP-21`
- `CAP-22`

Primary UAT coverage target:

- `UAT-22`
- `UAT-23`
- `UAT-24`
- `UAT-25`
- `UAT-26`

Exit criteria:

- full lineage and relation inspection are available
- audit trail explains lifecycle operations
- optimistic locking prevents conflicting latest corruption
- reads remain snapshot-consistent under writes

## Milestone 6. Embedding Migration And Operational Readiness

Goal:
Support safe embedding-model migration and complete the embedded operational model.

Why this milestone matters:
This milestone closes the loop on long-term maintainability and operator trust.

Backlog items:

- `BL-042` Specify Embedding Migration Workflow
- `BL-043` Implement Space Migration And Write Locking
- `BL-044` Verify Migration Safety And Recovery
- `BL-045` Specify Operational Config And Disk Layout
- `BL-046` Implement Config Loading And Operator-Facing Disk Layout
- `BL-047` Verify Config Application And Backup/Restore Flow

Primary capabilities unlocked:

- `CAP-23`
- `CAP-24`

Primary UAT coverage target:

- `UAT-27`
- `UAT-28`

Exit criteria:

- migration is safe and restartable
- writes are blocked during migration while reads stay available
- backup and restore flow is verified
- config behavior is operationally stable

## Milestone 7. Release Closure

Goal:
Run the release-level acceptance suite and produce `v1` sign-off.

Why this milestone matters:
This is where the system proves not just that pieces exist, but that the full product contract holds from an acceptance perspective.

Backlog items:

- `BL-048` Build UAT Harness And Traceability Matrix
- `BL-049` Execute Release-Blocking UAT Suite

Primary capabilities validated:

- `CAP-01`
- `CAP-06`
- `CAP-08`
- `CAP-10`
- `CAP-11`
- `CAP-12`
- `CAP-13`
- `CAP-16`
- `CAP-17`
- `CAP-18`
- `CAP-21`
- `CAP-22`
- `CAP-23`

Primary UAT coverage target:

- release-blocking scenarios from [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)

Exit criteria:

- release-blocking UAT scenarios pass
- traceability from backlog to UAT is complete
- `v1` release decision can be made from evidence rather than inference

## Critical Path Summary

The minimum dependency chain to reach `v1` release closure is:

1. Milestone 1
2. Milestone 2
3. Milestone 3
4. Milestone 4
5. Milestone 5
6. Milestone 6
7. Milestone 7

Milestone 0 is already the planning baseline.

## Suggested Working Rhythm

Recommended execution rhythm:

1. complete all `spec` tasks for a milestone first
2. implement the milestone's `impl` tasks
3. complete the milestone's `verify` tasks before moving on
4. run milestone-level acceptance smoke checks before opening the next wave

This keeps the system coherent at every stage and avoids accumulating unverified semantic debt.

## What To Do Next

The next practical planning step is to convert the current milestone plan into execution-ready work units.

Recommended next artifacts:

1. sprint plan for Milestone 1
2. ticket templates with traceability fields
3. definition of done per task type: `spec`, `impl`, `verify`
