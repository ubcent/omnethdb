# OmnethDB Advisory Curation APIs

This document defines two post-`v1` advisory APIs for curator-facing memory curation:

- `SynthesisCandidates`
- `PromotionSuggestions`

These APIs extend the memory-quality and inspection work planned in [SPRINT_PLAN_M8.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M8.md) and [SPRINT_PLAN_M9.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M9.md).

They must remain aligned with [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md): explicit lineage remains the only source of semantic structure, promotion remains a governed write, and hot-path retrieval remains predictable.

## Purpose

OmnethDB already supports:

- explicit write semantics
- explicit relation semantics
- governed promotion
- live retrieval over the current corpus
- cold-path inspection and diagnostics

What is still missing is a disciplined way to surface higher-order curation opportunities for memory curators without turning similarity or recurrence into hidden product semantics.

This document defines that surface.

## Problem

Two curator problems appear naturally once a live corpus accumulates enough episodic memory:

1. multiple episodic observations may collectively justify a synthesis review
2. an individual episodic memory may appear durable enough to justify promotion review

Without an explicit contract, these workflows tend to emerge as fuzzy inspector heuristics or implicit write-path logic. That would create semantic drift away from the architecture.

## Design Constraints

The APIs in this document must obey all of the following constraints:

- advisory only
- cold-path only
- read-only
- no write side effects
- no retrieval side effects
- no implicit lineage creation
- no implicit promotion
- no truth claims from similarity, recurrence, or score alone

This is the key boundary:

- these APIs can say "worth review"
- they cannot say "is true"
- they cannot say "must be linked"
- they cannot say "should be promoted automatically"

## Non-Goals

This document does not define:

- automatic synthesis into `KindDerived`
- automatic promotion into `KindStatic`
- automatic `Updates`, `Extends`, or `Derives` creation
- retrieval ranking changes driven by diagnostics
- human workflow UX beyond the contract needed by inspector, CLI, HTTP, or MCP surfaces

## Shared Contract Rules

The following rules apply to both `SynthesisCandidates` and `PromotionSuggestions`.

### 1. Scope Defaults

Default input scope is the live latest corpus only.

That means:

- `IsLatest=true`
- not forgotten
- no historical lineage members unless explicitly requested

### 2. Retrieval Isolation

Calling either API must not affect:

- `Recall`
- `GetProfile`
- `FindCandidates`

These APIs are diagnostics and review aids, not read-path modifiers.

### 3. Write Isolation

Calling either API must not create or mutate:

- memories
- relations
- audit records
- latest-state transitions
- promotion state

### 4. Evidence Over Opaque Ranking

The output should favor:

- explicit reason codes
- visible evidence fields
- review-oriented explanations

over one opaque global score.

### 5. Live Corpus Discipline

If forgotten or historical memories are useful for investigation, they must be included only via explicit request flags. They must not silently dominate default outputs.

## API 1. `SynthesisCandidates`

### Purpose

`SynthesisCandidates` returns clusters of live `KindEpisodic` memories that appear strong enough to justify curator review for possible synthesis.

This API does not claim:

- that the memories are duplicates
- that they must be linked by a relation
- that they should be written as a new memory automatically

It only claims that the cluster may be worth review for synthesis.

### Semantic Boundary

The unit of review is a cluster, not a single memory.

Likely downstream curator actions might include:

- authoring a new `KindDerived` memory with explicit sources and rationale
- deciding that the cluster should remain separate episodic observations
- deciding that one member should update another
- deciding that no action is warranted

The API itself makes none of those decisions.

### Default Input

Default corpus for clustering:

- `KindEpisodic` only
- `IsLatest=true`
- not forgotten

`KindStatic` and `KindDerived` are excluded by default because this API is intended to surface synthesis opportunities over observations, not to cluster already-curated durable knowledge.

### Request Shape

```go
type SynthesisCandidatesRequest struct {
    Spaces            []string
    Limit             int
    MinClusterSize    int
    MinPairSimilarity float64

    IncludeForgotten  bool
    IncludeHistorical bool

    Since             *time.Time
    Until             *time.Time

    ActorIDs          []string
    SubjectHints      []string
}
```

### Response Shape

```go
type SynthesisCandidatesResponse struct {
    Candidates []SynthesisCandidate
}

type SynthesisCandidate struct {
    CandidateID       string
    AdvisoryOnly      bool
    ReasonCodes       []string

    ClusterSize       int
    TimeSpanStart     time.Time
    TimeSpanEnd       time.Time
    DistinctActors    int
    DistinctSpaces    int

    MeanSimilarity    float64
    MinSimilarity     float64
    MaxSimilarity     float64

    SuggestedAction   string
    // review_for_derived | review_for_static | review_only

    ReviewScore       float64
    // score for "worth curator review", not score for truth

    Members           []SynthesisCandidateMember
}

type SynthesisCandidateMember struct {
    MemoryID          string
    Space             string
    ActorID           string
    CreatedAt         time.Time
    Content           string
    Similarity        float64
    IsLatest          bool
    IsForgotten       bool
}
```

### Reason Codes

Recommended reason codes include:

- `high_similarity_cluster`
- `repeated_observation`
- `cross_actor_confirmation`
- `stable_across_time`
- `mixed_fact_blob`

### Output Rules

- `AdvisoryOnly` must always be `true`
- `SuggestedAction` is triage guidance, not an instruction
- semantically noisy clusters should degrade to `review_only`
- cluster membership does not imply relation truth
- cluster similarity does not imply duplicate truth

## API 2. `PromotionSuggestions`

### Purpose

`PromotionSuggestions` returns live `KindEpisodic` memories or episodic lineages that appear durable enough to justify curator review for governed promotion into `KindStatic`.

