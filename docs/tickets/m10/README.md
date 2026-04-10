# Milestone 10 Tickets

These tickets define the first post-`M9` follow-up queue for evidence-centered curation and evaluation.

## Related Context

- [SPRINT_PLAN_M10.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M10.md)
- [ADVISORY_CURATION_APIS.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ADVISORY_CURATION_APIS.md)
- [ADVISORY_EVIDENCE_CONTRACTS.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ADVISORY_EVIDENCE_CONTRACTS.md)
- [INSPECTOR_EVIDENCE_GUIDELINES.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/INSPECTOR_EVIDENCE_GUIDELINES.md)
- [M10_BENCHMARK_GUIDELINES.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/M10_BENCHMARK_GUIDELINES.md)

## Scope

Milestone 10 focuses on:

- evidence-backed benchmark coverage for memory quality and review signals
- explicit evidence contracts for advisory curation surfaces
- inspector-facing supporting evidence for duplicate and update diagnostics
- verification that richer evidence remains separate from retrieval and governance semantics

## Ticket Order

1. [M10-001.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m10/M10-001.md)
2. [M10-003.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m10/M10-003.md)
3. [M10-002.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m10/M10-002.md)
4. [M10-004.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m10/M10-004.md)

## Ticket Summaries

### `M10-001` Build Evidence-Backed Memory Quality Benchmark Pack

Creates a repeatable benchmark harness and fixture set for duplicate, update, synthesis-review, and false-positive scenarios.

### `M10-003` Specify Explicit Evidence Contract For Advisory APIs

Defines a stable evidence vocabulary and downgrade explanation model for curator-facing advisory surfaces.

### `M10-002` Add Inspector Evidence Pane For Duplicate And Update Diagnostics

Exposes supporting context so curators can understand why duplicate and update candidates surfaced.

### `M10-004` Verify Rich Evidence Surfaces Do Not Leak Into Retrieval Or Governance Semantics

Proves that enriched diagnostics remain advisory-only, read-only, and cold-path only.

## Milestone Intent

This queue should preserve the architecture's core discipline:

- explicit lineage remains the only source of semantic structure
- retrieval remains predictable and separate from inspection
- evidence improves reviewability without becoming semantic authority
- curator workflows become more explainable and auditable

## Current State

- Milestone 10 is in progress.
- `M10-001` is todo.
- `M10-002` is todo.
- `M10-003` is done.
- `M10-004` is todo.

## Exit Condition

Milestone 10 is complete when OmnethDB can expose richer supporting evidence for curator review, evaluate those review signals repeatably, and prove that the enriched surfaces do not alter retrieval, lineage, or governed write behavior.
