# OmnethDB — Architecture

> **A versioned knowledge graph for autonomous agents.**
> Not a vector store. Not a cache. A first-class memory primitive.

---

## The Problem

LLMs are stateless. The work they do is not. Every observation an agent makes — a deployment pattern, a code invariant, a repeated failure — is discarded at the end of the run unless something persists it.

Existing solutions fail in one of two ways:

**Flat vector stores** have no versioning semantics. When a fact changes, both the old and new observation exist in the corpus with equal weight. Agents retrieve contaminated context. There is no concept of "this fact supersedes that one."

**File-based context** (AGENTS.md, decisions registers, .context/) is human-writable but not queryable, not structured, and not composable across multiple agents operating on the same project.

OmnethDB solves this with a narrow guarantee: **facts have explicit lineage, relations are typed, and retrieval always returns the current version of knowledge** — not a probabilistic mix of history and present.

---

## Core Invariants

These hold at all times. Any operation that would violate them is rejected with an error.

1. **IsLatest exclusivity**: within a root lineage, exactly one memory has `IsLatest = true` at any point in time.
2. **Updates atomicity**: setting a new `IsLatest` and clearing the previous one is a single transaction. Partial state does not exist.
3. **Relation locality**: both endpoints of any relation must belong to the same `SpaceID`. Cross-space relations are forbidden.
4. **Derives requires sources**: a `Derived` memory must carry `SourceIDs` with at least two distinct memory IDs, all `IsLatest = true` at write time. Sources must be `KindEpisodic` or `KindStatic` — not `KindDerived`.
5. **Lineage immutability**: memories are never deleted from storage. `Forget` marks; it does not remove.
6. **No implicit inference**: OmnethDB does not infer relation types. Relations are always set by the caller. The system enforces invariants; it does not make semantic decisions.
7. **Embedding model lock**: the `EmbeddingModelID` for a space is fixed on first write and cannot change without an explicit migration. Writes with a mismatched embedder are rejected.

---

## Data Model

### Memory

```go
type Memory struct {
    ID        string
    SpaceID   string

    Content   string     // the fact, observation, or derived pattern
    Embedding []float32  // vector of Content; dimension fixed per space

    Kind      MemoryKind // Static | Episodic | Derived

    // Provenance
    Actor      Actor     // who wrote this memory
    Confidence float32   // 0.0–1.0; caller's stated epistemic certainty at write time

    // Versioning
    Version  int
    IsLatest bool
    ParentID *string // ID of the memory this supersedes; nil for root
    RootID   *string // ID of the original memory in this lineage; nil if this is the root

    // For Derived memories only
    SourceIDs []string // IDs of the memories this was derived from (min 2)
    Rationale string   // explanation of the derivation; required; non-empty

    // Lifecycle
    IsForgotten bool
    ForgetAfter *time.Time // TTL; nil = no expiry

    Relations MemoryRelations
    Metadata  map[string]any
    CreatedAt time.Time
}

type MemoryKind uint8

const (
    KindEpisodic MemoryKind = iota // concrete event; may have TTL; any-actor writable
    KindStatic                      // stable project fact; long-lived; governed write
    KindDerived                     // synthesized from ≥2 Episodic/Static sources
)

type Actor struct {
    ID         string    // agent ID, user ID, or "system"
    Kind       ActorKind // Human | Agent | System
    TrustLevel float32   // 0.0–1.0; configured in SpaceWritePolicy; not caller-declared
}

type ActorKind uint8

const (
    ActorHuman  ActorKind = iota
    ActorAgent
    ActorSystem
)

type MemoryRelations struct {
    Updates []string // this memory supersedes these; they become IsLatest=false
    Extends []string // this memory adds context; no state change on targets
    Derives []string // same as SourceIDs; stored here for graph traversal
}
```

### Confidence vs TrustLevel

These two fields measure different things and must not be conflated:

| Field | Set by | Meaning |
|---|---|---|
| `Confidence` | Caller at write time | Epistemic certainty about the fact itself. An agent that directly observed a failure sets 0.9; one that inferred it from noisy output sets 0.4. |
| `Actor.TrustLevel` | Space write policy | System's trust in this actor's judgment. Populated by OmnethDB from `SpaceWritePolicy.TrustLevels` at write time — the caller does not set it. |

Both participate in retrieval scoring (see Retrieval Model). A confident agent with low system trust scores lower than a moderately-confident trusted agent. This is intentional: it separates what the agent believes from how much the system should weight that belief.

