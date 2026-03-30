# OmnethDB V1 Sprint Plan: Milestone 7

This document turns Milestone 7 from [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md) into an execution-ready sprint plan.

Milestone 7 focus:

- release-level UAT harness
- traceability closure
- execution of release-blocking acceptance scenarios
- final `v1` sign-off readiness

Use this together with:

- [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
- [CAPABILITY_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/CAPABILITY_MAP_V1.md)
- [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md)
- [SPRINT_PLAN_M6.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M6.md)

## Sprint Goal

Prove that OmnethDB `v1` satisfies its externally meaningful contract through release-grade acceptance evidence.

At the end of this sprint, the team should be able to decide on a `v1` release based on:

- completed release-blocking UAT scenarios
- explicit traceability from architecture to acceptance
- documented pass/fail evidence
- a clear list of residual risks, if any remain

## In Scope

Backlog items:

- `BL-048` Build UAT Harness And Traceability Matrix
- `BL-049` Execute Release-Blocking UAT Suite

Primary milestone capabilities validated:

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

Primary UAT alignment:

- `UAT-01`
- `UAT-05`
- `UAT-07`
- `UAT-10`
- `UAT-12`
- `UAT-13`
- `UAT-14`
- `UAT-15`
- `UAT-16`
- `UAT-19`
- `UAT-20`
- `UAT-21`
- `UAT-24`
- `UAT-25`
- `UAT-27`

## Out Of Scope

The following are outside Milestone 7:

- adding new `v1` scope
- redesigning architecture contracts
- performance tuning beyond acceptance-critical fixes
- new roadmap planning for `v1.x` or `v2`

## Workstreams

### Workstream A. UAT Harness And Traceability

Objective:
Make release acceptance executable and auditable rather than informal.

Tasks:

- `BL-048`

Expected outputs:

- runnable UAT harness or orchestration layer
- fixture and scenario mapping for release-blocking cases
- traceability matrix from backlog items to UAT scenarios
- evidence collection format for pass/fail outcomes

### Workstream B. Release-Blocking UAT Execution

Objective:
Run the release gate and determine whether `v1` is ready to ship.

Tasks:

- `BL-049`

Expected outputs:

- executed release-blocking UAT suite
- pass/fail results per scenario
- defect or risk log for any failing or partially satisfied cases
- explicit release recommendation

## Suggested Execution Order

Recommended order inside the sprint:

1. `BL-048`
2. `BL-049`

Rationale:

- release execution should happen only after traceability and harness readiness exist
- acceptance evidence is stronger when scenarios are run through a stable, repeatable harness

## Suggested Ticket Breakdown

If this sprint is turned into tickets, a good minimum split is:

1. traceability matrix and release evidence format
2. release-blocking UAT harness and fixtures
3. release-blocking scenario execution
4. release recommendation and residual risk review

## Definition Of Done

Milestone 7 sprint is done only when all of the following are true:

1. Release-blocking UAT scenarios are runnable through a defined harness or repeatable execution process.
2. Each release-blocking scenario has a recorded pass/fail outcome.
3. Traceability from backlog item to UAT scenario is complete for release-critical work.
4. Failures, waivers, or residual risks are documented explicitly rather than implied.
5. The team can make a `v1` release decision from evidence.

## Sprint Risks

### Risk 1. Treating UAT As A Demo Instead Of A Gate

Why it matters:
If UAT is only used to showcase happy paths, release confidence will be false.

Mitigation:

- run the release-blocking set as a decision gate
- record failures and partial passes explicitly

### Risk 2. Traceability Gaps Hiding Unverified Scope

Why it matters:
Unmapped backlog work can create the illusion of completeness without actual acceptance coverage.

Mitigation:

- require explicit traceability for release-critical tasks
- review missing mappings before sign-off

### Risk 3. Weak Evidence Collection

Why it matters:
A release decision based on memory or ad hoc notes will not stand up when regressions appear later.

Mitigation:

- standardize pass/fail recording
- capture scenario-level evidence and residual risks in one place

### Risk 4. Collapsing Release Closure Into Last-Minute Bug Triage

Why it matters:
If the sprint becomes a catch-all for unfinished implementation, acceptance loses meaning.

Mitigation:

- keep Milestone 7 focused on validation and sign-off
- push late-surfacing scope changes back into backlog rather than silently absorbing them here

## Review Checklist

Before closing the sprint, review:

- are all release-blocking UAT scenarios actually executed?
- does every release-critical capability have evidence behind it?
- are any failing or flaky scenarios being hidden behind informal sign-off?
- is the release recommendation explicit: ship, do not ship, or ship with named caveats?
- is the residual risk posture clear enough for downstream planning?

## Exit Criteria

Milestone 7 exits when the team has:

1. completed the release-blocking UAT run,
2. reviewed pass/fail evidence,
3. documented residual risks and open defects,
4. made an explicit `v1` release recommendation.

## What Comes Next

After Milestone 7, the planning stack for `v1` is complete and the project can move into one of two paths:

1. execute the `v1` release, or
2. create a post-acceptance backlog for failed scenarios, deferred items, and `v1.x` follow-up work.
