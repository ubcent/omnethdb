# Milestone 9 Tickets

These tickets define the first post-`M8` follow-up queue for explicit memory curation workflows built on top of the memory-quality and inspector foundations.

## Scope

Milestone 9 focuses on:

- advisory curation APIs for higher-order curator review
- explicit separation between synthesis review and promotion review
- verification that new curation surfaces do not weaken retrieval or governance semantics

## Ticket Order

1. [M9-001.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m9/M9-001.md)
2. [M9-002.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m9/M9-002.md)
3. [M9-003.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m9/M9-003.md)

## Ticket Summaries

### `M9-001` Specify Advisory Curation APIs For Synthesis And Promotion Review

Defines the contract for two separate cold-path advisory APIs:

- `SynthesisCandidates` for cluster-level synthesis review
- `PromotionSuggestions` for single-memory promotion review

The purpose is to improve curator visibility without introducing hidden semantics.

### `M9-002` Implement Advisory Curation APIs For Synthesis And Promotion Review

Builds the advisory APIs on top of existing live-corpus filtering, similarity tooling, and quality diagnostics so the system can surface explicit curator-review candidates.

### `M9-003` Verify Advisory Curation APIs Do Not Leak Into Retrieval Or Governance Semantics

Proves the new APIs remain advisory-only and do not alter retrieval outputs, lineage state, or governed promotion behavior.

## Milestone Intent

This queue should preserve the architecture's core discipline:

- explicit lineage remains the only source of semantic structure
- promotion remains a governed write
- retrieval remains predictable and separate from inspection
- embeddings assist curation rather than infer truth

## Exit Condition

Milestone 9 is complete when OmnethDB can surface synthesis-review and promotion-review candidates explicitly, and the verification layer demonstrates those signals do not create hidden write or retrieval behavior.