`TrustLevel` defaults:
- `ActorHuman`: 1.0
- `ActorSystem`: 1.0
- `ActorAgent` with explicit entry in policy: configured value
- `ActorAgent` with no entry in policy: `SpaceWritePolicy.DefaultAgentTrust` (default: 0.7)

The trust model is not a reputation system. Trust levels do not update dynamically based on history. They are configuration. This is sufficient for v1.

---

## Relation Semantics

### Updates

**Definition**: memory A Updates memory B means A is the authoritative successor to B. B's content is superseded.

**Effect on write**: B's `IsLatest` is set to `false` atomically within the same transaction. A's `IsLatest` is set to `true`. A's `ParentID = B.ID`. A's `RootID = B.RootID ?? B.ID`.

**Invariants**:
- A memory can be the target of at most one Updates relation. Forked versioning is not supported.
- A memory cannot Update itself or any of its ancestors (no cycles).
- A memory cannot Update a `IsForgotten` memory. Create a new root instead.
- `Updates` targets must be in the same space as the source.
- `KindDerived` memories may be Updated by another `KindDerived` memory. The replacement must provide a fresh `SourceIDs` set and a `Rationale` explaining the revision. It does not inherit the previous memory's sources.

**Prohibited**:
- Skipping versions: you must Update the current `IsLatest`, not an older ancestor.
- Updating across kinds (e.g., Episodic Updates Static). The replacement must be the same Kind as the target.

**Example**:
```
v1: "deploy takes ~5 min"                IsLatest=false, RootID=nil
  └─Updates─▶ v2: "deploy takes ~12 min after auth middleware added"   IsLatest=true, RootID=v1.ID
```

---

### Extends

**Definition**: memory A Extends memory B means A adds context to B without contradiction. Both remain valid simultaneously.

**Effect on write**: no state change on B. The relation is recorded in the graph.

**Invariants**:
- Both A and B must be `IsLatest = true` at write time.
- Extends does not propagate through lineage: if B is later superseded by C, A's Extends relation still points to B. The caller decides whether to re-anchor A.
- `Extends` targets must be in the same space.

**Prohibited**:
- Using Extends to record a correction or replacement. If the new information contradicts the old, use Updates.

**On retrieval**: Extends relations are NOT traversed during Recall or GetProfile. A query for "payments schema" that matches A does not automatically surface B because A Extends B. Relations are for graph inspection (see MemoryInspector), not for retrieval expansion. See Retrieval Semantics for the complete contract.

---

### Derives

**Definition**: memory A (Kind=Derived) was synthesized by an agent from a pattern across ≥2 source memories. A represents knowledge not explicit in any single source.

**Effect on write**: sources are recorded in `SourceIDs` and `Relations.Derives`. Sources are not modified.

**Invariants**:
- `len(SourceIDs) >= 2`.
- All `SourceIDs` must be `IsLatest = true` at write time.
- Sources must be `KindEpisodic` or `KindStatic`. `KindDerived` memories cannot be sources. This prevents compounding inference errors: Derived memories represent a synthesis boundary. Higher-order knowledge must be promoted to Static (a governed write) before it can serve as source for further synthesis.
- `Rationale` must be non-empty. Without it the derivation is not auditable.
- `Actor.Kind` must be `ActorAgent` or `ActorHuman`. System cannot create Derived memories.

**Evolution**: Derived memories CAN be Updated by another Derived memory. The replacement must supply a fresh `SourceIDs` set (≥2, Episodic/Static) and a non-empty `Rationale` explaining why the previous synthesis was wrong or incomplete. This models the correct case: an agent observes new evidence that refines a prior synthesis. The old Derived memory is preserved with `IsLatest=false`.

A Derived memory that is simply invalidated (not replaced by a better synthesis) is retired via `Forget`, not Updates.

**On retrieval**: Derives relations are NOT traversed during Recall or GetProfile. See Retrieval Semantics.

**Example**:
```go
Remember(ctx, MemoryInput{
    SpaceID:    "repo:company/app",
    Content:    "all DB migrations require a smoke test before production deploy",
    Kind:       KindDerived,
    Actor:      Actor{ID: "agent:scout-1", Kind: ActorAgent},
    Confidence: 0.85,
    SourceIDs:  []string{"mem-payments-migration-fail", "mem-orders-migration-fail", "mem-users-migration-fail"},
    Rationale:  "three independent modules failed in production due to schema migration without prior validation",
})
```

