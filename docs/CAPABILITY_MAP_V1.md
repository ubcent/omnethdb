# OmnethDB Full-Spec V1 Capability Map

This document translates the `v1` spec slices into externally visible system capabilities.

It is intended to bridge:

- architecture and spec planning
- implementation backlog
- UAT scenarios
- release acceptance

Use this together with [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md) and [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md).

## Purpose

The spec map defines what must be built.

The capability map defines what the system must be able to do from the perspective of:

- an agent writing and retrieving project memory
- an operator governing spaces and trust
- a developer auditing and debugging memory state

This is the layer that should anchor UAT.

## Capability Format

Each capability includes:

- outcome: what the user or system can accomplish
- primary actors: who exercises the capability
- UAT focus: what must be observable in acceptance testing
- key dependencies: which spec slices enable it

## Capability Areas

### A. Space Lifecycle

#### CAP-01. Create And Lock A Space On First Write

Outcome:
The system can bootstrap a new space from the first successful write and permanently lock its embedding identity.

Primary actors:

- agent
- human operator
- system

UAT focus:

- first write to a new `SpaceID` succeeds without explicit pre-creation
- the created space stores embedder model ID and dimensions
- subsequent writes with a mismatched embedder are rejected

Key dependencies:

- Space Bootstrap And Configuration Locking
- Domain Model And Error Contract

#### CAP-02. Apply Per-Space Runtime Behavior

Outcome:
Different spaces can operate with different defaults for trust, decay, limits, and weighting.

Primary actors:

- operator
- system

UAT focus:

- configured half-life affects episodic ranking
- configured default weight influences multi-space retrieval
- configured corpus and profile limits are enforced

Key dependencies:

- Operational Config And Disk Layout
- Write Governance And Promotion
- Scoring Engine

### B. Knowledge Writing

#### CAP-03. Write A New Root Memory

Outcome:
A caller can persist a new fact or event as a fresh lineage in a space.

Primary actors:

- agent
- human
- system

UAT focus:

- root memory is stored with correct kind, actor, confidence, and lifecycle state
- root is visible to retrieval immediately after commit
- invalid writes are rejected without partial state

Key dependencies:

- Domain Model And Error Contract
- Remember Write Path
- Live Corpus Storage Layout

#### CAP-04. Update Existing Knowledge Without Corrupting History

Outcome:
A caller can supersede current knowledge while preserving lineage history and keeping exactly one latest version.

Primary actors:

- agent
- human
- system

UAT focus:

- updating current latest retires the previous version atomically
- historical versions remain inspectable
- invalid update targets are rejected

Key dependencies:

- Lineage And Versioning Semantics
- Relation Validation Semantics
- Remember Write Path

#### CAP-05. Add Non-Contradictory Context

Outcome:
A caller can link a new memory as additional context using `Extends` without changing the target's live state.

Primary actors:

- agent
- human

UAT focus:

- `Extends` relation is stored when both endpoints are valid live memories
- target remains live and unchanged
- relation is visible via inspection but does not affect retrieval expansion

Key dependencies:

- Relation Validation Semantics
- Inspector And Graph Traversal

#### CAP-06. Create A Derived Memory From Multiple Sources

Outcome:
A caller can synthesize higher-level knowledge from multiple current source memories with explicit rationale.

Primary actors:

- agent
- human

UAT focus:

- derived write requires two or more valid non-derived latest sources
- rationale is mandatory
- derived memory is retrievable as live knowledge
- invalid source sets are rejected

Key dependencies:

- Derived Memory Contract
- Remember Write Path
- Relation Validation Semantics

#### CAP-07. Promote Episodic Knowledge Into Durable Static Knowledge

Outcome:
A governed actor can elevate an episodic observation into static project knowledge via an explicit write.

Primary actors:

- human
- system

UAT focus:

- promotion requires proper permission
- promoted memory becomes static latest knowledge
- original episodic memory becomes superseded rather than mutated in place

Key dependencies:

- Write Governance And Promotion
- Lineage And Versioning Semantics
- Remember Write Path

### C. Knowledge Lifecycle

#### CAP-08. Forget Live Knowledge Without Deleting History

Outcome:
A caller can remove knowledge from the live corpus while preserving cold-path history and audit data.

Primary actors:

- human
- system

UAT focus:

- forgotten memory disappears from live retrieval
- forgotten record remains available to inspection
- actor and reason are retained in audit data

