# OmnethDB M10 Benchmark Guidelines

This document defines the benchmark discipline for evidence-centered memory quality and curator-review evaluation in Milestone 10.

It extends:

- [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
- [SPRINT_PLAN_M8.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M8.md)
- [SPRINT_PLAN_M9.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M9.md)
- [SPRINT_PLAN_M10.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M10.md)
- [ADVISORY_CURATION_APIS.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ADVISORY_CURATION_APIS.md)
- [ADVISORY_EVIDENCE_CONTRACTS.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ADVISORY_EVIDENCE_CONTRACTS.md)

This document is not a model-selection policy.

It defines what the benchmark pack should measure, how fixtures should be shaped, and how results should be reported so memory-quality and curator-review work can be evaluated honestly.

## Purpose

OmnethDB already has post-`v1` workstreams for:

- duplicate and update detection
- synthesis-review clustering
- promotion-review suggestions
- supporting evidence and downgrade behavior

Those surfaces need a benchmark discipline that reflects real memory mistakes and real curator-review ambiguity.

Without an explicit benchmark guideline, the system risks drifting into:

- generic similarity benchmarks that do not match OmnethDB semantics
- headline scores that hide false positives
- evaluation sets that reward overconfident clustering
- benchmark narratives that overclaim product correctness

This document defines the evaluation standard that avoids that drift.

## Design Constraints

The benchmark pack in this document must obey all of the following constraints:

- scenario-driven rather than generic semantic search
- repeatable
- mode-separated
- evidence-aware
- false-positive visible
- advisory-semantics aligned
- retrieval-isolated

This is the key boundary:

- the benchmark can evaluate review signals and evidence quality
- the benchmark cannot silently redefine product semantics

## Non-Goals

This document does not define:

- automatic threshold tuning in production
- which embedder must be used by default
- one universal benchmark score
- benchmark-driven changes to retrieval contracts
- release-level acceptance criteria for `v1`

## Core Principles

### 1. Benchmark Real Memory Tasks

The benchmark should reflect tasks OmnethDB actually performs or advises on, such as:

- duplicate review
- update-target review
- synthesis-review clustering
- promotion-review support
- downgrade behavior under weak evidence

It should not optimize for generic sentence similarity outside those use cases.

### 2. Separate Scenario Classes

Results should be reported by scenario class rather than collapsed into one score.

At minimum, the pack should separate:

- duplicate scenarios
- update scenarios
- synthesis-review scenarios
- false-positive or downgrade scenarios

### 3. Measure Precision Risks Explicitly

False positives matter because they create curator noise and semantic drift pressure.

The benchmark must make visible:

- duplicate false positives
- update false positives
- weak synthesis clusters that should degrade
- contradiction-like or noisy cases

### 4. Respect Advisory Boundaries

Benchmark targets should reflect advisory review-worthiness rather than truth claims.

For example:

- good clustering for synthesis review is not the same as proving a synthesis is true
- good promotion suggestion support is not the same as proving durable truth

### 5. Keep Reporting Explainable

A benchmark report should help the team answer:

- what got better?
- what got worse?
- in which scenario class?
- at what false-positive cost?

Opaque aggregate reporting is not enough.

## Benchmark Scenario Classes

The benchmark pack should cover at least the following classes.

### Class A. Duplicate Review

Goal:

Evaluate whether the system surfaces candidate duplicate memories for review without overfiring on merely related memories.

Fixture types:

- true duplicates
- near-duplicates with meaningful differences
- related but non-duplicate memories
- mixed-fact blob cases

Useful measures:

- duplicate hit rate
- duplicate precision
- downgrade behavior for ambiguous pairs

### Class B. Update Review

Goal:

Evaluate whether the system surfaces plausible successor candidates without collapsing into generic semantic proximity.

Fixture types:

- true update candidates
- extension-like but non-update cases
- duplicate-like restatements
- unrelated but nearby semantic neighbors

Useful measures:

- update-target hit rate
- update precision
- false-positive rate on merely related memories

### Class C. Synthesis Review

Goal:

Evaluate whether the system surfaces clusters worth curator review for possible synthesis without treating clustering as semantic authority.

Fixture types:

- coherent repeated-observation clusters
- cross-actor reinforcement clusters
- stable-across-time clusters
- noisy or internally mixed clusters
- contradiction-like clusters

Useful measures:

- review-worthy cluster hit rate
- strong cluster precision
- downgrade behavior for noisy clusters

### Class D. Promotion-Review Support

Goal:

Evaluate whether the system surfaces durable episodic candidates for curator review with evidence that is useful rather than misleading.

Fixture types:

- repeated episodic observations with broad actor or time support
- narrow-support observations that should downgrade
- churny or contradiction-like observations
- frequently retrieved but blob-like cases

Useful measures:

- promotion-review candidate precision
- downgrade behavior under churn
- overpromotion risk visibility

### Class E. Evidence Quality

Goal:

Evaluate whether surfaced candidates include evidence that is actually useful for curator judgment.

Fixture types:

- strong evidence cases
- weak evidence cases
- widened-scope evidence cases
- contradiction-like evidence cases

Useful measures:

- snippet coverage
- downgrade visibility
- widened-scope labeling coverage
- reason-code and evidence completeness

## Fixture Design Guidelines

### 1. Use Realistic Memory Shapes

Fixtures should look like real OmnethDB memories and curator-review situations, not synthetic sentence pairs only.

Useful fixture ingredients include:

- actor diversity
- time spacing
- latest vs historical states
- forgotten states when explicitly widened
- mixed fact quality

### 2. Include Negative Fixtures Intentionally

Every strong positive class should have nearby negatives that are easy to confuse with it.

Examples:

- duplicate vs related
- update vs extension
- synthesis-worthy cluster vs noisy cluster
- promotion-worthy recurrence vs churny repetition

### 3. Include Downgrade Fixtures

The benchmark should explicitly reward graceful downgrade behavior where support is weak or ambiguous.

This is important because over-eager surfacing is often worse than cautious review-only output.

### 4. Preserve Scenario Legibility

Fixtures should be understandable by the next contributor.

The benchmark pack should not become a mysterious blob of examples with unclear rationale.

## Reporting Guidelines

Benchmark reports should be mode-separated and explicit.

Recommended reporting sections:

- duplicate review
- update review
- synthesis review
- promotion review
- false-positive and downgrade behavior
- evidence completeness

For each section, prefer reporting that surfaces:

- positives found
- false positives
- downgraded cases
- ambiguous cases
- residual weaknesses

Avoid relying only on:

- one total score
- one averaged similarity number
- one headline "improvement" statement

## Comparison Discipline

When comparing benchmark runs:

- compare like-for-like fixture sets
- keep scenario classes stable
- call out regressions even if one aggregate score improves
- treat false-positive regressions as real regressions

If a change helps one class but harms another, the report should say so plainly.

## Suggested Benchmark Shape

The exact implementation can vary, but the benchmark pack should conceptually support outputs like:

```go
type BenchmarkReport struct {
    RunID              string
    ScenarioReports    []ScenarioReport
}

type ScenarioReport struct {
    ScenarioClass      string
    TotalFixtures      int
    PositiveHits       int
    FalsePositives     int
    Downgraded         int
    Notes              []string
}
```

This shape is illustrative. The contract requirement is honest, scenario-separated reporting.

## Examples

### Example 1. Honest Duplicate Report

Good report:

- duplicate precision improved
- false positives on related memories also rose
- mixed-fact blob cases now downgrade more consistently

Bad report:

- one higher aggregate score with no explanation of increased false positives

### Example 2. Honest Synthesis Review Report

Good report:

- coherent clusters surface reliably
- contradiction-like clusters now downgrade instead of surfacing strongly
- noisy clusters remain a residual weakness

Bad report:

- "clustering improved" with no distinction between strong clusters and noisy false positives

### Example 3. Honest Evidence Report

Good report:

- snippet coverage improved
- widened-scope labeling remains incomplete
- downgrade reasons missing in a subset of ambiguous cases

Bad report:

- report focuses only on candidate hit rate and ignores evidence completeness

## Acceptance Criteria

This guideline is complete only if all of the following are true:

1. The benchmark model is defined around duplicate, update, synthesis-review, promotion-review, and evidence-quality scenario classes.
2. The guideline requires negative and downgrade fixtures rather than only positive examples.
3. The guideline requires mode-separated reporting rather than one aggregate score.
4. The guideline states that false-positive regressions are real regressions.
5. The guideline keeps benchmark interpretation aligned with advisory semantics rather than truth claims.

## Why This Fits OmnethDB

This design preserves the repo's core discipline:

- evaluation remains explainable
- similarity does not silently become semantics
- curator-review surfaces are measured honestly
- product claims stay tied to evidence rather than benchmark theater

That is the correct standard for OmnethDB.
