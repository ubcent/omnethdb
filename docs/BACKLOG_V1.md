# OmnethDB Full-Spec V1 Backlog

This document converts the planning stack into a spec-driven implementation backlog for `v1`.

It is designed to preserve traceability across:

- architecture
- spec slices
- capability outcomes
- UAT scenarios
- implementation work

Use this together with:

- [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
- [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
- [CAPABILITY_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/CAPABILITY_MAP_V1.md)
- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)

## Purpose

This backlog is not organized by package or file layout.

It is organized by capability slices and acceptance outcomes so that implementation stays aligned with the product contract rather than drifting into disconnected technical tasks.

## Task Format

Each task includes:

- type: `spec`, `impl`, or `verify`
- goal: what the task must accomplish
- depends on: prerequisite backlog items
- traces to spec: relevant spec slices
- traces to capability: relevant capability IDs
- traces to UAT: relevant UAT scenarios

## Epic 1. Domain Foundation

### BL-001. Define Domain Types And Error Surface

Type:
`spec`

Goal:
Freeze the canonical domain model, relation enums, policy types, config types, and public error contract for `v1`.

Depends on:
none

Traces to spec:

- Domain Model And Error Contract

Traces to capability:

- `CAP-01`
- `CAP-03`
- `CAP-16`

Traces to UAT:

- `UAT-01`
- `UAT-04`
- `UAT-19`

### BL-002. Implement Core Domain Model And Validation Helpers

Type:
`impl`

Goal:
Implement core structs, enums, validation rules, and reusable domain checks for IDs, kinds, actors, relations, and errors.

Depends on:

- `BL-001`

Traces to spec:

- Domain Model And Error Contract

Traces to capability:

- `CAP-03`
- `CAP-16`

Traces to UAT:

- `UAT-04`
- `UAT-19`

### BL-003. Verify Domain Reject Cases

Type:
`verify`

Goal:
Add contract tests for invalid IDs, invalid enum values, malformed inputs, and expected error classifications.

Depends on:

- `BL-002`

Traces to spec:

- Domain Model And Error Contract

Traces to capability:

- `CAP-03`
- `CAP-16`

Traces to UAT:

- `UAT-04`
- `UAT-19`

## Epic 2. Space Bootstrap And Configuration

### BL-004. Specify Space Bootstrap And Config Lock Semantics

Type:
`spec`

Goal:
Define the first-write lifecycle of a space, config persistence rules, embedder locking semantics, and mismatch failure behavior.

Depends on:

- `BL-001`

Traces to spec:

- Space Bootstrap And Configuration Locking

Traces to capability:

- `CAP-01`
- `CAP-02`

Traces to UAT:

- `UAT-01`
- `UAT-02`
- `UAT-03`

### BL-005. Implement Space Config Storage And First-Write Bootstrap

Type:
`impl`

Goal:
Persist per-space config on first write, including embedder model ID, dimensions, default weight, and policy.

Depends on:

- `BL-004`
- `BL-002`

Traces to spec:

- Space Bootstrap And Configuration Locking

Traces to capability:

- `CAP-01`
- `CAP-02`

Traces to UAT:

- `UAT-01`
- `UAT-02`

### BL-006. Verify Space Bootstrap, Locking, And Override Behavior

Type:
`verify`

Goal:
Prove first-write bootstrap, embedder mismatch rejection, and request-level override precedence over defaults.

Depends on:

- `BL-005`

Traces to spec:

- Space Bootstrap And Configuration Locking
- Operational Config And Disk Layout

Traces to capability:

- `CAP-01`
- `CAP-02`
- `CAP-15`

Traces to UAT:

- `UAT-01`
- `UAT-02`
- `UAT-03`

## Epic 3. Storage Architecture

### BL-007. Specify Bucket Layout And Hot/Cold Path Boundaries

Type:
`spec`

Goal:
Define the persisted layout for `spaces/`, `latest/`, `embeddings/`, `memories/`, `relations/`, `spaces_config/`, `audit_log/`, and `forget_log/`.

Depends on:

- `BL-001`
- `BL-004`

Traces to spec:

- Live Corpus Storage Layout

Traces to capability:

- `CAP-12`
- `CAP-19`
- `CAP-24`

Traces to UAT:

- `UAT-14`
- `UAT-22`
- `UAT-28`

### BL-008. Implement Core Buckets And Persistence Primitives

Type:
`impl`

Goal:
Implement the low-level persistence layer and transaction helpers for hot-path and cold-path buckets.

Depends on:

- `BL-007`

Traces to spec:

- Live Corpus Storage Layout

Traces to capability:

- `CAP-03`
- `CAP-12`
- `CAP-24`

Traces to UAT:

- `UAT-04`
- `UAT-14`
- `UAT-28`

### BL-009. Verify Storage Invariants And Persistence Boundaries

Type:
`verify`

Goal:
Verify that hot-path buckets contain only intended live data and that cold-path data remains independently inspectable.

Depends on:

- `BL-008`

Traces to spec:

- Live Corpus Storage Layout

Traces to capability:

- `CAP-12`
- `CAP-19`
- `CAP-24`

Traces to UAT:

- `UAT-14`
- `UAT-22`
- `UAT-28`

## Epic 4. Lineage And Relation Semantics

### BL-010. Specify Lineage, Versioning, And Relation Rules

Type:
`spec`

Goal:
Freeze the rules for root creation, updates, relation locality, cycle prevention, and latest-version exclusivity.

Depends on:

- `BL-001`

Traces to spec:

- Lineage And Versioning Semantics
- Relation Validation Semantics

Traces to capability:

- `CAP-04`
- `CAP-05`
- `CAP-20`

Traces to UAT:

- `UAT-05`
- `UAT-06`
- `UAT-23`

### BL-011. Implement Lineage State Transitions And Relation Validation

Type:
`impl`

Goal:
Implement root and update lineage transitions, relation validation, and invariants for `Updates` and `Extends`.

Depends on:

- `BL-010`
- `BL-008`

Traces to spec:

- Lineage And Versioning Semantics
- Relation Validation Semantics

Traces to capability:

- `CAP-04`
- `CAP-05`

Traces to UAT:

- `UAT-05`
- `UAT-06`

### BL-012. Verify Lineage And Relation Invariants

Type:
`verify`

Goal:
Add invariant tests for single-latest exclusivity, invalid targets, no cycles, no forked versioning, and relation locality.

Depends on:

- `BL-011`

Traces to spec:

- Lineage And Versioning Semantics
- Relation Validation Semantics

Traces to capability:

- `CAP-04`
- `CAP-05`

Traces to UAT:

- `UAT-05`
- `UAT-06`

## Epic 5. Governance And Trust

### BL-013. Specify Writer Policies, Promotion, Trust, And Limits

Type:
`spec`

Goal:
Freeze the rules for writer authorization, promotion, trust resolution, corpus limits, and profile limits.

Depends on:

- `BL-001`
- `BL-004`

Traces to spec:

- Write Governance And Promotion

Traces to capability:

- `CAP-07`
- `CAP-16`
- `CAP-17`
- `CAP-18`

Traces to UAT:

- `UAT-09`
- `UAT-19`
- `UAT-20`
- `UAT-21`

### BL-014. Implement Policy Enforcement And Limit Checks

Type:
`impl`

Goal:
Implement per-kind writer authorization, promotion gating, trust lookup, and live corpus limit enforcement.

Depends on:

- `BL-013`
- `BL-002`
- `BL-005`

Traces to spec:

- Write Governance And Promotion

Traces to capability:

- `CAP-07`
- `CAP-16`
- `CAP-17`
- `CAP-18`

Traces to UAT:

- `UAT-09`
- `UAT-19`
- `UAT-20`
- `UAT-21`

### BL-015. Verify Governance, Trust, And Limit Semantics

Type:
`verify`

Goal:
Verify authorized vs unauthorized writes, trust-policy ranking effects, and separation of corpus limits from profile limits.

Depends on:

- `BL-014`

Traces to spec:

- Write Governance And Promotion
- Scoring Engine

Traces to capability:

- `CAP-16`
- `CAP-17`
- `CAP-18`

Traces to UAT:

- `UAT-19`
- `UAT-20`
- `UAT-21`

## Epic 6. Remember Write Path

### BL-016. Specify Remember API Contract

Type:
`spec`

Goal:
Define the write-path contract for root writes, updates, promotion flows, error returns, and transaction boundaries.

Depends on:

- `BL-010`
- `BL-013`

Traces to spec:

- Remember Write Path

Traces to capability:

- `CAP-03`
- `CAP-04`
- `CAP-07`

Traces to UAT:

- `UAT-04`
- `UAT-05`
- `UAT-09`

### BL-017. Implement Remember End-To-End

Type:
`impl`

Goal:
Implement `Remember` with full validation, embedding generation, storage writes, latest switching, and transactional commit behavior.

Depends on:

- `BL-016`
- `BL-011`
- `BL-014`
- `BL-008`

Traces to spec:

- Remember Write Path

Traces to capability:

- `CAP-03`
- `CAP-04`
- `CAP-07`

Traces to UAT:

- `UAT-04`
- `UAT-05`
- `UAT-09`

### BL-018. Verify Remember Atomicity And Reject Behavior

Type:
`verify`

Goal:
Prove that successful writes commit fully and invalid writes leave no partial state behind.

Depends on:

- `BL-017`

Traces to spec:

- Remember Write Path

Traces to capability:

- `CAP-03`
- `CAP-04`

Traces to UAT:

- `UAT-04`
- `UAT-05`
- `UAT-08`

## Epic 7. Derived Memory

### BL-019. Specify Derived Memory And Provenance Contract

Type:
`spec`

Goal:
Freeze all rules around source counts, source eligibility, rationale, actor eligibility, and derived update semantics.

Depends on:

- `BL-010`
- `BL-013`

Traces to spec:

- Derived Memory Contract

Traces to capability:

- `CAP-06`

Traces to UAT:

- `UAT-07`
- `UAT-08`

### BL-020. Implement Derived Memory Validation And Persistence

Type:
`impl`

Goal:
Implement `KindDerived` validation, provenance persistence, and graph linkage through source relations.

Depends on:

- `BL-019`
- `BL-017`

Traces to spec:

- Derived Memory Contract

Traces to capability:

- `CAP-06`

Traces to UAT:

- `UAT-07`
- `UAT-08`

### BL-021. Verify Derived Write Validity And Failure Paths

Type:
`verify`

Goal:
Verify successful derived writes and rejection of invalid source sets, invalid actors, and missing rationale.

Depends on:

- `BL-020`

Traces to spec:

- Derived Memory Contract

Traces to capability:

- `CAP-06`
- `CAP-16`

Traces to UAT:

- `UAT-07`
- `UAT-08`

## Epic 8. Forget, Inactive Lineages, And Revive

### BL-022. Specify Forget, Inactive Lineage, And Revive Semantics

Type:
`spec`

Goal:
Freeze the lifecycle behavior of forgetting latest vs non-latest records, inactive lineage state, and explicit revive.

Depends on:

- `BL-010`
- `BL-013`

Traces to spec:

- Forget And Inactive Lineages
- Revive Contract

Traces to capability:

- `CAP-08`
- `CAP-09`
- `CAP-10`

Traces to UAT:

- `UAT-10`
- `UAT-11`
- `UAT-12`

### BL-023. Implement Forget And Inactive Lineage Handling

Type:
`impl`

Goal:
Implement `Forget`, latest removal, lineage deactivation, and audit writes for forget operations.

Depends on:

- `BL-022`
- `BL-017`
- `BL-008`

Traces to spec:

- Forget And Inactive Lineages

Traces to capability:

- `CAP-08`
- `CAP-09`

Traces to UAT:

- `UAT-10`
- `UAT-11`

### BL-024. Implement Revive For Inactive Lineages

Type:
`impl`

Goal:
Implement `Revive` with correct parent linkage, kind preservation, reactivation semantics, and audit output.

Depends on:

- `BL-022`
- `BL-023`

Traces to spec:

- Revive Contract

Traces to capability:

- `CAP-10`

Traces to UAT:

- `UAT-12`

### BL-025. Verify Forget, Inactive, And Revive Behavior

Type:
`verify`

Goal:
Verify forget visibility, inactive lineage semantics, and explicit reactivation correctness.

Depends on:

- `BL-023`
- `BL-024`

Traces to spec:

- Forget And Inactive Lineages
- Revive Contract

Traces to capability:

- `CAP-08`
- `CAP-09`
- `CAP-10`

Traces to UAT:

- `UAT-10`
- `UAT-11`
- `UAT-12`

## Epic 9. Retrieval And Scoring

### BL-026. Specify Recall, Profile, And Candidate Search Contracts

Type:
`spec`

Goal:
Freeze behavior differences between `Recall`, `GetProfile`, and `FindCandidates`, including corpus visibility and scoring rules.

Depends on:

- `BL-007`
- `BL-013`
- `BL-019`

Traces to spec:

- Recall Retrieval Contract
- Scoring Engine
- GetProfile Layered Retrieval
- FindCandidates Contract

Traces to capability:

- `CAP-12`
- `CAP-13`
- `CAP-14`
- `CAP-15`

Traces to UAT:

- `UAT-14`
- `UAT-15`
- `UAT-16`
- `UAT-17`
- `UAT-18`

### BL-027. Implement Recall Over Live Corpus

Type:
`impl`

Goal:
Implement live-corpus candidate loading, filtering, and ranked retrieval without graph traversal.

Depends on:

- `BL-026`
- `BL-008`
- `BL-014`

Traces to spec:

- Recall Retrieval Contract

Traces to capability:

- `CAP-12`
- `CAP-15`

Traces to UAT:

- `UAT-14`
- `UAT-18`

### BL-028. Implement Scoring Engine

Type:
`impl`

Goal:
Implement cosine, confidence, trust, recency, and space-weight scoring with per-request and per-space configuration support.

Depends on:

- `BL-026`
- `BL-027`

Traces to spec:

- Scoring Engine

Traces to capability:

- `CAP-12`
- `CAP-15`
- `CAP-17`

Traces to UAT:

- `UAT-15`
- `UAT-18`
- `UAT-20`

### BL-029. Implement GetProfile Layered Retrieval

Type:
`impl`

Goal:
Implement static and episodic layering, per-layer top-k behavior, and orphan filtering behavior in `GetProfile`.

Depends on:

- `BL-026`
- `BL-028`
- `BL-020`

Traces to spec:

- GetProfile Layered Retrieval

Traces to capability:

- `CAP-13`

Traces to UAT:

- `UAT-13`
- `UAT-16`

### BL-030. Implement FindCandidates

Type:
`impl`

Goal:
Implement raw-cosine-only candidate search with optional historical inclusion.

Depends on:

- `BL-026`
- `BL-008`

Traces to spec:

- FindCandidates Contract

Traces to capability:

- `CAP-14`

Traces to UAT:

- `UAT-17`

### BL-031. Verify Retrieval, Scoring, Profile, And Candidate Search

Type:
`verify`

Goal:
Verify live-corpus filtering, score ordering, profile layering, multi-space weighting, and candidate-search differences.

Depends on:

- `BL-027`
- `BL-028`
- `BL-029`
- `BL-030`

Traces to spec:

- Recall Retrieval Contract
- Scoring Engine
- GetProfile Layered Retrieval
- FindCandidates Contract

Traces to capability:

- `CAP-12`
- `CAP-13`
- `CAP-14`
- `CAP-15`
- `CAP-17`

Traces to UAT:

- `UAT-13`
- `UAT-14`
- `UAT-15`
- `UAT-16`
- `UAT-17`
- `UAT-18`
- `UAT-20`

## Epic 10. Orphaned Sources

### BL-032. Specify Orphaned Source Propagation

Type:
`spec`

Goal:
Freeze the behavior that marks live derived memories when one of their sources is forgotten and define retrieval consequences.

Depends on:

- `BL-019`
- `BL-022`
- `BL-026`

Traces to spec:

- Orphaned Source Propagation

Traces to capability:

- `CAP-11`

Traces to UAT:

- `UAT-13`

### BL-033. Implement Source-Forget Propagation To Derived Memories

Type:
`impl`

Goal:
Implement atomic setting of `HasOrphanedSources` on affected live derived memories during source forget operations.

Depends on:

- `BL-032`
- `BL-023`
- `BL-020`

Traces to spec:

- Orphaned Source Propagation

Traces to capability:

- `CAP-11`

Traces to UAT:

- `UAT-13`

### BL-034. Verify Orphaned Source Behavior

Type:
`verify`

Goal:
Verify permanent orphan marking, default inclusion in retrieval, and exclusion by caller request.

Depends on:

- `BL-033`
- `BL-029`

Traces to spec:

- Orphaned Source Propagation

Traces to capability:

- `CAP-11`
- `CAP-13`

Traces to UAT:

- `UAT-13`

## Epic 11. Inspection And Auditability

### BL-035. Specify Inspection And Audit Contracts

Type:
`spec`

Goal:
Freeze the contract for `GetLineage`, `GetRelated`, `GetAuditLog`, and indexed forget records.

Depends on:

- `BL-007`
- `BL-010`
- `BL-022`

Traces to spec:

- Inspector And Graph Traversal
- Auditability And Forget Records

Traces to capability:

- `CAP-19`
- `CAP-20`
- `CAP-21`

Traces to UAT:

- `UAT-22`
- `UAT-23`
- `UAT-24`

### BL-036. Implement Audit Log And Forget Record Storage

Type:
`impl`

Goal:
Implement append-only audit records and targeted forget-record indexing for writes, forgets, revives, and migrations.

Depends on:

- `BL-035`
- `BL-017`
- `BL-023`
- `BL-024`

Traces to spec:

- Auditability And Forget Records

Traces to capability:

- `CAP-21`

Traces to UAT:

- `UAT-10`
- `UAT-12`
- `UAT-24`

### BL-037. Implement MemoryInspector APIs

Type:
`impl`

Goal:
Implement lineage history lookup, relation traversal, and audit retrieval APIs over cold-path data.

Depends on:

- `BL-035`
- `BL-036`
- `BL-011`

Traces to spec:

- Inspector And Graph Traversal

Traces to capability:

- `CAP-19`
- `CAP-20`
- `CAP-21`

Traces to UAT:

- `UAT-22`
- `UAT-23`
- `UAT-24`

### BL-038. Verify Inspection And Audit Explainability

Type:
`verify`

Goal:
Verify that lineage, relations, and audit history together can explain live corpus state and historical change.

Depends on:

- `BL-036`
- `BL-037`

Traces to spec:

- Inspector And Graph Traversal
- Auditability And Forget Records

Traces to capability:

- `CAP-19`
- `CAP-20`
- `CAP-21`

Traces to UAT:

- `UAT-22`
- `UAT-23`
- `UAT-24`

## Epic 12. Concurrency

### BL-039. Specify Concurrency And Optimistic Lock Semantics

Type:
`spec`

Goal:
Freeze the behavior of `IfLatestID`, conflict handling, and read consistency under concurrent operations.

Depends on:

- `BL-010`
- `BL-016`

Traces to spec:

- Concurrency And Consistency

Traces to capability:

- `CAP-22`

Traces to UAT:

- `UAT-25`
- `UAT-26`

### BL-040. Implement Optimistic Locking And Conflict Detection

Type:
`impl`

Goal:
Implement `IfLatestID` checks and conflict rejection for contested latest-version updates.

Depends on:

- `BL-039`
- `BL-017`

Traces to spec:

- Concurrency And Consistency

Traces to capability:

- `CAP-22`

Traces to UAT:

- `UAT-25`

### BL-041. Verify Concurrent Write And Read Consistency

Type:
`verify`

Goal:
Verify conflict rejection, single-latest preservation, and snapshot consistency for reads during writes.

Depends on:

- `BL-040`
- `BL-027`

Traces to spec:

- Concurrency And Consistency

Traces to capability:

- `CAP-22`

Traces to UAT:

- `UAT-25`
- `UAT-26`

## Epic 13. Embedding Migration

### BL-042. Specify Embedding Migration Workflow

Type:
`spec`

Goal:
Define migration lifecycle, write-lock behavior, read availability, crash safety, and idempotent rerun behavior.

Depends on:

- `BL-004`
- `BL-007`

Traces to spec:

- Embedding Migration Workflow

Traces to capability:

- `CAP-23`

Traces to UAT:

- `UAT-02`
- `UAT-27`

### BL-043. Implement Space Migration And Write Locking

Type:
`impl`

Goal:
Implement `MigrateEmbeddings`, space write-lock state, full re-embedding of historical and live memories, and config update on completion.

Depends on:

- `BL-042`
- `BL-008`
- `BL-036`

Traces to spec:

- Embedding Migration Workflow

Traces to capability:

- `CAP-23`

Traces to UAT:

- `UAT-27`

### BL-044. Verify Migration Safety And Recovery

Type:
`verify`

Goal:
Verify write rejection during migration, read continuity, full migration completion, and safe rerun after interruption.

Depends on:

- `BL-043`

Traces to spec:

- Embedding Migration Workflow

Traces to capability:

- `CAP-23`

Traces to UAT:

- `UAT-27`

## Epic 14. Operations And Configuration

### BL-045. Specify Operational Config And Disk Layout

Type:
`spec`

Goal:
Freeze runtime config responsibilities, persisted file layout, and operator-facing backup expectations.

Depends on:

- `BL-004`
- `BL-007`
- `BL-042`

Traces to spec:

- Operational Config And Disk Layout

Traces to capability:

- `CAP-02`
- `CAP-24`

Traces to UAT:

- `UAT-03`
- `UAT-28`

### BL-046. Implement Config Loading And Operator-Facing Disk Layout

Type:
`impl`

Goal:
Implement config loading for limits, weights, and decay values, and expose a stable on-disk structure for data and config.

Depends on:

- `BL-045`
- `BL-005`
- `BL-043`

Traces to spec:

- Operational Config And Disk Layout

Traces to capability:

- `CAP-02`
- `CAP-24`

Traces to UAT:

- `UAT-03`
- `UAT-28`

### BL-047. Verify Config Application And Backup/Restore Flow

Type:
`verify`

Goal:
Verify config-driven behavior and prove the embedded backup/restore model through practical restore validation.

Depends on:

- `BL-046`

Traces to spec:

- Operational Config And Disk Layout

Traces to capability:

- `CAP-02`
- `CAP-24`

Traces to UAT:

- `UAT-03`
- `UAT-28`

## Epic 15. Release Verification

### BL-048. Build UAT Harness And Traceability Matrix

Type:
`impl`

Goal:
Create the test harness, fixtures, and mapping needed to run release-level UAT scenarios against the system.

Depends on:

- `BL-031`
- `BL-038`
- `BL-041`
- `BL-044`
- `BL-047`

Traces to spec:

- Spec Verification Suite

Traces to capability:

- `CAP-01`
- `CAP-06`
- `CAP-10`
- `CAP-12`
- `CAP-21`
- `CAP-22`
- `CAP-23`

Traces to UAT:

- `UAT-01`
- `UAT-07`
- `UAT-12`
- `UAT-14`
- `UAT-24`
- `UAT-25`
- `UAT-27`

### BL-049. Execute Release-Blocking UAT Suite

Type:
`verify`

Goal:
Execute and sign off the release-blocking UAT scenarios defined in the UAT map.

Depends on:

- `BL-048`

Traces to spec:

- Spec Verification Suite

Traces to capability:

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

Traces to UAT:

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

## Suggested Execution Order

Recommended implementation flow:

1. `BL-001` to `BL-009`
2. `BL-010` to `BL-018`
3. `BL-019` to `BL-025`
4. `BL-026` to `BL-034`
5. `BL-035` to `BL-041`
6. `BL-042` to `BL-047`
7. `BL-048` to `BL-049`

This order preserves the architectural dependency chain while still keeping UAT and release verification tied to user-visible outcomes.
