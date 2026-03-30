# OmnethDB V1 Sprint Plan: Milestone 2

This document turns Milestone 2 from [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md) into an execution-ready sprint plan.

Milestone 2 focus:

- lineage and versioning semantics
- relation validation
- governance and trust enforcement
- main `Remember` write path

Use this together with:

- [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
- [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
- [SPRINT_PLAN_M1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M1.md)

## Sprint Goal

Turn the storage foundation into a semantically correct write system.

At the end of this sprint, OmnethDB should no longer behave like generic record persistence. It should behave like governed, versioned memory with:

- one current truth per lineage
- valid explicit relations
- enforced writer policy
- a transactional `Remember` path

## In Scope

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

Primary milestone capabilities:

- `CAP-03` Write A New Root Memory
- `CAP-04` Update Existing Knowledge Without Corrupting History
- `CAP-05` Add Non-Contradictory Context
- `CAP-07` Promote Episodic Knowledge Into Durable Static Knowledge
- `CAP-16` Enforce Who Can Write Which Kind Of Knowledge
- `CAP-17` Rank Memories Using Current Trust Policy
- `CAP-18` Enforce Corpus Growth Limits

Primary UAT alignment:

- `UAT-04`
- `UAT-05`
- `UAT-06`
- `UAT-09`
- `UAT-19`
- `UAT-20`
- `UAT-21`

## Out Of Scope

The following remain outside Milestone 2:

- `KindDerived` semantics
- `Forget` and `Revive`
- retrieval scoring and profile assembly
- audit APIs
- migration workflow

## Workstreams

### Workstream A. Lineage And Relations

Objective:
Establish the structural rules that make versioned memory coherent.

Tasks:

- `BL-010`
- `BL-011`
- `BL-012`

Expected outputs:

- lineage transition rules for root and update cases
- validated `Updates` and `Extends` semantics
- invariant tests for latest exclusivity, no cycles, and relation locality

### Workstream B. Governance And Trust

Objective:
Make write behavior policy-driven instead of permissive by default.

Tasks:

- `BL-013`
- `BL-014`
- `BL-015`

Expected outputs:

- enforced writer policies by kind
- promotion gating rules
- trust lookup contract ready for later retrieval
- live corpus limit checks

### Workstream C. Remember Write Path

Objective:
Assemble the domain and policy rules into one transactional write API.

Tasks:

- `BL-016`
- `BL-017`
- `BL-018`

Expected outputs:

- concrete `Remember` API contract
- end-to-end root and update write flows
- transactional failure semantics with no partial state

## Suggested Execution Order

Recommended order inside the sprint:

1. `BL-010`
2. `BL-013`
3. `BL-016`
4. `BL-011`
5. `BL-014`
6. `BL-017`
7. `BL-012`
8. `BL-015`
9. `BL-018`

Rationale:

- lineage and governance rules should be frozen before wiring the write path
- `Remember` should be specified after those rules are clear
- verification closes each semantic area before the sprint is called done

## Suggested Ticket Breakdown

If this sprint is turned into tickets, a good minimum split is:

1. lineage model and latest-switch invariants
2. relation validation rules for `Updates` and `Extends`
3. writer policy and promotion authorization
4. trust and corpus-limit enforcement
5. `Remember` API and transaction flow
6. invariant and atomicity verification suite

## Definition Of Done

Milestone 2 sprint is done only when all of the following are true:

1. Root writes and updates behave according to lineage rules.
2. `Updates` can supersede current knowledge atomically.
3. `Extends` can link contextual knowledge without changing target state.
4. Unauthorized writes are rejected according to policy.
5. Promotion requires explicit governed write behavior.
6. Live corpus limits are enforced.
7. `Remember` commits successful writes fully and rejects invalid writes without partial state.

## Sprint Risks

### Risk 1. Mixing Policy Semantics Into Storage Primitives

Why it matters:
If authorization and governance rules leak into low-level persistence helpers, later changes become brittle.

Mitigation:

- keep policy evaluation in domain or service layers
- keep storage primitives focused on transactional persistence

### Risk 2. Under-Specifying Update Edge Cases

Why it matters:
The write path can look correct in happy-path tests while still allowing invalid latest transitions.

Mitigation:

- make negative-case tests first-class
- explicitly test stale targets, non-latest updates, and relation locality failures

### Risk 3. Making Remember Too Smart Too Early

Why it matters:
If `Remember` absorbs future lifecycle behaviors now, Milestone 3 will become harder to reason about.

Mitigation:

- implement only root, update, and promotion semantics in this sprint
- keep derived and revive logic for their own milestone

## Review Checklist

Before closing the sprint, review:

- does each lineage always have at most one latest memory?
- can an invalid update leave any stale `latest/` or `spaces/` state behind?
- do policy failures happen before mutation?
- is trust resolved from policy rather than persisted onto memory records?
- is `Remember` still small enough to extend safely in Milestone 3?

## Next Step After This Sprint

If Milestone 2 closes successfully, the next planning artifact should be:

- `SPRINT_PLAN_M3.md`

That sprint should focus on derived memories, forgetting, inactive lineages, revive, and orphaned-source propagation.