This API does not promote anything. It only surfaces candidates for review.

### Semantic Boundary

The unit of review is a single episodic memory or lineage, not a cluster.

Likely downstream curator actions might include:

- governed promotion to `KindStatic`
- authoring an `Updates` relation instead of promoting
- deciding the memory is still purely episodic
- deciding the memory is noisy, contradictory, or blob-like and should not be promoted

The API itself makes none of those decisions.

### Default Input

Default suggestion corpus:

- `KindEpisodic` only
- `IsLatest=true`
- not forgotten

### Request Shape

```go
type PromotionSuggestionsRequest struct {
    Spaces                 []string
    Limit                  int

    IncludeForgotten       bool
    IncludeHistorical      bool

    MinCumulativeScore     float64
    MinObservationCount    int
    MinDistinctTimeWindows int
    MinDistinctActors      int

    Since                  *time.Time
    Until                  *time.Time
}
```

### Response Shape

```go
type PromotionSuggestionsResponse struct {
    Suggestions []PromotionSuggestion
}

type PromotionSuggestion struct {
    MemoryID            string
    AdvisoryOnly        bool
    ReasonCodes         []string

    ObservationCount    int
    DistinctTimeWindows int
    DistinctActors      int
    CumulativeScore     float64
    LatestScore         float64

    FirstSeenAt         time.Time
    LastSeenAt          time.Time

    ChurnRisk           float64
    ContradictionRisk   float64

    SuggestedAction     string
    // review_for_promotion | review_for_update | review_only

    Explanation         string
    Memory              PromotionSuggestionMemory
}

type PromotionSuggestionMemory struct {
    MemoryID            string
    Content             string
    Space               string
    ActorID             string
    CreatedAt           time.Time
    IsLatest            bool
    IsForgotten         bool
}
```

### Reason Codes

Recommended reason codes include:

- `high_cumulative_score`
- `repeated_occurrence`
- `stable_across_time`
- `cross_actor_confirmation`
- `high_churn`
- `possible_contradiction`
- `mixed_fact_blob`

### Output Rules

- `AdvisoryOnly` must always be `true`
- recurrence is evidence of review-worthiness, not evidence of truth
- high churn or contradiction should downgrade or suppress promotion review
- this API must never bypass `PromotePolicy`
- `SuggestedAction` must remain review-oriented rather than imperative

## Separation Between APIs

These APIs must remain separate.

`SynthesisCandidates`:

- reviews a cluster
- is about possible synthesis
- may lead a curator to author a new derived or static memory explicitly

`PromotionSuggestions`:

- reviews a single episodic memory or lineage
- is about possible governed promotion
- may lead a curator to promote the memory explicitly

Merging them into one fuzzy surface would blur two different curator decisions and make the system harder to reason about.

## Guidance On Scoring Signals

The system may use heuristics and ranking signals internally, but the contract should expose them as review evidence rather than semantic authority.

Useful `SynthesisCandidates` signals:

- pairwise similarity within the cluster
- cluster size
- actor diversity
- time spread
- anti-blob penalties

Useful `PromotionSuggestions` signals:

- cumulative retrieval score
- recurrence across time windows
- actor diversity
- stability across time
- contradiction or churn penalties
- anti-blob penalties

The contract should avoid a design where a single undocumented score silently determines outcomes.

## Examples

### Example 1. Similar Observations Worth Synthesis Review

Three episodic memories describe recurring cache invalidation failures across different runs and actors.

Good output:

- one `SynthesisCandidate`
- reason codes such as `high_similarity_cluster` and `cross_actor_confirmation`
- `SuggestedAction=review_for_derived`

Bad output:

- automatic `KindDerived` creation
- implicit `Derives` relation
- claim that the synthesis is true

### Example 2. Recurrent Episodic Memory Worth Promotion Review

One episodic observation about a stable repository policy is repeatedly retrieved across multiple windows and reinforced by multiple actors.

Good output:

- one `PromotionSuggestion`
- reason codes such as `high_cumulative_score` and `stable_across_time`
- `SuggestedAction=review_for_promotion`

Bad output:

- in-place mutation from episodic to static
- promotion without governed authorization
- claim that recurrence alone proves durable truth

### Example 3. Noisy Cluster

A set of high-similarity memories mixes multiple facts, unrelated context, and unstable observations.

Good output:

- weak or downgraded review outcome
- `mixed_fact_blob`
- `SuggestedAction=review_only`

Bad output:

- strong synthesis recommendation with no caution

### Example 4. Churny Promotion Candidate

An episodic memory appears often but alternates with contradicting observations across time.

Good output:

- `PromotionSuggestion` either downgraded or absent
- reason codes such as `high_churn` or `possible_contradiction`

Bad output:

- strong promotion suggestion based only on frequency

## Acceptance Criteria

This contract is complete only if all of the following are true:

1. `SynthesisCandidates` and `PromotionSuggestions` are defined as separate advisory APIs with distinct units of review.
2. Both APIs default to live latest memories only.
3. Forgotten and historical memories appear only by explicit request.
4. The contract explicitly forbids retrieval-side effects.
5. The contract explicitly forbids write-side effects.
6. Similarity, recurrence, and cumulative score are defined as review signals rather than truth claims.
7. The contract is specific enough for inspector, CLI, HTTP, and MCP surfaces to share one behavior model.
8. The contract is specific enough for implementation and verification tickets to test directly.

## Why This Fits OmnethDB

This design preserves the repo's core discipline:

- explicit lineage remains primary
- governed writes remain the only path to durable truth
- retrieval remains predictable
- curation stays explainable
- embeddings assist judgment without replacing semantics

That is the correct standard for OmnethDB.