---

## Dual-Layer Model

`MemoryKind` maps to two retrieval layers with distinct semantics:

```
┌────────────────────────────────────────────────────────────────────┐
│  STATIC LAYER  (KindStatic | KindDerived)                          │
│                                                                    │
│  Long-lived facts about the project.                               │
│  Returned by GetProfile up to MaxStaticMemories per space.         │
│  No recency decay in scoring. No TTL.                              │
│                                                                    │
│  Examples:                                                         │
│    · "pagination is cursor-based, not offset"                      │
│    · "deploy requires approval from @senior-backend"               │
│    · "all DB migrations require a smoke test before prod" [D]      │
└────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────┐
│  EPISODIC LAYER  (KindEpisodic)                                    │
│                                                                    │
│  Concrete events. Scored by relevance to the current query.        │
│  Subject to recency decay. May have ForgetAfter TTL.               │
│                                                                    │
│  Examples:                                                         │
│    · "PR #142 rejected — N+1 query in payments.findAll()"          │
│    · "deploy failed 2024-03-15 — missing env var in prod"          │
│    · "merge freeze active until 2024-03-22"  [ForgetAfter set]     │
└────────────────────────────────────────────────────────────────────┘
```

`GetProfile` composes both layers for agent context injection:

```go
type MemoryProfile struct {
    // KindStatic + KindDerived memories, IsLatest=true, not forgotten.
    // Ordered by score(m, query) descending.
    // Bounded by MaxStaticMemories in space config.
    // If the static corpus is well-governed, this fits a context window.
    // If it does not, that is a governance failure — reduce MaxStaticMemories or curate.
    Static []ScoredMemory

    // Top-EpisodicTopK KindEpisodic by score, IsLatest=true, not forgotten, not expired.
    Episodic []ScoredMemory
}
```

`Static` is returned ordered by score against the query — not as a flat dump. This ensures the most relevant static facts appear first when the caller truncates to fit a context window.

---

## Space Model

A `SpaceID` is an opaque string. Format constraint: `[a-zA-Z0-9_:/-]{1,256}`.

OmnethDB does not parse, split, or traverse space hierarchies. `repo:company/app:agent:scout` is a string, not a path. The hierarchy is a caller convention.

**Space creation**: implicit on first write. The `EmbeddingModelID` and dimension are locked at that point.

**Cross-space reads**: `Recall` and `GetProfile` accept `SpaceIDs []string`. The caller explicitly lists every space to query. Results are merged and re-ranked before return. There is no automatic inheritance.

**Cross-space writes**: forbidden. A memory's `SpaceID` is immutable after creation.

**Space weights**: callers assign per-space weights in the request to control the influence of each space on the merged result. The default weight is 1.0.

```go
type RecallRequest struct {
    SpaceIDs     []string
    SpaceWeights map[string]float32 // spaceID → weight multiplier; default 1.0 if absent
    Query        string
    TopK         int
    Kinds        []MemoryKind       // nil = all kinds
}

type ProfileRequest struct {
    SpaceIDs     []string
    SpaceWeights map[string]float32
    Query        string
    EpisodicTopK int                // default: 10
}
```

**Multi-space scoring**: the score of a memory from space `s` is multiplied by `SpaceWeights[s]` before merging. This prevents a noisy run-scoped space (e.g., `repo:company/app:workflow:run-xyz`) from drowning out project-wide static facts. The caller is responsible for assigning appropriate weights — a typical pattern is 1.0 for project spaces and 0.5 for run spaces.

**Space deletion**: not supported in v1. Forgetting individual memories is the supported path.

---

## API Surface

