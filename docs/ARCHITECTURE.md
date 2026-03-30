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
    IsForgotten        bool
    ForgetAfter        *time.Time // TTL; nil = no expiry
    HasOrphanedSources bool       // permanent mark: one or more SourceIDs were forgotten after this memory was created
                                  // set atomically by Forget(sourceID); never cleared
                                  // records a historical fact about provenance, not a current validity verdict
                                  // only meaningful for KindDerived; always false for other kinds
                                  //
                                  // "never cleared" rationale: forgotten memories are not un-forgotten;
                                  // if the knowledge represented by the source was replaced (new lineage version),
                                  // the original SourceID still points to the forgotten record, not the replacement.
                                  // current validity is a caller judgment via GetRelated(sourceID, Derives, 1)

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
    ID   string    // agent ID, user ID, or "system"
    Kind ActorKind // Human | Agent | System
    // TrustLevel is NOT stored in the memory record.
    // It is resolved at query time from SpaceWritePolicy.TrustLevels[actor.ID].
    // See "Confidence vs TrustLevel" for the rationale.
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

These two signals measure different things and must not be conflated:

| Signal | Declared by | Stored in record | Meaning |
|---|---|---|---|
| `Confidence` | Caller at write time | Yes | Epistemic certainty about the fact itself. Direct observation → 0.9; noisy inference → 0.4. |
| `TrustLevel` | Space write policy | **No** | System's trust in this actor's judgment. Resolved at **query time** from `SpaceWritePolicy`. |

`TrustLevel` is **not persisted** in the memory record. It is resolved at retrieval time from the current `SpaceWritePolicy.TrustLevels[actor.ID]`. This is a deliberate choice:

- If an agent is discovered to be writing low-quality memories, lowering its `TrustLevel` in policy immediately deprioritizes **all** its memories in retrieval — not just future ones.
- Persisting TrustLevel would create a snapshot of the policy at write time. Old memories from a now-untrusted agent would retain their original high-trust ranking, making policy changes ineffective retroactively.
- Reproducibility of historical ranking is not a meaningful requirement for agent memory. Agents need the best current context, not a replay of past rankings.

The stored `Actor` carries only `ID` and `Kind` — enough to look up the current trust at query time and to audit who wrote the memory.

`TrustLevel` resolution at query time:
- `ActorHuman`: `SpaceWritePolicy.HumanTrust` (default: 1.0)
- `ActorSystem`: `SpaceWritePolicy.SystemTrust` (default: 1.0)
- `ActorAgent` with explicit entry: `SpaceWritePolicy.TrustLevels[actor.ID]`
- `ActorAgent` with no entry: `SpaceWritePolicy.DefaultAgentTrust` (default: 0.7)

The 1.0 defaults for Human and System reflect a typical deployment model where humans and the orchestrator are the primary authoritative writers. They are policy defaults, not ontological axioms. If a deployment has reason to distrust a specific human or system actor, `TrustLevels[actor.ID]` overrides the default for that actor regardless of `ActorKind`.

The trust model is not a reputation system. Trust levels are static configuration in `SpaceWritePolicy`. They do not update based on observed agent behavior. This is sufficient for v1.

---

## Relation Semantics

### Updates

**Definition**: memory A Updates memory B means A is the authoritative successor to B. B's content is superseded.

**Effect on write**: B's `IsLatest` is set to `false` atomically within the same transaction. A's `IsLatest` is set to `true`. A's `ParentID = B.ID`. A's `RootID = B.RootID ?? B.ID`.

**Invariants**:
- A memory can be the target of at most one Updates relation. Forked versioning is not supported.
- A memory cannot Update itself or any of its ancestors (no cycles).
- A memory cannot Update a `IsForgotten` memory. Use `Revive` to reactivate an inactive lineage (see Inactive Lineages).
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

**Orphaned sources**: if a source memory is forgotten after a Derived memory is created, `Forget(sourceID)` sets `HasOrphanedSources=true` on all live Derived memories that reference it — atomically, in the same transaction. The Derived memory remains in the live corpus. It is returned by `Recall` and `GetProfile` with `HasOrphanedSources=true` visible in the record. Callers that want to suppress these set `ExcludeOrphanedDerives=true` in the request. `GetRelated(sourceID, Derives, 1)` surfaces the affected Derived memories for audit.