Key dependencies:

- Forget And Inactive Lineages
- Auditability And Forget Records

#### CAP-09. Deactivate A Lineage By Forgetting Its Latest Version

Outcome:
The system can represent a lineage with no current truth when its latest version is forgotten.

Primary actors:

- human
- system

UAT focus:

- after forgetting latest, the lineage has no live version
- old versions are not automatically reactivated
- retrieval no longer surfaces the lineage

Key dependencies:

- Forget And Inactive Lineages
- Recall Retrieval Contract

#### CAP-10. Revive An Inactive Lineage

Outcome:
A caller can explicitly reactivate an inactive lineage with a new latest version.

Primary actors:

- agent
- human
- system

UAT focus:

- revive succeeds only on inactive lineages
- new version points to the forgotten latest
- lineage kind remains stable
- revive becomes visible in retrieval and audit history

Key dependencies:

- Revive Contract
- Forget And Inactive Lineages
- Auditability And Forget Records

#### CAP-11. Propagate Forgotten Sources Into Derived Provenance State

Outcome:
When a source memory is forgotten, dependent derived memories remain live but are marked as having orphaned provenance.

Primary actors:

- human
- system

UAT focus:

- forgetting a source marks affected live derived memories atomically
- orphaned flag remains set permanently
- retrieval can include or exclude orphaned derives based on request settings

Key dependencies:

- Orphaned Source Propagation
- Derived Memory Contract
- GetProfile Layered Retrieval
- Recall Retrieval Contract

### D. Retrieval And Context Assembly

#### CAP-12. Recall Live Knowledge For Mid-Task Queries

Outcome:
An agent can query relevant current knowledge across one or more spaces during execution.

Primary actors:

- agent
- human

UAT focus:

- only live memories are returned
- results are relevance-ordered by the documented scoring formula
- relations do not implicitly expand result sets

Key dependencies:

- Recall Retrieval Contract
- Scoring Engine
- Live Corpus Storage Layout

#### CAP-13. Assemble A Two-Layer Context Profile

Outcome:
An agent can request a context profile that separates durable project knowledge from episodic events.

Primary actors:

- agent
- system

UAT focus:

- static and episodic layers are returned separately
- each layer is capped independently
- static includes derived knowledge
- ranking within each layer follows scoring rules

Key dependencies:

- GetProfile Layered Retrieval
- Scoring Engine
- Derived Memory Contract

#### CAP-14. Search Candidate Memories For Relation Authoring

Outcome:
A caller can find semantically similar memories to help decide whether to create `Updates`, `Extends`, or no relation.

Primary actors:

- agent
- human

UAT focus:

- candidate ranking uses raw cosine only
- historical versions can be included explicitly
- the system returns candidates without making relation decisions

Key dependencies:

- FindCandidates Contract
- Live Corpus Storage Layout

#### CAP-15. Query Across Multiple Spaces With Controlled Weighting

Outcome:
A caller can combine project, workflow, or run-scoped spaces in one query without noisy spaces overwhelming the result.

Primary actors:

- agent
- system

UAT focus:

- multi-space results are merged into one ranking
- per-space default weights affect ordering
- request-time overrides can change those weights

Key dependencies:

- Space Bootstrap And Configuration Locking
- Recall Retrieval Contract
- Scoring Engine

### E. Governance And Trust

#### CAP-16. Enforce Who Can Write Which Kind Of Knowledge

Outcome:
The system can restrict write behavior by actor type, actor ID, and trust level.

Primary actors:

- operator
- system

UAT focus:

- unauthorized writers are rejected for protected kinds
- trusted and explicitly allowed actors can write
- promotion has stricter governance than ordinary episodic writes

Key dependencies:

- Write Governance And Promotion
- Domain Model And Error Contract

#### CAP-17. Rank Memories Using Current Trust Policy

Outcome:
The system can adapt retrieval quality immediately when actor trust policy changes.

Primary actors:

- operator
- agent

UAT focus:

- lowering trust for an actor decreases ranking of old and new memories from that actor
- trust is not persisted inside memory records
- human/system defaults and explicit overrides behave as documented

Key dependencies:

- Write Governance And Promotion
- Scoring Engine

#### CAP-18. Enforce Corpus Growth Limits

Outcome:
The system can stop uncontrolled memory growth and force curation when live corpora exceed policy limits.

Primary actors:

- operator
- system