```go
// Primary interface — hot path operations.
type MemoryStore interface {
    // Write a new memory. Relations in input.Relations are applied and validated.
    // Updates targets have IsLatest set to false atomically.
    // Returns ErrConflict if IfLatestID check fails.
    // Returns ErrPolicyViolation if actor lacks write permission for the Kind.
    // Returns ErrCorpusLimit if the space's memory limit is reached.
    // Returns ErrEmbeddingModelMismatch if the embedder's ModelID != space's locked ModelID.
    Remember(ctx context.Context, input MemoryInput) (*Memory, error)

    // Semantic retrieval across one or more spaces.
    // Returns only IsLatest=true, not forgotten, not expired memories.
    // Results are ordered by score (see Retrieval Model).
    // Relations are NOT traversed. This is a vector search, not a graph walk.
    Recall(ctx context.Context, req RecallRequest) ([]ScoredMemory, error)

    // Compose an agent context profile for a query.
    // Returns Static layer (up to MaxStaticMemories, ordered by score) +
    // Episodic layer (top-EpisodicTopK by score).
    GetProfile(ctx context.Context, req ProfileRequest) (*MemoryProfile, error)

    // Soft-delete a memory. Sets IsForgotten=true, IsLatest=false.
    // If the memory is IsLatest=true, the lineage becomes inactive (no current latest).
    // The previous version in the chain is NOT automatically restored.
    // Records actor and reason in audit_log.
    Forget(ctx context.Context, id string, actor Actor, reason string) error

    // Return top-K memories by raw cosine similarity to content.
    // Includes non-latest and forgotten memories. No score adjustments.
    // Used by callers to find relation candidates before writing.
    FindCandidates(ctx context.Context, spaceID string, content string, topK int) ([]ScoredMemory, error)
}

// Secondary interface — cold path for inspection and debugging.
// Not latency-sensitive. Not required for agent hot path.
type MemoryInspector interface {
    // Return all versions in a lineage, including superseded and forgotten, chronological.
    GetLineage(ctx context.Context, rootID string) ([]Memory, error)

    // Traverse the relation graph from a memory ID.
    // depth=1 returns immediate relations only.
    // Returns memories regardless of IsLatest or IsForgotten status.
    GetRelated(ctx context.Context, memoryID string, kind RelationType, depth int) ([]Memory, error)

    // Return the full audit log for a space: all writes, forgets, migrations.
    GetAuditLog(ctx context.Context, spaceID string, since time.Time) ([]AuditEntry, error)
}
```

```go
type MemoryInput struct {
    SpaceID     string
    Content     string
    Kind        MemoryKind
    Actor       Actor      // TrustLevel is populated by OmnethDB from policy; caller sets ID and Kind
    Confidence  float32    // required; use 1.0 if not reasoning about uncertainty
    ForgetAfter *time.Time
    Metadata    map[string]any

    // For Derived memories
    SourceIDs []string
    Rationale string

    // Optimistic lock for Updates chains.
    // If Relations.Updates is non-empty and IfLatestID is set,
    // the write fails with ErrConflict if current latest != IfLatestID.
    IfLatestID *string

    Relations MemoryRelations
}

type ScoredMemory struct {
    Memory
    Score float32
}
```

---

## Retrieval Semantics

### Contract

**Recall and GetProfile operate exclusively on the live corpus.** This is not a configuration option — it is a fundamental contract.

Live corpus definition: `IsLatest=true AND IsForgotten=false AND (ForgetAfter IS NULL OR ForgetAfter > now)`.

**Relations do NOT affect retrieval results.** The relation graph — Updates, Extends, Derives — is used only at write time (to enforce invariants and update `IsLatest`) and at inspection time (via `MemoryInspector`). It is not traversed during `Recall` or `GetProfile`. A query that matches memory A does not surface memory B because A Extends B. If B is relevant, it must score on its own.

This is intentional. Graph traversal in retrieval creates unpredictable result sets that are hard to reason about and impossible to tune. The caller can use `GetRelated` to traverse the graph explicitly after retrieval if needed.

### Scoring

Every candidate in the live corpus is scored:

```
score(m, query) = cosine(embed(query), m.Embedding)
                × m.Confidence
                × m.Actor.TrustLevel
                × recency(m)
                × space_weight(m.SpaceID)
```

Where:

```
recency(m) = 1.0
    if m.Kind ∈ {KindStatic, KindDerived}

recency(m) = exp( -ln(2) / H × age_days(m) )
    if m.Kind == KindEpisodic

age_days(m) = (now - m.CreatedAt).Days()
H           = half-life in days; default 30; configurable per space in config.toml

space_weight(spaceID) = SpaceWeights[spaceID] from the request; default 1.0
```

**Rationale for each factor:**

