# OmnethDB Full-Spec V1 Spec Map

This document translates the architecture in [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/ARCHITECTURE.md) into capability slices for a full-spec `v1`.

It is intentionally not a task backlog yet. Its purpose is to:

- define the implementation surface of `v1`
- identify foundational slices and dependency order
- make later spec-driven task generation consistent and non-duplicative

## Scope

`v1` is treated as a reference implementation of the architectural contract, not a trimmed MVP.

Included in scope:

- full lineage and versioning semantics
- full relation semantics: `Updates`, `Extends`, `Derives`
- write-path correctness: `Remember`, `Forget`, `Revive`
- retrieval correctness: `Recall`, `GetProfile`, `FindCandidates`
- governance and policy enforcement
- auditability and inspection APIs
- embedding model locking and migration workflow
- concurrency guarantees described in the architecture

Out of scope for this document:

- detailed task breakdown
- milestone scheduling
- package/file layout
- benchmarking targets or SLAs

## Slice Format

Each slice below captures:

- the capability boundary
- why it exists
- the architecture sections it covers
- the main contracts it must prove
- whether it is foundational

## Capability Slices

### 1. Domain Model And Error Contract

Why:
Establish the canonical vocabulary, persisted entities, and reject conditions for the whole system.

Covers:

- Core Invariants
- Data Model
- Space Model
- Memory Governance
- Inactive Lineages
- API Surface

Must prove:

- canonical definitions for `Memory`, `Actor`, `MemoryRelations`, `SpaceWritePolicy`, `SpaceConfig`
- enums for `MemoryKind`, `ActorKind`, relation types
- domain error set and rejection semantics
- `SpaceID` format constraints

Type:
`foundation`

Depends on:
none

### 2. Space Bootstrap And Configuration Locking

Why:
Make first-write space creation deterministic and lock space identity around embeddings and config.

Covers:

- Space Model
- Embedding model lock
- `spaces_config/`

Must prove:

- spaces are created implicitly on first write
- first write locks `EmbeddingModelID` and dimension
- default weight and per-space config are persisted
- later writes with mismatched embedder are rejected
- cross-space writes are not possible

Type:
`foundation`

Depends on:

- Domain Model And Error Contract

### 3. Lineage And Versioning Semantics

Why:
This is the core product guarantee: exactly one current version of knowledge per lineage.

Covers:

- IsLatest exclusivity
- Updates atomicity
- Lineage immutability
- Updates semantics

Must prove:

- root creation semantics for `Version`, `ParentID`, `RootID`, `IsLatest`
- atomic latest-switch on `Updates`
- no updates against non-latest historical versions
- no forked versioning
- no lineage cycles
- no invalid cross-kind replacement where forbidden by the architecture

Type:
`foundation`

Depends on:

- Domain Model And Error Contract
- Space Bootstrap And Configuration Locking

### 4. Relation Validation Semantics

Why:
Relations must be explicitly authored and strictly validated, not inferred or partially accepted.

Covers:

- Relation locality
- Updates
- Extends
- Derives
- No implicit inference

Must prove:

- both endpoints always belong to the same space
- relation-specific validation for `Updates`, `Extends`, `Derives`
- self-relations and invalid relation shapes are rejected
- no automatic relation inference exists in the system

Type:
`foundation`

Depends on:

- Domain Model And Error Contract
- Lineage And Versioning Semantics

### 5. Derived Memory Contract

Why:
`KindDerived` is one of the most distinctive and risk-heavy parts of the full `v1` contract.

Covers:

- Derives requires sources
- Derived evolution
- derived provenance rules

Must prove:

- `KindDerived` requires at least two distinct source IDs
- all sources must be latest at write time
- derived memories cannot use derived memories as sources
- `Rationale` is mandatory and non-empty
- system actors cannot create derived memories
- derived memories may be updated only by fresh derived memories with fresh source sets and rationale

Type:
`foundation`

Depends on:

- Relation Validation Semantics
- Lineage And Versioning Semantics
- Write Governance And Promotion

### 6. Write Governance And Promotion

Why:
The static layer must stay governed, and trust must affect retrieval without being baked into history.

Covers:

- SpaceWritePolicy
- Confidence vs TrustLevel
- Promotion: Episodic -> Static
- corpus and profile limits

Must prove:

- writer policy enforcement per kind
- trust resolution at query time from policy
- promotion is implemented as a governed write, not in-place mutation
- live corpus limits are enforced per kind

Type:
`foundation`

Depends on:

- Domain Model And Error Contract
- Lineage And Versioning Semantics