The forgetting of a source does not automatically invalidate the synthesis. A pattern derived from three past incidents does not become false because one incident record was cleaned up. The `HasOrphanedSources` flag surfaces the condition; the agent or operator decides whether to revalidate or retire the Derived memory.

**On retrieval**: Derives relations are NOT traversed during Recall or GetProfile. See API Contracts at a Glance.

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
    // Capped at ProfileRequest.StaticTopK (default: SpaceWritePolicy.ProfileMaxStatic).
    // This is a context-assembly limit, not a corpus limit. The corpus may contain more.
    Static []ScoredMemory

    // KindEpisodic memories by score, IsLatest=true, not forgotten, not expired.
    // Capped at ProfileRequest.EpisodicTopK (default: SpaceWritePolicy.ProfileMaxEpisodic).
    Episodic []ScoredMemory
}
```

`Static` is ordered by score — not a flat dump. The most relevant static facts appear first. When the caller truncates to fit a context window, the least relevant facts are dropped, not arbitrary ones.

**Corpus limit vs profile limit**: `MaxStaticMemories` (corpus) and `ProfileMaxStatic` (profile) are independent. The corpus can hold 500 static memories; GetProfile returns the top 50 by relevance. Hitting `ErrCorpusLimit` means you have too many memories stored — a governance problem. Hitting `ProfileMaxStatic` means you are selecting from a well-governed corpus — normal operation.

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
    Kinds        []MemoryKind // nil = all kinds

    // Exclude KindDerived memories whose sources have been partially forgotten.
    // Default: false — they are returned with HasOrphanedSources=true.
    ExcludeOrphanedDerives bool
}

type ProfileRequest struct {
    SpaceIDs     []string
    SpaceWeights map[string]float32
    Query        string
    StaticTopK   int // default: SpaceWritePolicy.ProfileMaxStatic; 0 = use default
    EpisodicTopK int // default: SpaceWritePolicy.ProfileMaxEpisodic; 0 = use default

    // Exclude KindDerived memories whose sources have been partially forgotten.
    // Default: false — orphaned-source Derived memories are returned with HasOrphanedSources=true.
    // Set true to suppress them from the profile; use MemoryInspector.GetRelated to audit.
    ExcludeOrphanedDerives bool
}
```

**Multi-space scoring**: the score of a memory from space `s` is multiplied by `SpaceWeights[s]` before merging. This prevents a noisy run-scoped space (e.g., `repo:company/app:workflow:run-xyz`) from drowning out project-wide static facts.

Callers should not need to hand-tune weights on every request. Each space carries a `DefaultWeight float32` in its `SpaceConfig` (stored in `spaces_config/`; default: 1.0). If a space is not present in `RecallRequest.SpaceWeights`, its `SpaceConfig.DefaultWeight` is used. The per-request `SpaceWeights` map is an override, not the only mechanism.

A run-scoped space created with `DefaultWeight: 0.5` is automatically deprioritized in every query that includes it, without caller discipline. Project-wide spaces keep `DefaultWeight: 1.0`. Callers set per-space defaults once at space creation; they override per-request only for exceptional cases.

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
    // If the memory is a SourceID in any live KindDerived memories,
    // those memories have HasOrphanedSources set to true atomically in the same transaction.
    // Records actor and reason in audit_log.
    Forget(ctx context.Context, id string, actor Actor, reason string) error

    // Reactivate an inactive lineage by creating a new IsLatest=true memory.
    // Returns ErrLineageActive if the lineage is not inactive.
    // Returns ErrPolicyViolation if actor lacks write permission for the Kind.
    // See Inactive Lineages section for full semantics.
    Revive(ctx context.Context, rootID string, input ReviveInput) (*Memory, error)

    // Return top-K memories by raw cosine similarity to content.
    // Default: live corpus only (IsLatest=true, IsForgotten=false).
    // Use IncludeSuperseded / IncludeForgotten to access historical data.
    // No confidence/trust/recency adjustments — raw cosine only.
    // Purpose: find relation candidates before writing. Not for agent context assembly.
    FindCandidates(ctx context.Context, req FindCandidatesRequest) ([]ScoredMemory, error)
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

