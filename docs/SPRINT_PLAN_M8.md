# OmnethDB Post-V1 Sprint Plan: Milestone 8

This document defines the first post-`v1` execution sprint after release closure.

Milestone 8 focus:

- memory quality control under real agent usage
- embedding-assisted write hygiene
- operator-visible diagnostics for duplicate and churn risk
- safer config application for long-lived spaces

Use this together with:

- [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
- [INDEX.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/INDEX.md)
- [SPRINT_PLAN_M7.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M7.md)
- [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md)
- [RELEASE_RECOMMENDATION_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_RECOMMENDATION_V1.md)

## Sprint Goal

Make OmnethDB resilient to real agent memory load by improving write quality, operator control, and explainable diagnostics without weakening the architecture.

At the end of this sprint, the system should support:

- similarity-assisted linting before or during memory writes
- explicit signals for duplicate and update candidates
- repeatable embedding-quality evaluation for memory hygiene use cases
- inspector diagnostics for duplicate clusters and churny subjects
- operator commands to validate and apply runtime config safely to persisted spaces

## In Scope

Primary follow-up slices:

- similarity lint for `remember`
- embedding eval harness for duplicate and update detection
- inspector quality diagnostics
- config apply/diff/validate workflow

Architecture alignment:

- preserve explicit lineage and governed writes
- keep hot-path retrieval predictable
- use embeddings to assist quality control, not to infer truth

## Out Of Scope

The following remain outside Milestone 8:

- replacing explicit lineage with similarity-based inference
- hidden graph traversal in hot-path retrieval
- auto-promotion of agent memories into durable truth
- major new transport layers or product surfaces unrelated to memory quality
- broad `v2` roadmap planning

## Workstreams

### Workstream A. Similarity-Assisted Write Hygiene

Objective:
Reduce duplicate, blob-like, and misclassified writes before they degrade the live corpus.

Tasks:

- `M8-001`
- `M8-002`

Expected outputs:

- lint contract for candidate memory writes
- similarity search against live corpus for duplicate and update hints
- warning-oriented integration into `remember` and MCP-facing flows

### Workstream B. Quality Diagnostics And Evaluation

Objective:
Make memory quality measurable for both operators and automated regression checks.

Tasks:

- `M8-003`
- `M8-004`

Expected outputs:

- embedding eval dataset and benchmark harness
- duplicate/update metrics
- inspector panels for duplicate clusters and candidate update targets

### Workstream C. Operator Config Reconciliation

Objective:
Remove operator friction around runtime config changes so governance can evolve without destructive resets.

Tasks:

- `M8-005`

Expected outputs:

- config diff command
- config validation command
- config apply command for persisted spaces

## Suggested Execution Order

Recommended order inside the sprint:

1. `M8-001`
2. `M8-005`
3. `M8-002`
4. `M8-003`
5. `M8-004`

Rationale:

- the lint contract should be explicit before wiring it into write flows
- config reconciliation solves operator pain already exposed by real usage
- evaluation should exist before inspector diagnostics depend on similarity signals

## Suggested Ticket Breakdown

If this sprint is turned into tickets, a good minimum split is:

1. specify write-lint contract and warning semantics
2. implement similarity lint on the write path
3. build embedding eval suite for duplicate and update detection
4. add duplicate/update diagnostics to inspector
5. implement config validate/diff/apply workflow

## Definition Of Done

Milestone 8 sprint is done only when all of the following are true:

1. A new memory candidate can be linted against the live corpus for duplicate and update risk.
2. Similarity signals are advisory and do not create hidden lineage or truth claims.
3. Embedding-quality evaluation is runnable and produces repeatable metrics.
4. Operators can inspect likely duplicate clusters and update candidates in the inspector.
5. Persisted spaces can reconcile runtime config changes through explicit validation and apply flows.
6. Tests demonstrate that memory quality tooling does not leak historical or graph behavior into hot-path retrieval.

## Sprint Risks

### Risk 1. Embedding Similarity Starts Driving Semantics

Why it matters:
If similarity begins to imply lineage or truth, OmnethDB drifts away from explicit, governed memory.

Mitigation:

- keep similarity outputs advisory only
- prohibit automatic relation creation from similarity alone

### Risk 2. Linting Becomes A Hidden Write Blocker

Why it matters:
Silent rejection or opaque behavior would make agent integrations harder to trust.

Mitigation:

- start with warnings and explicit suggestion codes
- make blocking policy an explicit later decision, not an incidental side effect

### Risk 3. Quality Metrics Become Toy Benchmarks

Why it matters:
If evals are not tied to real duplicate/update scenarios, they will not improve real memory quality.

Mitigation:

- use realistic memory-pair fixtures
- include false-positive scenarios, not only happy-path matches

### Risk 4. Config Reconciliation Remains Destructive

Why it matters:
If operators still need to recreate spaces or databases to apply config changes, governance will remain brittle.

Mitigation:

- implement explicit config diff/apply commands
- verify persisted config changes against existing populated spaces

## Review Checklist

Before closing the sprint, review:

- does similarity lint only advise, or is it secretly changing behavior?
- can operators distinguish duplicate hints from true lineage?
- do eval metrics catch both duplicate misses and false positives?
- does inspector quality output help clean up memory, or only create noise?
- can config changes be reconciled safely without wiping persisted state?

## Exit Criteria

Milestone 8 exits when the team has:

1. shipped advisory similarity linting for writes,
2. added measurable embedding-quality evaluation,
3. exposed useful quality diagnostics in the inspector,
4. removed destructive config-reconciliation workflows for normal operator changes.

## What Comes Next

After Milestone 8, the next planning move should likely be one of:

1. promotion and curation workflows for mixed human/agent memory, or
2. domain-specific operating packs for coding, support, and consultant-style agents.
