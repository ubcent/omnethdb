# OmnethDB Post-V1 Sprint Plan: Milestone 10

This document defines the follow-up execution sprint after Milestone 9.

Milestone 10 focus:

- evidence-backed evaluation for memory quality and curation signals
- richer evidence surfaces for inspector and advisory review outputs
- stable evidence contracts shared across diagnostics transports
- verification that richer evidence remains isolated from retrieval and governance semantics

Use this together with:

- [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
- [INDEX.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/INDEX.md)
- [SPRINT_PLAN_M8.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M8.md)
- [SPRINT_PLAN_M9.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M9.md)
- [ADVISORY_CURATION_APIS.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ADVISORY_CURATION_APIS.md)
- [ADVISORY_EVIDENCE_CONTRACTS.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ADVISORY_EVIDENCE_CONTRACTS.md)
- [INSPECTOR_EVIDENCE_GUIDELINES.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/INSPECTOR_EVIDENCE_GUIDELINES.md)
- [M10_BENCHMARK_GUIDELINES.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/M10_BENCHMARK_GUIDELINES.md)
- [docs/tickets/m10/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m10/README.md)

## Sprint Goal

Strengthen OmnethDB's curator-facing and evaluation-facing evidence model so memory quality work becomes more explainable, more inspectable, and more measurable without weakening the architecture.

At the end of this sprint, the system should support:

- repeatable benchmark coverage for duplicate, update, and synthesis-review quality
- explicit supporting evidence in inspector and advisory review surfaces
- stable evidence fields and downgrade explanations across transports
- verification that richer evidence remains cold-path only

## In Scope

Primary follow-up slices:

- evidence-backed quality benchmark pack
- inspector evidence pane for duplicate and update diagnostics
- explicit evidence contract for advisory curation APIs
- semantic-isolation verification for rich evidence surfaces

Architecture alignment:

- preserve explicit lineage as the only source of semantic structure
- keep hot-path retrieval predictable and unchanged
- keep curator evidence explainable rather than opaque
- use evidence and embeddings to support review, not to infer truth

## Out Of Scope

The following remain outside Milestone 10:

- automatic synthesis into `Derived` or `Static`
- automatic promotion into governed durable knowledge
- hidden relation creation from evidence or similarity
- retrieval ranking changes driven by evidence panes or advisory metadata
- broad end-user workflow UX beyond inspector, CLI, HTTP, MCP, or shared store contracts

## Workstreams

### Workstream A. Evidence-Backed Benchmarking

Objective:

Make memory-quality and curator-review signals measurable on scenarios that resemble real agent memory mistakes and review opportunities.

Tasks:

- `M10-001`

Expected outputs:

- benchmark fixtures for duplicate, update, and synthesis-review scenarios
- repeatable harness with mode-separated reporting
- visible false-positive and downgrade coverage

### Workstream B. Shared Evidence Contracts

Objective:

Define a stable evidence vocabulary so diagnostics and advisory review surfaces expose why a candidate surfaced rather than only how strongly it ranked.

Tasks:

- `M10-003`

Expected outputs:

- explicit evidence fields for advisory review responses
- downgrade explanation vocabulary
- shared behavior model for inspector, CLI, HTTP, and MCP surfaces

Current status:

- `M10-003` specification is complete in [ADVISORY_EVIDENCE_CONTRACTS.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ADVISORY_EVIDENCE_CONTRACTS.md)

### Workstream C. Evidence Surfaces In Inspector

Objective:

Help memory curators inspect duplicate and update diagnostics with direct supporting context instead of opaque signals.

Tasks:

- `M10-002`

Expected outputs:

- supporting snippet or excerpt panes
- visible actor, time, and lineage evidence
- clearer curator judgment during duplicate or update review

### Workstream D. Semantic Isolation Verification

Objective:

Prove that richer evidence surfaces do not leak into retrieval, lineage, audit, or governed write semantics.

Tasks:

- `M10-004`

Expected outputs:

- regression coverage for retrieval isolation
- non-mutation verification for lineage, relations, and audit state
- transport-parity checks for enriched evidence outputs

## Suggested Execution Order

Recommended order inside the sprint:

1. `M10-001`
2. `M10-003`
3. `M10-002`
4. `M10-004`

Rationale:

- evidence-backed evaluation should exist before richer evidence contracts depend on it
- the shared contract should freeze expectations before inspector exposure expands
- verification should prove that the richer surfaces remain outside product semantics

## Suggested Ticket Breakdown

If this sprint is turned into tickets, a good minimum split is:

1. build evidence-backed benchmark fixtures and reporting
2. specify shared evidence contract for advisory review outputs
3. expose supporting evidence in inspector diagnostics
4. verify semantic isolation for enriched evidence surfaces

## Definition Of Done

Milestone 10 sprint is done only when all of the following are true:

1. Memory-quality evaluation covers duplicate, update, synthesis-review, and false-positive scenarios through a repeatable harness.
2. Advisory and inspector surfaces expose explicit supporting evidence rather than only opaque ranking output.
3. Evidence fields and downgrade reasons are stable enough to share across transport layers.
4. Richer evidence remains advisory and cold-path only.
5. Tests demonstrate that enriched evidence does not change `Recall`, `GetProfile`, lineage state, relation state, audit behavior, or governed promotion semantics.

## Sprint Risks

### Risk 1. Evidence Surfaces Start Acting Like Semantic Authority

Why it matters:

If supporting evidence begins to imply truth or lineage automatically, OmnethDB will drift away from explicit governed memory.

Mitigation:

- keep evidence outputs advisory only
- require explicit wording that evidence supports review, not truth

### Risk 2. Benchmarking Collapses Into Generic Similarity Scores

Why it matters:

If the benchmark no longer reflects real memory-quality mistakes and curator review cases, it will create false confidence.

Mitigation:

- use fixtures drawn from realistic agent memory patterns
- report false positives and downgrade behavior, not only headline hit rate

### Risk 3. Inspector Evidence Becomes Noisy Or Unusable

Why it matters:

If supporting context is verbose or weakly selected, it will reduce curator trust instead of improving it.

Mitigation:

- keep evidence focused and review-oriented
- expose downgrade reasons when the supporting context is weak or ambiguous

### Risk 4. Rich Evidence Leaks Into Hot-Path Retrieval

Why it matters:

If retrieval starts depending on diagnostics evidence, results become less predictable and less auditable.

Mitigation:

- verify no `Recall` or `GetProfile` behavior changes
- treat semantic-isolation tests as release-blocking for this milestone

## Review Checklist

Before closing the sprint, review:

- do benchmark fixtures reflect real memory-quality and curator-review scenarios?
- can a curator see why a candidate surfaced without overclaiming semantic certainty?
- are downgrade reasons visible when evidence is ambiguous, noisy, or contradiction-like?
- do enriched surfaces remain clearly separate from retrieval and governed writes?
- do verification tests prove non-mutation across store and transport layers?

## Exit Criteria

Milestone 10 exits when the team has:

1. shipped evidence-backed benchmark coverage for quality and review signals,
2. defined and exposed stable evidence fields across diagnostics surfaces,
3. improved inspector reviewability with supporting evidence,
4. verified that enriched evidence remains outside retrieval and governance semantics.

## What Comes Next

After Milestone 10, the next planning move should likely be one of:

1. explicit curator action workflows built on top of reviewed evidence surfaces, or
2. stronger policy and operating packs for long-running mixed human and agent stewardship.