- `cosine`: primary relevance signal.
- `Confidence`: separates a well-grounded observation from speculative noise written by the same actor.
- `Actor.TrustLevel`: separates a well-grounded actor from a noisy one writing with the same stated confidence. Prevents an untrusted agent from contaminating retrieval even if it writes coherent-sounding facts.
- `recency`: Static and Derived facts represent synthesized, durable knowledge — age is not a staleness signal for them. An `Updates` relation retires stale static facts. Episodic events (failures, observations, transient states) naturally decay in relevance over time; the half-life models this.
- `space_weight`: allows callers to explicitly deprioritize noisy spaces (e.g., run-scoped episodics) relative to project-wide facts.

**Hard filters (applied before scoring; zero score memories are not returned):**

1. `IsLatest = true`
2. `IsForgotten = false`
3. `ForgetAfter IS NULL OR ForgetAfter > now`
4. `SpaceID IN request.SpaceIDs`
5. `Kind IN request.Kinds` (if specified)

---

## Inactive Lineages

A lineage becomes **inactive** when its `IsLatest=true` memory is forgotten:

1. `Forget(id)` sets `IsForgotten=true` and `IsLatest=false` on the target.
2. The `latest/` bucket entry for `rootID` is cleared.
3. The previous version in the lineage (ParentID) is NOT automatically restored. The previous version's `IsLatest` was already `false`; it stays `false`.

**Retrieval behavior**: inactive lineages have no `IsLatest=true` member. They are invisible to `Recall` and `GetProfile`. The knowledge is gone from the live corpus.

**Inspection**: `GetLineage(rootID)` returns all versions including forgotten, allowing audit of why knowledge was removed.

**Reactivation**: the caller has two options:

Option A — create a new memory that Updates any version in the lineage:
```go
// Reactivates the lineage; new memory becomes IsLatest=true
Remember(ctx, MemoryInput{
    Content:   "revised: deploy takes ~8 min",
    Kind:      KindStatic,
    Relations: MemoryRelations{Updates: []string{forgottenMemoryID}},
    ...
})
```
The Updates invariant normally requires the target to not be forgotten. **Exception**: a memory CAN Update a `IsForgotten` memory for the purpose of reactivation, provided the caller passes the forgotten memory's ID explicitly. The target remains forgotten; the new memory starts a fresh `IsLatest=true` in the same lineage with `ParentID` pointing to the forgotten one. This preserves the chain without un-forgetting.

Option B — create a fresh root with no relation to the old lineage. Use this when the old knowledge should remain buried.

---

## Memory Governance

Without governance, the static layer accumulates unboundedly. Every agent writes "important facts" and three months later the context window is full of stale, conflicting, low-quality static memories. This section defines the mechanisms that prevent this.

### SpaceWritePolicy

Each space has a `SpaceWritePolicy` configured at construction time. It governs who can write each Kind.

```go
type SpaceWritePolicy struct {
    // Default trust level for agents not explicitly listed.
    // Used to populate Actor.TrustLevel at write time.
    DefaultAgentTrust float32 // default: 0.7

    // Per-actor trust overrides.
    TrustLevels map[string]float32 // actorID → trust level

    // Who can write KindEpisodic. Default: any actor.
    EpisodicWriters WritersPolicy

    // Who can write KindStatic. Default: Human or System only.
    StaticWriters WritersPolicy

    // Who can write KindDerived. Default: any Agent or Human.
    DerivedWriters WritersPolicy

    // Who can call Promote (Episodic → Static via a governed write).
    // Default: Human only.
    PromotePolicy WritersPolicy

    // Hard corpus limits.
    MaxStaticMemories  int // default: 100
    MaxEpisodicMemories int // default: 10_000
}

type WritersPolicy struct {
    AllowHuman      bool
    AllowSystem     bool
    AllowAllAgents  bool     // overrides AllowedAgentIDs if true
    AllowedAgentIDs []string // explicit whitelist; empty = deny all agents
    MinTrustLevel   float32  // actor.TrustLevel must be >= this; default 0.0
}
```

`SpaceWritePolicy` is not a runtime ACL system. It is a configuration object passed at `Open(path, policy)`. There is no API to mutate it after the space is opened. Changes require restarting the process with a new policy. This is intentional: governance rules should be in code, not in the database.

### Promotion: Episodic → Static

An Episodic memory cannot become Static in-place. Promotion is an explicit write:

```go
// Caller reads an episodic memory, decides it represents a durable fact,
// and creates a new Static memory that Updates the episodic one.
Remember(ctx, MemoryInput{
    Content:   "pagination in this codebase is cursor-based",  // refined formulation
    Kind:      KindStatic,
    Actor:     Actor{ID: "user:alice", Kind: ActorHuman},
    Confidence: 1.0,
    Relations: MemoryRelations{Updates: []string{episodicMemoryID}},
})
```

