# OmnethDB Advisory Evidence Contracts

This document defines the shared evidence contract for curator-facing advisory and diagnostic surfaces introduced after the core advisory curation APIs.

It extends:

- [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
- [ADVISORY_CURATION_APIS.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ADVISORY_CURATION_APIS.md)
- [SPRINT_PLAN_M10.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M10.md)

It exists to answer a narrower question than the advisory API contract:

- not "what review surfaces exist?"
- but "what evidence must those surfaces expose, and how must that evidence behave?"

## Purpose

OmnethDB already defines curator-facing advisory surfaces such as:

- `SynthesisCandidates`
- `PromotionSuggestions`
- duplicate diagnostics
- update-candidate diagnostics

Those surfaces are only trustworthy if they explain why a candidate surfaced.

Without an explicit evidence contract, the system risks drifting into:

- opaque scores with unclear meaning
- transport-specific evidence shapes
- inspector-only heuristics that do not match API behavior
- evidence fields that overclaim truth, lineage, or promotion authority

This document defines the shared evidence discipline that prevents that drift.

## Design Constraints

The evidence model in this document must obey all of the following constraints:

- advisory only
- cold-path only
- read-only
- no retrieval side effects
- no write side effects
- no implicit lineage creation
- no implicit promotion authority
- evidence supports review-worthiness, not truth

This is the key boundary:

- evidence can say "here is why this candidate may be worth review"
- evidence cannot say "this memory is true"
- evidence cannot say "these memories are definitely duplicates"
- evidence cannot say "this memory should be promoted automatically"

## Non-Goals

This document does not define:

- new advisory APIs
- retrieval ranking changes
- write-path behavior changes
- inspector-specific visual layout
- benchmark methodology

## Shared Evidence Rules

The following rules apply to all evidence-bearing diagnostics and advisory review outputs.

### 1. Evidence Must Be Visible

A candidate must expose explicit supporting evidence rather than only a global score.

Acceptable evidence includes:

- supporting snippets or excerpts
- visible reason codes
- actor spread
- time spread
- similarity summary
- recurrence summary
- downgrade reasons

### 2. Evidence Must Stay Review-Oriented

Evidence must explain why a candidate surfaced for review.

It must not overclaim:

- truth
- contradiction certainty
- duplicate certainty
- lineage certainty
- promotion certainty

### 3. Evidence Must Preserve Live-Corpus Discipline

By default, supporting evidence should come from:

- live latest memories
- non-forgotten memories

Historical or forgotten support may be included only by explicit request or clearly labeled widened scope.

### 4. Evidence Must Be Transport-Stable

Inspector, CLI, HTTP, MCP, and gRPC surfaces should share one evidence vocabulary even if their formatting differs.

The goal is behavioral parity, not UI identity.

### 5. Evidence Must Prefer Explanation Over Opaque Ranking

Internal scoring signals may exist, but the contract should expose:

- why the score is strong or weak
- what kind of support exists
- what kind of ambiguity remains

One undocumented score must never be the only explanation.

### 6. Evidence Must Support Downgrade Semantics

When support is weak, noisy, blob-like, ambiguous, or contradiction-like, the output should degrade explicitly.

That downgrade must be visible in the result shape rather than hidden in a lower score alone.

## Shared Evidence Shape

The exact transport types may differ, but the shared model should cover the following concepts.

```go
type AdvisoryEvidence struct {
    ReasonCodes       []string
    DowngradeReasons  []string

    Snippets          []EvidenceSnippet
    Similarity        SimilarityEvidence
    Recurrence        RecurrenceEvidence
    ActorSpread       ActorSpreadEvidence
    TimeSpread        TimeSpreadEvidence
    Scope             EvidenceScope
}

type EvidenceSnippet struct {
    MemoryID          string
    Space             string
    ContentSnippet    string
    CreatedAt         time.Time
    ActorID           string
    IsLatest          bool
    IsForgotten       bool
    RelevanceLabel    string
    // supporting_match | contrasting_match | cluster_member | update_target
}

type SimilarityEvidence struct {
    Mean              float64
    Min               float64
    Max               float64
    PairCount         int
}

type RecurrenceEvidence struct {
    ObservationCount    int
    DistinctTimeWindows int
    Summary             string
}

type ActorSpreadEvidence struct {
    DistinctActors      int
    Summary             string
}

type TimeSpreadEvidence struct {
    FirstSeenAt         time.Time
    LastSeenAt          time.Time
    Summary             string
}

type EvidenceScope struct {
    LiveOnly            bool
    IncludeHistorical   bool
    IncludeForgotten    bool
}
```

This shape is illustrative, not mandatory field-for-field. The contract requirement is semantic parity across transports.

## Evidence Concepts

### Snippets

Snippets should provide compact supporting context tied to explicit memory IDs.

Snippet rules:

- keep them short and review-oriented
- include enough text to justify the surfaced candidate
- preserve source memory identity
- label widened-scope or forgotten support visibly

Snippets should help a curator answer:

- what specifically matched?
- which memory provided that support?
- is this support current or widened-scope?

### Similarity Summary

Similarity evidence is useful for:

- duplicate diagnostics
- update-target review
- synthesis clustering

Similarity is a review signal only.

It must not imply:

- duplicate truth
- relation truth
- lineage truth

### Recurrence Summary

Recurrence evidence is useful for:

- promotion review
- repeated-observation review
- synthesis review

Recurrence is evidence of review-worthiness, not evidence of durable truth.

### Actor Spread

Actor diversity is useful because repeated support from different actors can increase review-worthiness.

It must not be presented as automatic confirmation of truth.

### Time Spread

Time spread is useful because stable support across time often matters for curator judgment.

It must not be presented as sufficient proof that a memory should be promoted or linked.

## Downgrade Reason Vocabulary

Recommended downgrade reasons include:

- `weak_support`
- `ambiguous_subject`
- `mixed_fact_blob`
- `possible_contradiction`
- `high_churn`
- `narrow_actor_support`
- `narrow_time_support`
- `historical_only_support`
- `forgotten_support_only`

These reasons are not truth verdicts.

They explain why the surfaced candidate should be treated more cautiously.

## Surface-Specific Guidance

### `SynthesisCandidates`

Should expose:

- cluster-level snippets
- pairwise or aggregate similarity summary
- actor spread
- time spread
- downgrade reasons when clusters are noisy or weak

Should avoid implying:

- that members must be linked
- that a derived memory should be created automatically

### `PromotionSuggestions`

Should expose:

- supporting snippets from repeated episodic observations
- recurrence summary
- actor spread
- time spread
- downgrade reasons for churn, contradiction-like support, or weak reinforcement

Should avoid implying:

- that promotion is justified automatically
- that recurrence alone establishes durable truth

### Duplicate Diagnostics

Should expose:

- matching snippets
- similarity summary
- latest and forgotten visibility
- downgrade reasons when support is weak or mixed

Should avoid implying:

- duplicate certainty
- automatic cleanup

### Update-Candidate Diagnostics

Should expose:

- source and candidate snippets
- similarity or change-support summary
- lineage context
- downgrade reasons when the candidate appears merely related

Should avoid implying:

- automatic `Updates` creation
- semantic certainty that one memory supersedes another

## Output Rules

All evidence-bearing outputs should obey the following rules:

- evidence fields must remain visible in every transport, even if summarized
- widened-scope support must be labeled explicitly
- downgrade reasons must be visible when present
- evidence should help a curator decide whether to review, not decide on the curator's behalf

## Examples

### Example 1. Strong But Advisory Duplicate Diagnostic

Good output:

- matching snippets from two latest memories
- high similarity summary
- no downgrade reasons
- wording that says "possible duplicate review"

Bad output:

- only one score with no visible basis
- wording that says the memories are definitely duplicates

### Example 2. Promotion Suggestion With Weak Reinforcement

Good output:

- recurrence summary with narrow actor spread
- downgrade reason `narrow_actor_support`
- wording that says review may still be useful

Bad output:

- strong promotion framing despite weak support diversity

### Example 3. Synthesis Candidate With Contradiction-Like Support

Good output:

- surfaced or downgraded candidate
- downgrade reason `possible_contradiction`
- snippets that make the ambiguity inspectable

Bad output:

- hidden downgrade logic
- strong synthesis framing with no caution

## Acceptance Criteria

This contract is complete only if all of the following are true:

1. The evidence model defines shared concepts for snippets, similarity, recurrence, actor spread, time spread, scope, and downgrade reasons.
2. The contract states that evidence is advisory review support rather than truth, lineage, or promotion authority.
3. The contract makes widened-scope evidence visible when historical or forgotten support is included.
4. The contract is specific enough for inspector, CLI, HTTP, MCP, and gRPC surfaces to share one behavior model.
5. The contract is specific enough for implementation and verification work to test directly.

## Why This Fits OmnethDB

This design preserves the repo's core discipline:

- explicit lineage remains primary
- governed writes remain the only path to durable truth
- retrieval remains predictable
- curation remains inspectable
- evidence improves reviewability without replacing semantics

That is the correct standard for OmnethDB.
