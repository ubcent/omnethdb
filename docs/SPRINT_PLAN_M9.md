# OmnethDB Post-V1 Sprint Plan: Milestone 9

This document defines the follow-up execution sprint after Milestone 8.

Milestone 9 focus:

- explicit curator-review workflows for higher-order memory curation
- advisory synthesis-review clusters over episodic memory
- advisory promotion-review suggestions for durable episodic knowledge
- verification that curation diagnostics remain separate from retrieval and governance semantics

Use this together with:

- [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
- [INDEX.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/INDEX.md)
- [SPRINT_PLAN_M8.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M8.md)
- [docs/tickets/m8/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m8/README.md)
- [docs/tickets/m9/README.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets/m9/README.md)

## Sprint Goal

Extend OmnethDB from memory-quality diagnostics into explicit curator-facing curation review without weakening the architecture.

At the end of this sprint, the system should support:

- cluster-level advisory outputs for episodic memories worth synthesis review
- single-memory advisory outputs for episodic memories worth promotion review
- clear separation between synthesis review and promotion review semantics
- verification that these advisory surfaces do not change retrieval, lineage, or governed write behavior

## In Scope

Primary follow-up slices:

- `SynthesisCandidates` contract and implementation
- `PromotionSuggestions` contract and implementation
- shared reason-code and evidence vocabulary for curator review
- verification of advisory-only behavior and hot-path isolation

Architecture alignment:

- preserve explicit lineage as the only source of semantic structure
- preserve governed promotion as an explicit write path
- keep hot-path retrieval predictable and unchanged
- use embeddings and scoring signals to assist review, not to infer truth

## Out Of Scope

The following remain outside Milestone 9:

- automatic synthesis into `Derived` or `Static`
- automatic promotion of episodic memories into durable truth
- hidden relation creation from similarity or recurrence
- retrieval ranking changes driven by curator-review outputs
- broad human workflow UX beyond basic diagnostics exposure

## Workstreams

### Workstream A. Advisory Curation Contracts

Objective:

Define explicit contracts for higher-order curator review surfaces before implementation begins.

Tasks:

- `M9-001`

Expected outputs:

- `SynthesisCandidates` advisory contract
- `PromotionSuggestions` advisory contract
- stable reason-code taxonomy and evidence expectations

Current status:

- `M9-001` specification is complete in [ADVISORY_CURATION_APIS.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ADVISORY_CURATION_APIS.md)

### Workstream B. Advisory Curation Implementation

Objective:

Expose explicit synthesis-review and promotion-review candidates over the live corpus without changing memory semantics.

Tasks:

- `M9-002`

Expected outputs:

- implemented cold-path advisory APIs
- live-only default behavior
- reusable behavior model for inspector, CLI, HTTP, or MCP surfaces

Current status:

- `M9-002` implementation is complete in store, HTTP, CLI, inspector, and MCP surfaces
- the implementation currently includes explicit downgrade handling for blob-like, noisy, and contradiction-like support clusters

### Workstream C. Semantic Isolation Verification

Objective:

Prove that curator-review surfaces remain separate from retrieval and governance semantics.

Tasks:

- `M9-003`

Expected outputs:

- regression coverage for `Recall` and `GetProfile`
- verification of non-mutation for relations and latest state
- noisy and contradictory fixture coverage for downgrade behavior

Current status:

- `M9-003` verification is complete
- regression coverage now covers `Recall`, `GetProfile`, live-only defaults, widened scope flags, broader ambiguous or false-positive downgrade behavior, audit non-mutation, and relation or lineage non-mutation
- transport parity now also includes gRPC exposure for `SynthesisCandidates` and `PromotionSuggestions`, verified against the shared store contract

## Suggested Execution Order

Recommended order inside the sprint:

1. `M9-001`
2. `M9-002`
3. `M9-003`

Rationale:

- the contract should freeze semantics before implementation begins
- implementation should reuse the advisory discipline established in M8
- verification must prove that the new surfaces stay outside hot-path retrieval and governed write behavior

## Suggested Ticket Breakdown

If this sprint is turned into tickets, a good minimum split is:

1. specify advisory curation APIs for synthesis and promotion review
2. implement advisory curation APIs over the live corpus
3. verify that advisory curation does not leak into retrieval or governance semantics

## Definition Of Done

Milestone 9 sprint is done only when all of the following are true:

1. Memory curators can inspect clusters of episodic memories that are worth synthesis review.
2. Memory curators can inspect individual episodic memories that are worth promotion review.
3. The system keeps synthesis review and promotion review as distinct advisory concepts.
4. Similarity, recurrence, and cumulative score remain advisory signals rather than truth or lineage claims.
5. Tests demonstrate that the new curation APIs do not change `Recall`, `GetProfile`, relation state, or governed promotion behavior.

## Sprint Risks

### Risk 1. Advisory Review Starts Acting Like Hidden Semantics

Why it matters:

If review outputs begin to imply truth, OmnethDB will drift away from explicit lineage and governed memory.

Mitigation:

- keep review outputs advisory only
- require explicit wording and reason codes that avoid semantic overclaim

### Risk 2. Synthesis Review And Promotion Review Collapse Into One Fuzzy Surface

Why it matters:

Cluster review and single-memory promotion review are different curator actions with different risks.

Mitigation:

- define separate APIs and acceptance criteria
- keep output shapes and examples distinct

### Risk 3. Curator Diagnostics Leak Into Hot-Path Retrieval

Why it matters:

If retrieval starts depending on advisory curation signals, result sets become less predictable and less auditable.

Mitigation:

- verify no `Recall` or `GetProfile` behavior changes
- treat semantic-isolation tests as release-blocking for this milestone

## Review Checklist

Before closing the sprint, review:

- do synthesis-review outputs avoid implying relation truth?
- do promotion-review outputs avoid implying automatic promotion?
- can a curator distinguish clearly between cluster review and single-memory review?
- do noisy, churny, or contradictory cases degrade to weaker review outcomes?
- do verification tests prove the new APIs remain outside hot-path retrieval and governed writes?

## Exit Criteria

Milestone 9 exits when the team has:

1. specified advisory curation contracts for synthesis and promotion review,
2. implemented explicit cold-path APIs for those review surfaces,
3. verified that the new surfaces do not alter retrieval, lineage, or promotion semantics.

Current status:

- Milestone 9 is complete
- `M9-001`, `M9-002`, and `M9-003` are complete

## What Comes Next

After Milestone 9, the next planning move should likely be one of:

1. explicit curator action workflows layered on top of advisory review outputs, or
2. stronger operating packs and policy controls for mixed human and agent memory stewardship.