The original Episodic memory is set `IsLatest=false`. The new Static memory is `IsLatest=true`. The promotion is recorded in the audit log with the actor.

Promotion is a governed write: the caller must have `PromotePolicy` permission. Only actors with permission can elevate episodic observations to permanent facts.

### Corpus limits

`MaxStaticMemories` and `MaxEpisodicMemories` are hard limits per space. On limit breach, `Remember` returns `ErrCorpusLimit`. The caller must `Forget` existing memories before writing new ones.

These are not garbage collectors. They are pressure valves that surface governance failures immediately. An agent that hits `ErrCorpusLimit` must curate. This is the intended behavior.

### Preventing garbage: summary

| Mechanism | What it prevents |
|---|---|
| `ForgetAfter` TTL on Episodic | Time-bounded facts (freeze, incident) accumulating after expiry |
| `Updates` chain discipline | Multiple contradictory versions of the same fact competing in retrieval |
| `StaticWriters` policy | Agents promoting every observation to permanent facts |
| `Confidence × TrustLevel` in scoring | Low-quality writes by untrusted actors ranking alongside trusted ones |
| `MaxStaticMemories` hard limit | Silent static corpus bloat |
| Audit log | Inability to explain why the corpus contains what it contains |

---

## Concurrency Model

bbolt uses a process-level write lock. All write transactions serialize. This is not a weakness at the expected write volume: agent memory writes are low-frequency relative to reads.

### The concurrent Updates problem

Two agents both read `latest(root) = v1`. Both write `v2` updating `v1`. Without protection, both succeed — but only one transaction actually set `v1.IsLatest=false`. The other transaction wrote a memory whose Updates target is no longer the current latest. The chain is corrupt.

**Solution: `IfLatestID` optimistic lock**

```go
current, _ := store.Recall(ctx, RecallRequest{SpaceIDs: []string{spaceID}, Query: topic, TopK: 1})
latestID := current[0].ID

_, err := store.Remember(ctx, MemoryInput{
    ...
    Relations:  MemoryRelations{Updates: []string{latestID}},
    IfLatestID: &latestID,
})
if errors.Is(err, ErrConflict) {
    // re-read latest and retry
}
```

When `IfLatestID` is set, the write transaction checks `latest[rootID] == *IfLatestID` before committing. If not, it returns `ErrConflict` without writing.

`IfLatestID` is optional. Callers that do not set it get last-write-wins behavior. This is acceptable for Episodic writes (events are rarely contended) and for new roots. For Static Updates, callers should set it.

### Read consistency

`Recall` runs in a bbolt read transaction. It is isolated from concurrent writes by bbolt's MVCC. A `Recall` during a concurrent `Remember` sees a consistent snapshot — either the old state or the new, never a partial write.

---

## Storage Architecture

OmnethDB is embedded. Single Go binary, no external processes, no CGO.

### Bucket layout

```
Hot path (millisecond latency; accessed on every Recall / GetProfile):

  spaces/        → spaceID → []memoryID
                   Contains ONLY IsLatest=true, IsForgotten=false memory IDs.
                   Updated atomically with every Remember/Forget write.
                   Recall candidate loading = single bucket scan, no filter pass.

  latest/        → rootID → latestID
                   O(1) lookup for IfLatestID optimistic lock checks.
                   Cleared when Forget makes a lineage inactive.

  embeddings/    → memoryID → []float32 (LE binary)
                   Stores ALL embeddings including historical versions.
                   Hot path reads only IDs from spaces/ then fetches embeddings here.

  memories/      → memoryID → MemoryRecord (msgpack, no Embedding field)
                   All memory records including superseded and forgotten.


Cold path (seconds latency acceptable; governance, inspection, debugging):

  relations/     → "fromID:relationType" → []toID
                   Adjacency list for graph traversal via MemoryInspector.

  spaces_config/ → spaceID → SpaceConfig (msgpack)
                   Stores EmbeddingModelID, dimension, half-life H, write policy.

  audit_log/     → timestamp+uuid → AuditEntry (msgpack)
                   Append-only record of every write, forget, and migration.
                   Actor, operation, affected memory IDs, timestamp.

  forget_log/    → memoryID → ForgetRecord (actor, reason, time)
                   Subset of audit_log indexed by memory ID for fast per-memory lookup.
```