type FindCandidatesRequest struct {
    SpaceID string
    Content string
    TopK    int

    // Default false. Superseded memories (IsLatest=false) are excluded.
    // Set true to surface all historical versions — use when authoring Updates relations.
    IncludeSuperseded bool

    // Default false. Forgotten memories are excluded.
    // Set true only for explicit archaeology or audit workflows.
    IncludeForgotten bool
}

type ReviveInput struct {
    Content    string
    Kind       MemoryKind // must match the Kind of the lineage root; see "Inactive Lineages" for cross-kind evolution
    Actor      Actor
    Confidence float32
    Metadata   map[string]any
}
```

---

## API Contracts at a Glance

The three retrieval operations have different purposes and different scoring contracts:

| | `Recall` | `GetProfile` | `FindCandidates` |
|---|---|---|---|
| **Purpose** | Direct knowledge query during task execution | Context assembly before agent run | Candidate search for relation authoring |
| **Corpus** | Live only | Live only | Live by default; opt-in historical |
| **cosine** | ✓ | ✓ | ✓ (only factor) |
| **Confidence** | ✓ | ✓ | ✗ |
| **TrustLevel** | ✓ | ✓ | ✗ |
| **recency** | ✓ | ✓ episodic, ✗ static | ✗ |
| **space_weight** | ✓ | ✓ | ✗ |
| **Output shape** | Flat `[]ScoredMemory` | `MemoryProfile` (Static + Episodic separated) | Flat `[]ScoredMemory` |

`FindCandidates` uses raw cosine because the caller is evaluating similarity for relation-authoring purposes, not for ranking trustworthiness. A low-confidence memory that is highly similar to the new content IS a candidate for an `Updates` relation — even if it should rank low in agent retrieval.

**`FindCandidates` is a similarity aid, not a relation recommendation engine.** High cosine similarity does not imply a correct or useful relation. Two memories with cosine > 0.95 may be duplicates (`Updates`), complementary context (`Extends`), or simply about the same topic with no structured relationship. The caller inspects the candidates and decides. Do not treat high-scoring results as automatic `Updates` targets — incorrect `Updates` silently retire knowledge that was still valid.

`Recall` and `GetProfile.Episodic` use the same scoring formula. `GetProfile` bundles Static and Episodic into separate layers, applies `ProfileMaxStatic` and `ProfileMaxEpisodic` caps, and is the single call for agent initialization. `Recall` is the general-purpose query for mid-task lookups.

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
                × trust(m.Actor.ID)           ← resolved from SpaceWritePolicy at query time
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

**Reactivation**: use `Revive` to reactivate an inactive lineage, or create a fresh root to start unrelated knowledge.

`Revive` is an explicit operation — not a special case inside `Updates`. The `Updates` invariant is clean: it never targets a forgotten memory, period.

```go
// Revive creates a new IsLatest=true memory as the successor to an inactive lineage.
// rootID must identify a lineage whose latest is IsForgotten=true (inactive).
// Returns ErrLineageActive if the lineage has a non-forgotten latest.
// The new memory's ParentID = the forgotten latest's ID; RootID = rootID.
// The forgotten memory is NOT un-forgotten; its IsForgotten status is unchanged.
// Kind, Actor, Confidence must be provided in ReviveInput; they are not inherited.
// Recorded in audit_log with operation=revive.
Revive(ctx context.Context, rootID string, input ReviveInput) (*Memory, error)
```

```go
type ReviveInput struct {
    Content    string
    Kind       MemoryKind // must match the Kind of the root memory
    Actor      Actor
    Confidence float32
    Metadata   map[string]any
}
```

**Why `Kind` must match the lineage root**: a lineage has a single semantic class established at its first write. Allowing `Revive` to change it would silently bypass `StaticWriters` governance (an Episodic lineage reactivated as Static would skip the promotion audit trail and the `PromotePolicy` check). To evolve knowledge across kinds, use the promotion path: create a new `KindStatic` memory via `Remember` with `StaticWriters` permission, optionally with an `Extends` relation pointing to the old lineage root for historical linkage. That path creates a governed write with explicit actor attribution and an audit record.

To permanently bury an inactive lineage, do nothing — it remains in cold storage, invisible to retrieval, accessible only via `GetLineage`. There is no "delete lineage" operation.

---

## Memory Governance

Without governance, the static layer accumulates unboundedly. Every agent writes "important facts" and three months later the context window is full of stale, conflicting, low-quality static memories. This section defines the mechanisms that prevent this.

### SpaceWritePolicy

Each space has a `SpaceWritePolicy` configured at construction time. It governs who can write each Kind.

```go
type SpaceWritePolicy struct {
    // Default trust levels by ActorKind.
    // Override per-actor via TrustLevels.
    HumanTrust  float32 // default: 1.0
    SystemTrust float32 // default: 1.0
    DefaultAgentTrust float32 // default: 0.7

    // Per-actor trust overrides. Applied regardless of ActorKind.
    // Use to lower trust for a specific human or system actor, or raise it for a specific agent.
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

    // ── Corpus limits (governance) ─────────────────────────────────────────
    // Hard caps on how many IsLatest=true memories can exist per Kind in this space.
    // Remember returns ErrCorpusLimit when a cap is breached.
    // These govern storage growth. They are not retrieval limits.
    MaxStaticMemories   int // default: 500;  includes KindDerived
    MaxEpisodicMemories int // default: 10_000

    // ── Profile limits (context assembly) ──────────────────────────────────
    // Default caps for GetProfile. These bound how much is returned to the agent,
    // not how much is stored. Callers can override per-request via ProfileRequest.
    // ErrCorpusLimit is never triggered by these limits.
    ProfileMaxStatic   int // default: 50  — KindStatic + KindDerived in GetProfile.Static
    ProfileMaxEpisodic int // default: 10  — KindEpisodic in GetProfile.Episodic
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

### Corpus limits vs profile limits

**Corpus limits** (`MaxStaticMemories`, `MaxEpisodicMemories`) are hard caps on how many `IsLatest=true` memories can exist per Kind in a space. On breach, `Remember` returns `ErrCorpusLimit`. The caller must `Forget` before writing. These govern storage growth.

**Profile limits** (`ProfileMaxStatic`, `ProfileMaxEpisodic`) cap how many memories appear in `GetProfile`. They do not trigger `ErrCorpusLimit`. They govern context assembly.

The two limits are independent by design. The corpus can hold 500 static memories while `GetProfile` returns only the top 50 by relevance. Hitting `ErrCorpusLimit` means the corpus has not been curated — a governance failure. Hitting `ProfileMaxStatic` during retrieval is normal operation.

Corpus limits are pressure valves: they surface governance failures immediately rather than letting the corpus silently degrade. An agent that hits `ErrCorpusLimit` must curate. This is intentional.

### Preventing garbage: summary

| Mechanism | What it prevents |
|---|---|
| `ForgetAfter` TTL on Episodic | Time-bounded facts accumulating after expiry |
| `Updates` chain discipline | Contradictory versions of the same fact competing in retrieval |
| `StaticWriters` policy | Agents promoting every observation to a permanent fact |
| `Confidence × trust(actor)` in scoring | Low-quality writes by untrusted actors ranking alongside trusted ones |
| `MaxStaticMemories` corpus limit | Silent static corpus bloat (surfaces as `ErrCorpusLimit`) |
| `ProfileMaxStatic` profile limit | Context window overflow from an ungoverned static layer |
| `HasOrphanedSources` flag | Silent inclusion of Derived memories with partially invalidated evidence |
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
                   Stores EmbeddingModelID, dimension, half-life H, DefaultWeight, write policy.

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

**Not a reputation system.** Trust levels are static configuration in `SpaceWritePolicy`, resolved at query time. They do not update based on observed agent behavior. Dynamic trust is future work contingent on evidence that static trust is insufficient.

---

## Design Decisions

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| Storage engine | bbolt | PostgreSQL, SQLite+sqlite-vec, Redis | Zero external processes; pure Go; ACID; etcd-proven; single file backup |
| Vector search | Brute-force cosine | HNSW, pgvector, FAISS | Avoids CGO entirely; HNSW is an upgrade path, not a starting point |
| Relation inference | None — caller-explicit | Automatic inference on write | Automatic inference makes `IsLatest` mutations non-auditable; `FindCandidates` provides search without making the decision |
| Derived sources | Episodic/Static only, not Derived | Allow Derived-from-Derived | Prevents compounding inference errors; higher-order knowledge must pass through Static promotion (governed write) |
| Derived evolution | Updates allowed with fresh SourceIDs | Forbid all Updates on Derived | A synthesis can be refined by better evidence; forbidding evolution forces Forget+recreate which loses lineage |
| TrustLevel storage | Query-time from policy (not persisted) | Persisted snapshot at write time | Policy changes must retroactively affect old memories; persisted snapshots make policy changes ineffective for existing records |
| Trust model | Static config in SpaceWritePolicy | Dynamic reputation | Minimal and auditable; dynamic trust requires evidence that static is insufficient |
| Corpus vs profile limits | Two separate limits (MaxXxx vs ProfileMaxXxx) | Single MaxXxx serving both | Conflating governance (corpus growth) with context assembly (agent window) creates confusing semantics |
| Derived + forgotten sources | `HasOrphanedSources` flag + caller decides | Auto-invalidate on source forget | Auto-invalidation is destructive; a pattern across past events does not become false because one event record is cleaned up |
| Lineage reactivation | Explicit `Revive` operation | Exception inside Updates semantics | Hidden exceptions inside invariants are technical debt; explicit operation is auditable and has clear error semantics |
| FindCandidates defaults | Live-only; opt-in for historical | Always include all corpus | Safe by default; callers who need archaeological access are explicit about it |
| Relations in retrieval | Not traversed | Graph-expanded retrieval | Traversal creates unpredictable result sets; callers use MemoryInspector for explicit graph walks |
| Multi-space scoring | Per-space DefaultWeight in SpaceConfig + per-request override | Automatic hierarchy; all-manual weights | DefaultWeight removes caller discipline requirement for common cases; per-request override handles exceptions |
| Embedding versioning | ModelID lock + migration API | Best-effort compatibility | Silent vector space corruption is undetectable and catastrophic; hard lock forces explicit migration |
| Concurrency for Updates | Optimistic lock via `IfLatestID` | Last-write-wins, pessimistic lock | LWW silently corrupts lineage; pessimistic lock is unnecessary overhead for low-contention workload |
| Versioning model | Immutable lineage + IsLatest pointer | In-place mutation | Full history required for audit and for MemoryInspector |
| Memory kinds | Three-value enum | `IsStatic bool` | Distinct governance, scoring, and decay rules per kind; extensible |
| Governance | SpaceWritePolicy at open() time | Runtime ACL | Governance rules belong in code; runtime-mutable policy is an attack surface |

---

## Open Questions

**Dynamic trust**: TrustLevels are static configuration in `SpaceWritePolicy`. Should they update based on observed agent accuracy — e.g., an agent whose memories are frequently Forgotten or whose Derived memories are frequently invalidated? Left for v2. Requires empirical evidence that static trust is insufficient and a clear definition of what constitutes a "trust-decreasing event."

**Cross-Derived Derives**: prohibited in v1 (sources must be Episodic or Static). If this proves too restrictive, v2 can relax it with an explicit depth limit (max one level of Derived-as-source) and a required re-validation step at the intermediate.

---

## Roadmap

```
v0.1 — Core
  · Memory model: Kind, Actor (ID/Kind), Confidence, HasOrphanedSources, SourceIDs, Rationale
  · Remember, Recall, GetProfile, Forget, Revive, FindCandidates
  · Updates invariant + IsLatest exclusivity + IfLatestID optimistic lock
  · Extends and Derives with all structural invariants
  · Inactive lineage semantics + Revive operation (no exception in Updates)
  · Forget(sourceID) sets HasOrphanedSources=true on affected Derived memories atomically
  · SpaceWritePolicy: EpisodicWriters, StaticWriters, DerivedWriters, PromotePolicy
  · Corpus limits: MaxStaticMemories, MaxEpisodicMemories (ErrCorpusLimit)
  · Profile limits: ProfileMaxStatic, ProfileMaxEpisodic (context assembly caps)
  · TrustLevel resolved at query time from SpaceWritePolicy (not persisted)
  · Scoring formula: cosine × confidence × trust(actor.ID) × recency × space_weight
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