UAT focus:

- writes are rejected when live corpus limits are exceeded
- profile limits do not behave like corpus limits
- forgetting or superseding memories restores available capacity

Key dependencies:

- Write Governance And Promotion
- GetProfile Layered Retrieval
- Forget And Inactive Lineages

### F. Audit And Inspection

#### CAP-19. Inspect Full Lineage History

Outcome:
A developer or operator can inspect every version in a lineage, including forgotten and superseded records.

Primary actors:

- developer
- operator

UAT focus:

- lineage returns chronological version history
- historical states are inspectable even when absent from live retrieval

Key dependencies:

- Inspector And Graph Traversal
- Lineage And Versioning Semantics

#### CAP-20. Traverse Explicit Memory Relations

Outcome:
A developer or operator can inspect how memories are linked through `Updates`, `Extends`, and `Derives`.

Primary actors:

- developer
- operator

UAT focus:

- relation traversal returns related memories regardless of live status
- depth-limited traversal behaves predictably
- retrieval and graph inspection remain separate concerns

Key dependencies:

- Inspector And Graph Traversal
- Relation Validation Semantics

#### CAP-21. Audit Why The Corpus Looks The Way It Does

Outcome:
An operator can reconstruct writes, forgets, revives, and migrations after the fact.

Primary actors:

- operator
- developer

UAT focus:

- audit log includes major lifecycle operations
- forget reason and actor are inspectable
- current corpus state can be explained through stored history

Key dependencies:

- Auditability And Forget Records
- Forget And Inactive Lineages
- Revive Contract
- Embedding Migration Workflow

### G. Reliability And Operations

#### CAP-22. Stay Correct Under Concurrent Writers

Outcome:
Multiple writers can operate safely without corrupting lineage state.

Primary actors:

- agent
- system

UAT focus:

- conflicting updates are rejected when optimistic lock is used
- no partial writes become visible
- reads observe consistent snapshots during concurrent writes

Key dependencies:

- Concurrency And Consistency
- Remember Write Path
- Live Corpus Storage Layout

#### CAP-23. Migrate A Space To A New Embedding Model Safely

Outcome:
An operator can move a space to a new embedder without mixing incompatible vectors.

Primary actors:

- operator
- system

UAT focus:

- migration write-locks the space
- reads remain available during migration
- all stored embeddings are migrated, including historical records
- interrupted migration can be rerun safely

Key dependencies:

- Embedding Migration Workflow
- Space Bootstrap And Configuration Locking
- Auditability And Forget Records

#### CAP-24. Operate As A Single-File Embedded Store

Outcome:
The system can run as an embedded local database with a simple backup model.

Primary actors:

- operator
- developer

UAT focus:

- data persists in a single database file
- runtime config and persisted data have clear locations
- backup/restore expectations are operationally simple

Key dependencies:

- Operational Config And Disk Layout
- Live Corpus Storage Layout

## Suggested UAT Coverage Matrix

The fastest way to use this document for UAT is to define one or more end-to-end scenarios per capability area.

Recommended UAT scenario groups:

1. Space bootstrap and config
Coverage:
`CAP-01`, `CAP-02`, `CAP-24`

2. Author and evolve knowledge
Coverage:
`CAP-03`, `CAP-04`, `CAP-05`, `CAP-06`, `CAP-07`

3. Retire and reactivate knowledge
Coverage:
`CAP-08`, `CAP-09`, `CAP-10`, `CAP-11`

4. Retrieve agent context
Coverage:
`CAP-12`, `CAP-13`, `CAP-14`, `CAP-15`

5. Govern memory quality
Coverage:
`CAP-16`, `CAP-17`, `CAP-18`

6. Inspect and explain system state
Coverage:
`CAP-19`, `CAP-20`, `CAP-21`

7. Operate safely under load and change
Coverage:
`CAP-22`, `CAP-23`

## UAT Design Guidance

When converting capabilities into UAT cases:

- write tests in outcome language, not storage-language first
- keep one primary assertion per UAT case
- explicitly separate live retrieval expectations from inspection expectations
- always include at least one negative case for governance and invariants
- include concurrency and migration in release acceptance, not only in lower-level test suites

## Traceability

Recommended traceability chain:

`Architecture section -> Spec slice -> Capability -> UAT scenario -> Implementation task -> Verification artifact`

This prevents drift between the system promise, the backlog, and the release bar.