### 7. Remember Write Path

Why:
Combine domain rules, storage updates, and error handling into the main hot-path write API.

Covers:

- `Remember`
- optimistic locking
- write-time validation
- transactional updates

Must prove:

- root, update, promotion, and derived write flows behave correctly
- `IfLatestID` acts as optimistic concurrency control
- all affected storage changes commit atomically
- rejected writes leave no partial state

Type:
`foundation`

Depends on:

- Space Bootstrap And Configuration Locking
- Lineage And Versioning Semantics
- Relation Validation Semantics
- Derived Memory Contract
- Write Governance And Promotion

### 8. Forget And Inactive Lineages

Why:
Soft delete changes both lifecycle and retrieval visibility and must remain fully auditable.

Covers:

- `Forget`
- Lineage immutability
- Inactive Lineages

Must prove:

- forget sets `IsForgotten=true` and clears `IsLatest`
- forgetting latest deactivates the lineage
- older versions are not automatically restored
- inactive lineages disappear from live retrieval
- forget records actor and reason in audit storage

Type:
`high`

Depends on:

- Remember Write Path
- Live Corpus Storage Layout

### 9. Revive Contract

Why:
Reactivating a lineage is an explicit lifecycle operation with its own correctness rules.

Covers:

- `Revive`
- Inactive Lineages

Must prove:

- revive works only for inactive lineages
- new latest is chained to the forgotten latest
- root kind remains authoritative for the lineage
- revive does not un-forget historical records
- revive is independently audited

Type:
`high`

Depends on:

- Forget And Inactive Lineages
- Lineage And Versioning Semantics
- Write Governance And Promotion

### 10. Live Corpus Storage Layout

Why:
The retrieval contract depends on how hot-path candidate sets are physically maintained.

Covers:

- bucket layout
- hot path vs cold path split

Must prove:

- `spaces/` contains only live memory IDs
- `latest/` resolves current latest per root
- `embeddings/` and `memories/` remain consistent
- cold-path buckets stay off the retrieval hot path

Type:
`foundation`

Depends on:

- Domain Model And Error Contract

### 11. Recall Retrieval Contract

Why:
`Recall` is the primary read-path API and must exactly match the live-corpus semantics in the architecture.

Covers:

- `Recall`
- Retrieval Semantics
- API Contracts at a Glance

Must prove:

- recall operates only on live corpus
- relations are not traversed
- filters are applied before scoring
- returned results are ordered by final score

Type:
`foundation`

Depends on:

- Live Corpus Storage Layout
- Space Bootstrap And Configuration Locking

### 12. Scoring Engine

Why:
The scoring formula is a product contract, not an implementation detail.

Covers:

- scoring formula
- Confidence vs TrustLevel
- multi-space scoring

Must prove:

- score uses cosine × confidence × trust × recency × space weight
- recency applies only to episodic memories
- static and derived memories do not decay by age
- request overrides take precedence over space defaults
- trust is resolved from current policy, not persisted snapshots

Type:
`foundation`

Depends on:

- Recall Retrieval Contract
- Write Governance And Promotion

### 13. GetProfile Layered Retrieval

Why:
`GetProfile` is a separate contract for context assembly, not a thin wrapper around `Recall`.

Covers:

- Dual-Layer Model
- `ProfileRequest`
- `MemoryProfile`

Must prove:

- static and episodic layers are returned separately
- static layer includes both static and derived memories
- profile limits are independent from corpus limits
- each layer is sorted by score
- orphaned derives are filtered only when requested

Type:
`high`

Depends on:

- Recall Retrieval Contract
- Scoring Engine
- Derived Memory Contract

### 14. FindCandidates Contract

Why:
Candidate search has a different purpose and a deliberately different scoring contract.

Covers:

- `FindCandidates`
- API Contracts at a Glance

Must prove:

- only raw cosine is used
- live-only search is the default
- historical inclusion is explicit and opt-in
- trust, confidence, recency, and space weighting are not applied
- the API does not imply relation recommendations

Type:
`medium-high`

Depends on:

- Live Corpus Storage Layout

### 15. Inspector And Graph Traversal

Why:
A full-spec `v1` must expose lineage, graph, and audit inspection paths.

Covers:

- `MemoryInspector`
- `GetLineage`
- `GetRelated`
- `GetAuditLog`

Must prove:

- full lineage retrieval returns chronological history
- graph traversal can reach historical and forgotten records
- audit retrieval is space- and time-scoped
- inspection APIs do not affect hot-path retrieval semantics

