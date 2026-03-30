# OmnethDB Full-Spec V1 UAT Map

This document defines the User Acceptance Testing map for `v1`.

It translates the capability outcomes in [CAPABILITY_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/CAPABILITY_MAP_V1.md) into acceptance scenarios that can serve as:

- release gates
- demo flows
- stakeholder validation cases
- traceability anchors for implementation and verification

Use this together with:

- [ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
- [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
- [CAPABILITY_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/CAPABILITY_MAP_V1.md)

## Purpose

UAT in OmnethDB should validate externally meaningful system behavior, not just low-level implementation details.

Each scenario below is written to answer:

- can the system do the intended job?
- is the result observable from the outside?
- does it preserve the core architectural promise?

## Structure

Each UAT scenario includes:

- objective: what the scenario proves
- actors: who participates
- setup: the minimum initial state
- flow: the user-visible sequence
- expected outcome: what must be true for acceptance
- traces to capabilities: the capability IDs covered

## UAT Scenario Groups

### Group 1. Space Bootstrap And Configuration

#### UAT-01. Bootstrap A New Space On First Write

Objective:
Prove that a new space can be created implicitly and immediately used.

Actors:

- agent
- operator

Setup:

- no existing space for target `SpaceID`
- a valid embedder and write policy are configured

Flow:

1. Write a new root memory into a previously unseen `SpaceID`
2. Retrieve the memory from that same space
3. Inspect the space configuration

Expected outcome:

- first write succeeds without explicit space creation
- the memory is immediately available in live retrieval
- the space stores locked embedding model ID and dimensions

Traces to capabilities:

- `CAP-01`

#### UAT-02. Reject Writes With A Mismatched Embedder

Objective:
Prove that a space cannot silently mix incompatible embedding spaces.

Actors:

- operator
- system

Setup:

- a space already exists with locked embedder identity

Flow:

1. Attempt a write to the existing space using a different embedder model or dimension
2. Query the space again using the valid embedder

Expected outcome:

- mismatched write is rejected
- the existing corpus remains unchanged
- subsequent valid retrieval still works

Traces to capabilities:

- `CAP-01`
- `CAP-23`

#### UAT-03. Apply Per-Space Defaults During Retrieval

Objective:
Prove that space-specific config affects retrieval behavior in observable ways.

Actors:

- operator
- agent

Setup:

- two spaces exist with different default weights or decay settings
- both contain relevant memories

Flow:

1. Run the same retrieval against both spaces together
2. Compare result ordering
3. Override a weight at request time and repeat

Expected outcome:

- default per-space configuration affects ranking
- request overrides take precedence over defaults
- behavior is consistent with configured weighting and decay

Traces to capabilities:

- `CAP-02`
- `CAP-15`

### Group 2. Author And Evolve Knowledge

#### UAT-04. Store A New Root Memory

Objective:
Prove that a fresh memory can enter the live corpus as a new lineage.

Actors:

- agent
- human

Setup:

- an existing writable space

Flow:

1. Write a new root memory
2. Query the space for that content
3. Inspect the stored record

Expected outcome:

- write succeeds
- memory is visible in live retrieval
- record has valid initial lineage and lifecycle fields

Traces to capabilities:

- `CAP-03`

#### UAT-05. Supersede Current Knowledge With An Update

Objective:
Prove that the system can replace current knowledge without losing history.

Actors:

- agent
- human

Setup:

- a space with one live root memory

Flow:

1. Write a second memory that `Updates` the current latest
2. Query live retrieval
3. Inspect the lineage history

Expected outcome:

- the new memory becomes the only latest version
- the previous memory is no longer returned in live retrieval
- both versions remain visible in lineage inspection

Traces to capabilities:

- `CAP-04`
- `CAP-19`

#### UAT-06. Add Context With Extends Without Changing Live State

Objective:
Prove that contextual enrichment does not behave like replacement.

Actors:

- agent
- human

Setup:

- two valid live memories in the same space

Flow:

1. Write a new memory using `Extends` against another live memory
2. Query for both memories
3. Inspect graph relations

Expected outcome:

- both memories remain live
- the relation is visible through inspection
- retrieval does not auto-expand from one memory to the other via the relation alone

Traces to capabilities:

- `CAP-05`
- `CAP-20`

#### UAT-07. Create A Derived Memory From Valid Sources

Objective:
Prove that the system can store synthesized knowledge with explicit provenance.

Actors:

- agent
- human

Setup:

- at least two latest non-derived source memories in the same space

Flow:

1. Write a derived memory with valid `SourceIDs` and non-empty rationale
2. Query retrieval for the derived knowledge
3. Inspect graph relations and stored provenance

Expected outcome:

- derived write succeeds
- derived memory is live and retrievable
- provenance and rationale are preserved and inspectable

Traces to capabilities:

- `CAP-06`
- `CAP-20`

#### UAT-08. Reject Invalid Derived Writes

Objective:
Prove that derived provenance rules are enforced and not best-effort.

Actors:

- agent

Setup:

- a writable space with existing memories

Flow:

1. Attempt a derived write with one source only
2. Attempt a derived write with a derived source
3. Attempt a derived write without rationale

Expected outcome:

- each invalid write is rejected
- no partial records are created
- existing corpus remains unchanged

Traces to capabilities:

- `CAP-06`
- `CAP-16`

#### UAT-09. Promote Episodic Knowledge Into Static Knowledge

Objective:
Prove that durable knowledge is introduced through governed promotion rather than in-place mutation.

Actors:

- human
- system

Setup:

- an episodic memory exists
- actor with promotion permission exists

Flow:

1. Write a static memory that updates the episodic source
2. Query live retrieval and profile
3. Inspect lineage history

Expected outcome:

- static write succeeds only for authorized actor
- the static memory becomes live current knowledge
- the original episodic memory remains historical rather than changing kind in place

Traces to capabilities:

- `CAP-07`
- `CAP-16`

### Group 3. Retire And Reactivate Knowledge

#### UAT-10. Forget A Live Memory

Objective:
Prove that a memory can be removed from the live corpus without deleting its history.

Actors:

- human
- system

Setup:

- an existing live memory

Flow:

1. Forget the live memory with actor and reason
2. Query live retrieval
3. Inspect historical and audit data

Expected outcome:

- forgotten memory is no longer returned in live retrieval
- historical record still exists
- actor and reason are audit-visible

Traces to capabilities:

- `CAP-08`
- `CAP-21`

#### UAT-11. Deactivate A Lineage By Forgetting Its Latest Version

Objective:
Prove that a lineage can become inactive and disappear from the live corpus.

Actors:

- human

Setup:

- a lineage with a live latest version and at least one historical version

Flow:

1. Forget the current latest version
2. Query live retrieval
3. Inspect lineage state

Expected outcome:

- no version from that lineage is returned as live
- older versions are not automatically restored
- lineage remains inspectable as historical data

Traces to capabilities:

- `CAP-09`
- `CAP-19`

#### UAT-12. Revive An Inactive Lineage

Objective:
Prove that inactive knowledge can be explicitly reactivated through a new latest version.

Actors:

- agent
- human

Setup:

- an inactive lineage exists

Flow:

1. Call `Revive` with valid input
2. Query live retrieval
3. Inspect lineage and audit history

Expected outcome:

- revive succeeds only on inactive lineage
- revived memory becomes live
- forgotten predecessor remains forgotten
- audit log shows the revive operation

Traces to capabilities:

- `CAP-10`
- `CAP-21`

#### UAT-13. Mark Derived Memories With Orphaned Sources

Objective:
Prove that forgetting a source updates provenance state without silently deleting derived knowledge.

Actors:

- human
- system

Setup:

- a live derived memory references multiple live sources

Flow:

1. Forget one of the source memories
2. Query retrieval with default settings
3. Query retrieval with orphaned derives excluded
4. Inspect the derived memory record

Expected outcome:

- derived memory remains live by default
- derived memory is marked `HasOrphanedSources=true`
- exclusion flag removes it from retrieval when requested

Traces to capabilities:

- `CAP-11`
- `CAP-13`

### Group 4. Retrieval And Context Assembly

#### UAT-14. Recall Only Live Current Knowledge

Objective:
Prove that retrieval excludes superseded, forgotten, and expired memories.

Actors:

- agent

Setup:

- a space contains latest, superseded, forgotten, and expired memories

Flow:

1. Run `Recall` for a relevant query
2. Inspect the returned result set

Expected outcome:

- only live latest unforgotten unexpired memories are returned
- historical and expired records are absent from results

Traces to capabilities:

- `CAP-12`

#### UAT-15. Rank Recall Results Using Full Scoring Rules

Objective:
Prove that retrieval ranking reflects confidence, trust, recency, and space weighting in addition to similarity.

Actors:

- agent
- operator

Setup:

- multiple relevant memories exist with deliberately different confidence, actor trust, age, and space weight conditions

Flow:

1. Run `Recall` with a query matching all candidate memories
2. Compare result order against expected ranking

Expected outcome:

- ranking follows the documented scoring model
- older episodic memories decay
- static and derived memories do not decay
- lower-trust actors rank lower given comparable relevance

Traces to capabilities:

- `CAP-12`
- `CAP-17`

#### UAT-16. Build A Two-Layer Agent Profile

Objective:
Prove that context assembly separates durable knowledge from episodic observations.

Actors:

- agent
- system

Setup:

- a space contains relevant static, derived, and episodic memories

Flow:

1. Run `GetProfile` for a relevant query
2. Inspect the returned profile structure

Expected outcome:

- profile contains separate static and episodic sections
- static section includes static and derived memories
- each section is capped and ordered independently

Traces to capabilities:

- `CAP-13`

#### UAT-17. Search Candidate Memories For Relation Authoring

Objective:
Prove that the system can help author relations without asserting their semantics.

Actors:

- agent
- human

Setup:

- a space contains multiple semantically similar live and historical memories

Flow:

1. Run `FindCandidates` for a new content string
2. Repeat with historical inclusion enabled

Expected outcome:

- results are ranked by raw cosine only
- historical results appear only when explicitly requested
- the API does not mutate data or imply relation choice

Traces to capabilities:

- `CAP-14`

#### UAT-18. Retrieve Across Multiple Spaces With Weight Control

Objective:
Prove that cross-space retrieval works without noisy spaces dominating results.

Actors:

- agent
- system

Setup:

- multiple spaces contain relevant memories
- spaces have distinct default weights

Flow:

1. Run a multi-space retrieval
2. Compare ordering with default weights
3. Re-run with explicit request-time overrides

Expected outcome:

- results are merged across spaces into one ordered set
- weighting affects final ranking
- request override behavior is observable

Traces to capabilities:

- `CAP-15`
- `CAP-02`

### Group 5. Governance And Trust

#### UAT-19. Enforce Write Permissions By Kind

Objective:
Prove that different kinds of knowledge are writable only by permitted actors.

Actors:

- operator
- agent
- human
- system

Setup:

- policy restricts static and promotion writes

Flow:

1. Attempt writes with authorized and unauthorized actors across kinds
2. Compare results

Expected outcome:

- unauthorized writes are rejected
- authorized writes succeed
- policy behavior is consistent across episodic, static, and derived flows

Traces to capabilities:

- `CAP-16`

#### UAT-20. Apply Trust Policy Retroactively In Retrieval

Objective:
Prove that retrieval ranking responds to current trust policy rather than stored historical trust snapshots.

Actors:

- operator
- agent

Setup:

- an actor has already written memories into a space
- trust policy is then changed for that actor

Flow:

1. Run retrieval before changing trust policy
2. Lower trust for the actor
3. Run the same retrieval again

Expected outcome:

- the same stored memories rank differently after policy change
- no memory rewrite is needed for ranking to change

Traces to capabilities:

- `CAP-17`

#### UAT-21. Enforce Corpus Limits Independently From Profile Limits

Objective:
Prove that storage governance and context assembly are separate constraints.

Actors:

- operator
- system

Setup:

- a space configured with known corpus and profile limits

Flow:

1. Write up to the live corpus limit
2. Attempt one additional live write
3. Run `GetProfile`

Expected outcome:

- extra write is rejected with corpus-limit behavior
- profile still returns only its own capped number of results
- profile capping does not change storage capacity

Traces to capabilities:

- `CAP-18`
- `CAP-13`

### Group 6. Audit And Inspection

#### UAT-22. Inspect Full Lineage History

Objective:
Prove that lineage inspection exposes full history, not just the live record.

Actors:

- developer
- operator

Setup:

- a lineage with multiple historical versions and at least one forgotten or superseded state

Flow:

1. Request lineage history for the root
2. Review returned versions in order

Expected outcome:

- all versions are returned
- chronological order is preserved
- historical states remain accessible even when absent from retrieval

Traces to capabilities:

- `CAP-19`

#### UAT-23. Traverse Explicit Memory Relations

Objective:
Prove that graph inspection can surface linked records without changing retrieval behavior.

Actors:

- developer
- operator

Setup:

- memories linked through `Updates`, `Extends`, and `Derives`

Flow:

1. Traverse relations from a selected memory
2. Repeat with depth-limited traversal
3. Compare with retrieval behavior for the same memory content

Expected outcome:

- relation traversal returns linked memories regardless of live status
- depth behavior is explicit and bounded
- retrieval still does not auto-traverse those relations

Traces to capabilities:

- `CAP-20`
- `CAP-12`

#### UAT-24. Explain Corpus State Through Audit History

Objective:
Prove that the system can explain why current knowledge looks the way it does.

Actors:

- operator
- developer

Setup:

- a space with writes, updates, forgets, and revive operations

Flow:

1. Request audit history for the space
2. Correlate audit events with current lineage and retrieval state

Expected outcome:

- all major lifecycle operations are present in the audit trail
- forget reason and actor are visible
- current corpus state can be reconstructed from history

Traces to capabilities:

- `CAP-21`

### Group 7. Reliability And Operations

#### UAT-25. Reject Conflicting Concurrent Updates

Objective:
Prove that concurrent writers cannot silently corrupt latest-version semantics when optimistic locking is used.

Actors:

- agent
- system

Setup:

- two writers target the same current latest memory

Flow:

1. Both writers read the same latest version
2. Both attempt to write an update with `IfLatestID`
3. Inspect final lineage and retrieval state

Expected outcome:

- one update succeeds
- the conflicting update is rejected
- lineage retains a single latest version

Traces to capabilities:

- `CAP-22`

#### UAT-26. Keep Reads Consistent During Concurrent Writes

Objective:
Prove that retrieval sees coherent snapshots even while writes are in flight.

Actors:

- agent
- system

Setup:

- an active space under concurrent read and write activity

Flow:

1. Trigger retrieval during a write transaction
2. Compare observed result set before and after commit

Expected outcome:

- retrieval sees either pre-write or post-write state
- retrieval never returns partial transitional state

Traces to capabilities:

- `CAP-22`

#### UAT-27. Migrate Embeddings Without Mixing Vector Spaces

Objective:
Prove that embedding migration is safe, explicit, and operationally controlled.

Actors:

- operator
- system

Setup:

- a populated space exists under one embedder

Flow:

1. Start embedding migration
2. Attempt writes during migration
3. Run retrieval during migration
4. Complete migration and retry normal writes

Expected outcome:

- writes are rejected while migration is active
- reads remain available
- post-migration writes succeed only with the new embedder identity
- no mixed-model state is observable

Traces to capabilities:

- `CAP-23`

#### UAT-28. Operate And Backup As A Single Embedded Store

Objective:
Prove that local operation and backup expectations are simple and stable.

Actors:

- operator
- developer

Setup:

- running system with persisted data

Flow:

1. Confirm data and config locations
2. Perform a backup of the database file
3. Restore into a clean environment and re-run basic retrieval

Expected outcome:

- persisted state is located in the documented embedded layout
- restored data is readable and usable
- operational model does not depend on external database services

Traces to capabilities:

- `CAP-24`

## Recommended Release Gates

The following UAT scenarios should be treated as release-blocking for `v1`:

- `UAT-01` Bootstrap A New Space On First Write
- `UAT-05` Supersede Current Knowledge With An Update
- `UAT-07` Create A Derived Memory From Valid Sources
- `UAT-10` Forget A Live Memory
- `UAT-12` Revive An Inactive Lineage
- `UAT-13` Mark Derived Memories With Orphaned Sources
- `UAT-14` Recall Only Live Current Knowledge
- `UAT-15` Rank Recall Results Using Full Scoring Rules
- `UAT-16` Build A Two-Layer Agent Profile
- `UAT-19` Enforce Write Permissions By Kind
- `UAT-20` Apply Trust Policy Retroactively In Retrieval
- `UAT-21` Enforce Corpus Limits Independently From Profile Limits
- `UAT-24` Explain Corpus State Through Audit History
- `UAT-25` Reject Conflicting Concurrent Updates
- `UAT-27` Migrate Embeddings Without Mixing Vector Spaces

## Traceability Guidance

Recommended traceability chain:

`Architecture -> Spec Slice -> Capability -> UAT Scenario -> Backlog Item -> Test Artifact`

Each implementation task created later should reference:

- at least one spec slice
- at least one capability
- at least one UAT scenario

This keeps the backlog aligned to externally meaningful acceptance outcomes rather than internal implementation fragments alone.
