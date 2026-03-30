# OmnethDB V1 Sprint Plan: Milestone 6

This document turns Milestone 6 from [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md) into an execution-ready sprint plan.

Milestone 6 focus:

- embedding migration
- config completion
- disk layout stability
- operator-facing readiness

Use this together with:

- [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
- [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
- [SPRINT_PLAN_M5.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M5.md)

## Sprint Goal

Make OmnethDB operationally safe for long-lived use by supporting embedding migration and completing the embedded runtime model.

At the end of this sprint, the system should support:

- explicit full-space embedding migration
- write-locking during migration with reads still available
- restart-safe migration behavior
- config-driven operational behavior
- stable data and config layout for local operation and backup

## In Scope

Backlog items:

- `BL-042` Specify Embedding Migration Workflow
- `BL-043` Implement Space Migration And Write Locking
- `BL-044` Verify Migration Safety And Recovery
- `BL-045` Specify Operational Config And Disk Layout
- `BL-046` Implement Config Loading And Operator-Facing Disk Layout
- `BL-047` Verify Config Application And Backup/Restore Flow

Primary milestone capabilities:

- `CAP-23` Migrate A Space To A New Embedding Model Safely
- `CAP-24` Operate As A Single-File Embedded Store
- `CAP-02` Apply Per-Space Runtime Behavior

Primary UAT alignment:

- `UAT-02`
- `UAT-03`
- `UAT-27`
- `UAT-28`

## Out Of Scope

The following remain outside Milestone 6:

- release-level UAT harness and final sign-off
- post-`v1` benchmarking or performance tuning
- new capabilities beyond the architecture-defined `v1`

## Workstreams

### Workstream A. Embedding Migration

Objective:
Support safe, explicit transition of a space from one embedder identity to another without mixing vector spaces.

Tasks:

- `BL-042`
- `BL-043`
- `BL-044`

Expected outputs:

- migration lifecycle contract
- space write-lock state during migration
- full re-embedding of historical and live memories
- recovery-safe migration verification

### Workstream B. Operational Config And Layout

Objective:
Complete the operator-facing embedded runtime model with stable config handling and disk layout expectations.

Tasks:

- `BL-045`
- `BL-046`
- `BL-047`

Expected outputs:

- config loading for decay, limits, weights, and embedder settings
- stable `data/` and config layout
- backup and restore validation

## Suggested Execution Order

Recommended order inside the sprint:

1. `BL-042`
2. `BL-045`
3. `BL-043`
4. `BL-046`
5. `BL-044`
6. `BL-047`

Rationale:

- migration and operational layout should both be specified before implementation
- migration implementation depends on a clear lock and recovery model
- config and disk layout should be verified with real restore-oriented validation

## Suggested Ticket Breakdown

If this sprint is turned into tickets, a good minimum split is:

1. migration lifecycle and write-lock semantics
2. re-embedding implementation for full corpus
3. migration safety and crash-recovery verification
4. config loading and per-space operational settings
5. embedded disk layout and backup/restore validation

## Definition Of Done

Milestone 6 sprint is done only when all of the following are true:

1. A populated space can be migrated to a new embedder through an explicit workflow.
2. Writes are rejected during migration while reads remain available.
3. Migration re-embeds both live and historical memories.
4. Interrupted migration can be rerun safely.
5. Runtime config controls documented operational behavior such as limits, decay, and weights.
6. The on-disk embedded layout is stable and validated through backup/restore flow.
7. Operator-facing behavior around data location and config location is documented and test-backed.

## Sprint Risks

### Risk 1. Partial Migration State Escaping Into Runtime

Why it matters:
If migration can expose mixed old/new embeddings, retrieval correctness is compromised in ways that are hard to detect.

Mitigation:

- enforce explicit write-lock semantics
- verify all-or-nothing migration visibility at the space level

### Risk 2. Treating Historical Embeddings As Optional During Migration

Why it matters:
The architecture requires migration of all records, not only the live corpus.

Mitigation:

- include historical records in migration implementation and tests
- validate corpus-wide completion before releasing the write lock

### Risk 3. Config Behavior Drifting From Earlier Milestones

Why it matters:
If config loading is added late without validating prior assumptions, behavior can silently diverge from earlier specs.

Mitigation:

- cross-check config implementation against earlier sprint assumptions
- use scenario-based verification instead of only unit-level parsing tests

### Risk 4. Backup/Restore Validation Remaining Too Theoretical

Why it matters:
Operator readiness depends on practical recoverability, not just documented file locations.

Mitigation:

- run restore-oriented verification using actual persisted state
- validate post-restore retrieval and config behavior

## Review Checklist

Before closing the sprint, review:

- can a space ever accept writes with mixed embedder identities?
- does migration preserve read availability end-to-end?
- are historical embeddings definitely reprocessed?
- do config values actually change runtime behavior in observable ways?
- can persisted data be restored into a clean environment and still behave correctly?

## Next Step After This Sprint

If Milestone 6 closes successfully, the next planning artifact should be:

- `SPRINT_PLAN_M7.md`

That sprint should focus on release-level UAT execution and final `v1` sign-off.
