# OmnethDB V1 Sprint Plan: Milestone 3

This document turns Milestone 3 from [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md) into an execution-ready sprint plan.

Milestone 3 focus:

- `KindDerived` semantics and provenance
- `Forget` and inactive lineage behavior
- `Revive`
- orphaned-source propagation

Use this together with:

- [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
- [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
- [SPRINT_PLAN_M2.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M2.md)

## Sprint Goal

Complete the full lifecycle semantics of the write side so OmnethDB can represent not only current knowledge, but also synthesized knowledge, retired knowledge, and explicit reactivation.

At the end of this sprint, the system should support:

- derived memories with auditable provenance
- forgetting without deletion
- inactive lineages with no current truth
- explicit lineage revival
- source-forget propagation into derived provenance state

## In Scope

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

Primary milestone capabilities:

- `CAP-06` Create A Derived Memory From Multiple Sources
- `CAP-08` Forget Live Knowledge Without Deleting History
- `CAP-09` Deactivate A Lineage By Forgetting Its Latest Version
- `CAP-10` Revive An Inactive Lineage
- `CAP-11` Propagate Forgotten Sources Into Derived Provenance State

Primary UAT alignment:

- `UAT-07`
- `UAT-08`
- `UAT-10`
- `UAT-11`
- `UAT-12`
- `UAT-13`

## Out Of Scope

The following remain outside Milestone 3:

- full retrieval scoring behavior
- `GetProfile` and `FindCandidates`
- inspector APIs and audit querying
- concurrency workflow
- embedder migration

## Workstreams

### Workstream A. Derived Memory

Objective:
Add synthesis as a first-class memory behavior with explicit and enforceable provenance rules.

Tasks:

- `BL-019`
- `BL-020`
- `BL-021`

Expected outputs:

- derived memory validation rules
- source provenance persistence
- rationale enforcement
- negative-case coverage for invalid derived writes

### Workstream B. Forget And Revive Lifecycle

Objective:
Represent knowledge retirement and explicit reactivation without breaking lineage history.

Tasks:

- `BL-022`
- `BL-023`
- `BL-024`
- `BL-025`

Expected outputs:

- `Forget` semantics for latest and non-latest records
- inactive lineage handling
- `Revive` flow with kind preservation and parent linkage
- lifecycle tests for forget and revive paths

### Workstream C. Orphaned Sources

Objective:
Preserve derived memories when evidence changes, while surfacing provenance degradation explicitly.

Tasks:

- `BL-032`
- `BL-033`
- `BL-034`

Expected outputs:

- atomic source-forget propagation
- permanent orphaned-source flag behavior
- retrieval-ready filtering semantics for later milestones

## Suggested Execution Order

Recommended order inside the sprint:

1. `BL-019`
2. `BL-022`
3. `BL-032`
4. `BL-020`
5. `BL-023`
6. `BL-024`
7. `BL-033`
8. `BL-021`
9. `BL-025`
10. `BL-034`

Rationale:

- define derived and lifecycle contracts before wiring persistence
- specify orphan propagation before implementing source-forget side effects
- complete verification after each behavior family is implemented

## Suggested Ticket Breakdown

If this sprint is turned into tickets, a good minimum split is:

1. derived validation and source eligibility
2. derived provenance persistence and relation linkage
3. forget semantics and inactive lineage handling
4. revive semantics and lifecycle tests
5. orphaned-source propagation
6. lifecycle and derived verification suite

## Definition Of Done

Milestone 3 sprint is done only when all of the following are true:

1. Derived memories require valid current non-derived sources and rationale.
2. Invalid derived writes are rejected without partial state.
3. `Forget` removes knowledge from the live corpus without deleting history.
4. Forgetting the latest version deactivates the lineage without restoring older versions.
5. `Revive` reactivates only inactive lineages and preserves lineage kind.
6. Forgetting a source marks dependent live derived memories with `HasOrphanedSources=true`.
7. Orphaned-source state is permanent and externally observable for later retrieval filtering.

## Sprint Risks

### Risk 1. Derived Semantics Becoming Too Permissive

Why it matters:
If source validation is weak, the system will allow compounding inference errors that violate the architecture.

Mitigation:

- validate source count, source kind, and source latest-state explicitly
- treat invalid provenance as hard rejection, not warning behavior

### Risk 2. Forget Breaking Lineage Invariants

Why it matters:
Incorrect forget handling can accidentally reactivate stale knowledge or leave inconsistent live state behind.

Mitigation:

- test latest-forget and non-latest-forget separately
- verify `spaces/` and `latest/` state after each lifecycle transition

### Risk 3. Revive Smuggling Cross-Kind Evolution

Why it matters:
If revive can change lineage kind, it bypasses the governance model.

Mitigation:

- enforce root-kind matching in revive validation
- keep cross-kind evolution deferred to explicit promotion or new-root paths

### Risk 4. Orphan Propagation Becoming Retrieval Logic Too Early

Why it matters:
This milestone should mark provenance state, not prematurely entangle full retrieval implementation.

Mitigation:

- focus on write-time marking semantics
- keep retrieval filtering contract minimal and ready for Milestone 4

## Review Checklist

Before closing the sprint, review:

- can a derived memory ever be created from stale or derived sources?
- does forgetting latest always produce an inactive lineage rather than a silent rollback?
- can revive ever succeed on an active lineage?
- are orphaned derived memories still present as live records after source forgetting?
- are lifecycle transitions visible in persisted state without requiring retrieval implementation?

## Next Step After This Sprint

If Milestone 3 closes successfully, the next planning artifact should be:

- `SPRINT_PLAN_M4.md`

That sprint should focus on `Recall`, scoring, `GetProfile`, `FindCandidates`, and multi-space retrieval behavior.