Type:
`high`

Depends on:

- Live Corpus Storage Layout
- Remember Write Path
- Forget And Inactive Lineages
- Revive Contract

### 16. Auditability And Forget Records

Why:
The system must be explainable after the fact, not just operationally correct at write time.

Covers:

- `audit_log/`
- `forget_log/`
- cold-path observability

Must prove:

- every write, forget, revive, and migration emits audit entries
- forget actor/reason are indexed for targeted lookup
- the stored history is sufficient to explain current corpus state

Type:
`high`

Depends on:

- Remember Write Path
- Forget And Inactive Lineages
- Revive Contract
- Embedding Migration Workflow

### 17. Orphaned Source Propagation

Why:
Forgetting sources of derived memories is a cross-cutting transactional contract and should be treated explicitly.

Covers:

- `HasOrphanedSources`
- source-forget propagation
- orphan filtering in retrieval

Must prove:

- forgetting a source atomically marks affected live derived memories
- the orphaned flag is never cleared
- derived memories remain live unless callers exclude them
- affected derives remain inspectable for audit

Type:
`high`

Depends on:

- Derived Memory Contract
- Forget And Inactive Lineages
- GetProfile Layered Retrieval
- Recall Retrieval Contract

### 18. Concurrency And Consistency

Why:
The write path must stay correct under contention, and reads must never observe partial state.

Covers:

- Concurrency Model
- `IfLatestID`
- bbolt transaction semantics

Must prove:

- update races are rejected with `ErrConflict` when optimistic locking is used
- reads observe consistent snapshots
- accepted last-write-wins behavior is limited to cases allowed by the architecture

Type:
`high`

Depends on:

- Remember Write Path
- Live Corpus Storage Layout

### 19. Embedding Migration Workflow

Why:
A full-spec `v1` includes not just model locking, but explicit migration between embedding models.

Covers:

- `MigrateEmbeddings`
- space write lock during migration
- idempotent restart behavior

Must prove:

- writes are rejected with `ErrSpaceMigrating` during migration
- reads continue to work during migration
- all embeddings are re-embedded, including historical ones
- crash recovery preserves safety via write lock
- re-running migration is safe

Type:
`medium-high`

Depends on:

- Space Bootstrap And Configuration Locking
- Live Corpus Storage Layout
- Auditability And Forget Records

### 20. Operational Config And Disk Layout

Why:
Runtime configuration and persisted data layout need a stable contract for local operation and backup.

Covers:

- `config.toml`
- on-disk layout
- per-space runtime overrides

Must prove:

- half-life, limits, weights, and embedder settings are read and applied correctly
- on-disk layout is stable and backup-friendly
- runtime config and persisted data responsibilities are clearly separated

Type:
`medium`

Depends on:

- Space Bootstrap And Configuration Locking
- Scoring Engine
- Embedding Migration Workflow

### 21. Spec Verification Suite

Why:
For a full-spec `v1`, tests are part of the product contract and must mirror the architecture.

Covers:

- all architectural contracts

Must prove:

- invariants hold across normal and failing writes
- lineage, relation, and governance edge cases are covered
- retrieval semantics match the live corpus contract
- multi-space scoring behaves correctly
- migration and concurrency behavior are verified
- auditability and orphan propagation are verified

Type:
`mandatory cross-cutting`

Depends on:

- all capability slices

## Dependency View

The practical foundation order for implementation planning is:

1. Domain Model And Error Contract
2. Space Bootstrap And Configuration Locking
3. Lineage And Versioning Semantics
4. Relation Validation Semantics
5. Write Governance And Promotion
6. Live Corpus Storage Layout
7. Derived Memory Contract
8. Remember Write Path
9. Forget And Inactive Lineages
10. Recall Retrieval Contract
11. Scoring Engine
12. GetProfile Layered Retrieval
13. FindCandidates Contract
14. Revive Contract
15. Orphaned Source Propagation
16. Inspector And Graph Traversal
17. Auditability And Forget Records
18. Concurrency And Consistency
19. Embedding Migration Workflow
20. Operational Config And Disk Layout
21. Spec Verification Suite

## How To Use This Document

This spec map should be used as the source for the next planning step:

1. convert each slice into one or more spec-driven backlog items
2. define acceptance criteria per slice before implementation
3. preserve dependency order for foundational slices
4. keep verification tasks tied to architectural contracts, not package names

When backlog generation begins, tasks should be grouped by capability slice rather than by API method alone. This reduces hidden coupling and makes it easier to verify the system against the architecture.
