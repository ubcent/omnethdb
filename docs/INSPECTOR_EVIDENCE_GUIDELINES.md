# OmnethDB Inspector Evidence Guidelines

This document defines how supporting evidence should be exposed in inspector surfaces for duplicate and update diagnostics.

It extends:

- [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
- [ADVISORY_CURATION_APIS.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ADVISORY_CURATION_APIS.md)
- [ADVISORY_EVIDENCE_CONTRACTS.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ADVISORY_EVIDENCE_CONTRACTS.md)
- [SPRINT_PLAN_M10.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M10.md)

This is not a visual design spec.

It defines the behavioral contract the inspector should satisfy when exposing evidence panes for curator review.

## Purpose

The inspector already exists to make OmnethDB's cold path explainable.

For duplicate and update diagnostics, explainability is not complete if the inspector only shows:

- a score
- a reason code
- a ranked list of candidates

Curators need to see the supporting context behind a surfaced candidate so they can decide whether:

- review is warranted
- the support is weak or noisy
- the candidate is merely related rather than duplicative
- the candidate looks like a possible update target rather than a true successor

The inspector evidence pane exists to provide that review support without turning diagnostics into hidden semantics.

## Design Constraints

The inspector evidence pane must obey all of the following constraints:

- cold-path only
- read-only
- no write side effects
- no retrieval side effects
- no hidden lineage creation
- no automatic cleanup behavior
- no semantic overclaim

This is the key boundary:

- the pane can explain why a candidate surfaced
- the pane cannot decide what relation, if any, should exist
- the pane cannot imply that review output is a product truth

## Non-Goals

This document does not define:

- inspector visual styling
- exact component hierarchy
- editing workflows
- bulk cleanup workflows
- automatic acceptance or rejection of candidates

## Pane Responsibilities

The inspector evidence pane should help a curator answer five questions quickly:

1. Why did this candidate surface?
2. Which memories provide the supporting context?
3. Is the support current, forgotten, or widened-scope?
4. Is the support strong, weak, noisy, or ambiguous?
5. What should the curator review next without assuming an automatic action?

If the pane cannot answer those questions, it is not yet doing its job.

## Shared Pane Rules

### 1. Evidence Must Be Specific

The pane should show explicit supporting material tied to explicit memory IDs.

Acceptable support includes:

- short snippets or excerpts
- memory content previews
- reason codes
- actor, time, and lineage context
- latest and forgotten visibility
- downgrade explanations

### 2. Evidence Must Stay Compact

The pane should not dump entire memory bodies or long historical trails by default.

Good inspector evidence is:

- focused
- review-oriented
- easy to scan

Deeper inspection can remain available through drill-down, but the default pane should stay compact.

### 3. Widened Scope Must Be Visible

If evidence includes:

- forgotten memories
- historical lineage members
- non-default scope

the pane must label that clearly.

Widened scope should never look indistinguishable from live default support.

### 4. Downgrades Must Be Explicit

If support is weak, ambiguous, noisy, blob-like, or contradiction-like, the pane should show that explicitly.

A curator should never need to infer downgrade state indirectly from a weak score alone.

### 5. Pane Output Must Stay Advisory

The pane may suggest review framing such as:

- possible duplicate review
- possible update review
- review only

It must not imply:

- automatic duplicate truth
- automatic `Updates` truth
- automatic cleanup
- automatic relation creation

## Duplicate Diagnostics Guidance

For duplicate diagnostics, the pane should expose:

- supporting snippets from each surfaced memory
- similarity summary
- reason codes
- latest and forgotten visibility
- actor and time context where it helps curator judgment
- downgrade reasons for mixed, weak, or ambiguous support

The pane should help the curator distinguish between:

- likely duplicate
- near-duplicate with meaningful difference
- related memory that should remain separate
- noisy or blob-like false positive

The pane should avoid framing that implies:

- the memories are definitely duplicates
- one memory should automatically replace another

## Update-Candidate Diagnostics Guidance

For update diagnostics, the pane should expose:

- source memory snippet
- candidate memory snippet
- explicit lineage context when available
- reason codes
- change-support or similarity summary
- actor and time context
- downgrade reasons when the candidate appears merely related

The pane should help the curator distinguish between:

- plausible successor worth review
- extension-like related memory
- duplicate-like restatement
- unrelated but semantically nearby memory

The pane should avoid framing that implies:

- an `Updates` relation already exists
- an `Updates` relation should be created automatically

## Suggested Pane Fields

The exact layout can vary, but the pane should cover fields like:

```go
type InspectorEvidencePane struct {
    CandidateType        string
    // duplicate_review | update_review

    AdvisoryOnly         bool
    ReasonCodes          []string
    DowngradeReasons     []string

    PrimaryMemory        InspectorEvidenceMemory
    RelatedMemories      []InspectorEvidenceMemory

    SimilaritySummary    *InspectorSimilaritySummary
    Scope                InspectorEvidenceScope
    ReviewHint           string
}

type InspectorEvidenceMemory struct {
    MemoryID             string
    ContentSnippet       string
    ActorID              string
    CreatedAt            time.Time
    IsLatest             bool
    IsForgotten          bool
    LineageRole          string
    // source | candidate | related | historical_support
}

type InspectorSimilaritySummary struct {
    Mean                 float64
    Min                  float64
    Max                  float64
}

type InspectorEvidenceScope struct {
    LiveOnly             bool
    IncludeHistorical    bool
    IncludeForgotten     bool
}
```

This shape is illustrative. The contract requirement is behavioral, not UI-specific.

## Review Hints

Review hints should stay non-imperative.

Good review hints:

- `possible_duplicate_review`
- `possible_update_review`
- `review_only`

Bad review hints:

- `merge_now`
- `promote_now`
- `create_update_relation`

## Examples

### Example 1. Useful Duplicate Evidence Pane

Good pane behavior:

- shows short snippets from each live latest memory
- shows similarity summary and reason codes
- shows `mixed_fact_blob` when support is noisy
- keeps the framing at review level

Bad pane behavior:

- shows only one rank score
- implies automatic deduplication

### Example 2. Useful Update Evidence Pane

Good pane behavior:

- shows the current memory and candidate side by side
- includes created-at and actor context
- labels when a candidate is historical or widened-scope
- downgrades merely related support

Bad pane behavior:

- labels a candidate as the next version without explicit lineage

### Example 3. Weak Support Pane

Good pane behavior:

- visible downgrade reason such as `ambiguous_subject`
- snippets that reveal why the support is weak
- `review_only` framing

Bad pane behavior:

- low score with no explanation
- strong duplicate or update framing despite weak support

## Acceptance Criteria

This guideline is complete only if all of the following are true:

1. The inspector evidence pane is defined as a read-only cold-path review aid.
2. The guideline requires explicit supporting snippets or equivalent focused context tied to memory IDs.
3. The guideline requires visible latest, forgotten, actor, time, and widened-scope context where relevant.
4. The guideline requires visible downgrade explanations for weak, noisy, or ambiguous support.
5. The guideline explicitly forbids semantic overclaim and automatic relation or cleanup behavior.

## Why This Fits OmnethDB

This design preserves the repo's core discipline:

- inspection remains explainable
- retrieval remains separate
- semantic structure remains explicit
- curator judgment is improved without hidden automation

That is the correct standard for OmnethDB.