All buckets modified by a single `Remember` call are written in one bbolt transaction. `Forget` modifies `memories/`, `spaces/`, `latest/`, `audit_log/`, and `forget_log/` in one transaction.

The hot/cold split is a physical separation: hot-path queries never touch `relations/`, `audit_log/`, or `forget_log/`. This keeps hot-path latency predictable regardless of how large the cold-path data grows.

### Vector search

Brute-force cosine over the `spaces/` candidate set. Pure Go, no CGO.

The candidate set contains only `IsLatest=true, IsForgotten=false` IDs — this is enforced by how `spaces/` is maintained, not by a filter at query time. Recall loads embeddings for those IDs from `embeddings/`, computes scores, applies formula, returns top-K.

At corpus sizes up to ~10k memories per space this is fast enough for the agent hot path. At larger corpora, a pure Go HNSW index ([coder/hnsw](https://github.com/coder/hnsw)) can be placed in front without changing the `MemoryStore` interface. The `spaces/` bucket continues to define the candidate set; the index provides ANN over it.

These are not claims about specific latency numbers — that requires benchmarking against the target hardware and embedding dimensions. Run the benchmark suite (v1.0 milestone) before committing to SLAs.

### Embedding Provider

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    Dimensions() int
    ModelID() string // stable, unique identifier for this model version
                     // e.g., "openai/text-embedding-3-small" or "local/bge-m3-v1.2"
}
```

**Embedding model versioning:**

On first write to a space, OmnethDB records `embedder.ModelID()` and `embedder.Dimensions()` in `spaces_config/`. On every subsequent write, it checks:

```
embedder.ModelID() == spaces_config[spaceID].EmbeddingModelID
AND embedder.Dimensions() == spaces_config[spaceID].Dimension
```

If either check fails, the write is rejected with `ErrEmbeddingModelMismatch`. This prevents silent corruption from mixed-model vector spaces where cosine similarity is no longer meaningful.

**When the embedding model changes:**

The operator calls `MigrateEmbeddings(ctx, spaceID, newEmbedder)`. This:

1. Sets the space to write-locked (Recall still works; Remember returns `ErrSpaceMigrating`).
2. Re-embeds all memories in `embeddings/` using the new embedder — all versions, not just latest.
3. Updates `spaces_config[spaceID].EmbeddingModelID` and `Dimension`.
4. Releases the write lock.

Migration is a full-corpus operation. Duration scales with corpus size and embedder latency. The caller is responsible for scheduling it during low-traffic windows.

There is no partial migration state — the space is either fully on the old model or fully on the new one. A crash during migration leaves the space write-locked; re-running `MigrateEmbeddings` is safe (idempotent re-embedding).

---

## On-Disk Layout

```
omneth/
  data/
    memory.db     ← single bbolt file; all spaces, all buckets
  config.toml     ← per-space overrides: half-life H, corpus limits, embedder config
```

Backup = copy `memory.db`. No WAL files, no side-car processes.

---

## What OmnethDB Is Not

**Not a general-purpose graph database.** The schema is fixed. The relation types are closed. This is the design.

**Not a conversation memory system.** Mem0, Zep, and Supermemory optimize for conversational continuity across user sessions. OmnethDB optimizes for project knowledge: facts about codebases, architecture decisions, and patterns that persist across agents and runs.

**Not an embedding service.** OmnethDB stores and retrieves vectors. It does not generate them.

**Not responsible for relation semantics.** OmnethDB enforces structural invariants. It does not determine whether two memories are semantically contradictory or complementary. That judgment belongs to the caller. `FindCandidates` gives the caller the nearest neighbors; the caller decides what to do with them.

**Not a reputation system.** `Actor.TrustLevel` is static configuration, not a score that updates based on agent behavior. Dynamic trust is future work contingent on evidence that static trust is insufficient.

---

## Design Decisions

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| Storage engine | bbolt | PostgreSQL, SQLite+sqlite-vec, Redis | Zero external processes; pure Go; ACID; etcd-proven; single file backup |
| Vector search | Brute-force cosine | HNSW, pgvector, FAISS | Avoids CGO entirely; HNSW is an upgrade path, not a starting point |
| Relation inference | None — caller-explicit | Automatic inference on write | Automatic inference makes `IsLatest` mutations non-auditable; `FindCandidates` provides search without making the decision |
| Derived sources | Episodic/Static only, not Derived | Allow Derived-from-Derived | Prevents compounding inference errors; higher-order knowledge must pass through Static promotion (governed write) |
| Derived evolution | Updates allowed with fresh SourceIDs | Forbid all Updates on Derived | A synthesis can be refined by better evidence; forbidding evolution forces Forget+recreate which loses lineage |
| Trust model | Static TrustLevel in policy + caller Confidence | Dynamic reputation | Minimal and auditable; dynamic trust requires evidence that static is insufficient |
| Relations in retrieval | Not traversed | Graph-expanded retrieval | Traversal creates unpredictable result sets; callers use MemoryInspector for explicit graph walks |
| Multi-space scoring | Explicit SpaceWeights per request | Automatic hierarchy | Caller knows which spaces are authoritative; automatic weighting is implicit coupling |
| Embedding versioning | ModelID lock + migration API | Best-effort compatibility | Silent vector space corruption is undetectable and catastrophic; hard lock forces explicit migration |
| Inactive lineage | No auto-restore | Restore previous on Forget | Auto-restore is action at a distance; caller decides whether to reactivate or bury |
| Concurrency for Updates | Optimistic lock via `IfLatestID` | Last-write-wins, pessimistic lock | LWW silently corrupts lineage; pessimistic lock is unnecessary overhead for low-contention workload |
| Versioning model | Immutable lineage + IsLatest pointer | In-place mutation | Full history required for audit and for MemoryInspector |
| Memory kinds | Three-value enum | `IsStatic bool` | Distinct governance, scoring, and decay rules per kind; extensible |
| Governance | SpaceWritePolicy at open() time | Runtime ACL | Governance rules belong in code; runtime-mutable policy is an attack surface |

---

## Open Questions

**Dynamic trust**: TrustLevels are static configuration. Should they update based on observed agent accuracy (e.g., memories from an agent that were frequently Forgotten or Derived from were wrong)? Left for v2; requires empirical evidence that static trust is insufficient.

**Forget cascade on Derives sources**: forgetting a source memory does not cascade to Derived memories that used it. A Derived memory may now have a SourceIDs entry pointing to a forgotten memory. Whether this should trigger automatic invalidation of the Derived memory is unresolved. Current behavior: Derived memory remains live; `GetLineage` on its sources reveals the forgotten entry.

**Cross-Derived Derives**: prohibited in v1 (sources must be Episodic or Static). If this proves too restrictive in practice, v2 can relax it with an explicit depth limit (e.g., max one level of Derived-as-source, with required re-validation of the intermediate).

---

## Roadmap

```
v0.1 — Core
  · Memory model: Kind, Actor (ID/Kind/TrustLevel), Confidence, SourceIDs, Rationale
  · Remember, Recall, GetProfile, Forget, FindCandidates
  · Updates invariant + IsLatest exclusivity + IfLatestID optimistic lock
  · Extends and Derives with all structural invariants
  · Inactive lineage semantics + reactivation via Updates on forgotten memory
  · SpaceWritePolicy: EpisodicWriters, StaticWriters, DerivedWriters, PromotePolicy
  · TrustLevel populated from policy at write time
  · Scoring formula: cosine × confidence × trust × recency × space_weight
  · Corpus limits: MaxStaticMemories, MaxEpisodicMemories
  · EmbeddingModelID lock per space (ErrEmbeddingModelMismatch)
  · Hot/cold bucket layout in bbolt

v0.2 — Embedding Layer
  · Embedder interface with ModelID()
  · OpenAI adapter (text-embedding-3-small)
  · Local ONNX adapter
  · MigrateEmbeddings with write-lock semantics
  · Embedding cache (skip re-embedding identical content strings)

v0.3 — Inspection & Observability
  · MemoryInspector: GetLineage, GetRelated, GetAuditLog
  · Prometheus metrics: write latency, recall latency, corpus size per space, active/inactive lineage counts
  · Structured audit_log with full write provenance

v0.4 — gRPC Service Mode
  · Proto definition + generated server
  · TLS + token auth
  · Health check, readiness probe

v1.0 — Production Hardening
  · HNSW index path for >100k memories per space (behind feature flag)
  · Snapshot/restore API
  · Configurable half-life H and SpaceWeights defaults per space via config.toml
  · Benchmark suite: recall latency at 1k / 10k / 100k memories, various embedding dims
```
