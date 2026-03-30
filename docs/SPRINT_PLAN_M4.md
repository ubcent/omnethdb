# OmnethDB V1 Sprint Plan: Milestone 4

This document turns Milestone 4 from [MILESTONE_PLAN_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/MILESTONE_PLAN_V1.md) into an execution-ready sprint plan.

Milestone 4 focus:

- `Recall`
- scoring engine
- `GetProfile`
- `FindCandidates`
- multi-space retrieval behavior

Use this together with:

- [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
- [SPEC_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPEC_MAP_V1.md)
- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
- [SPRINT_PLAN_M3.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/SPRINT_PLAN_M3.md)

## Sprint Goal

Deliver the full live-corpus retrieval contract that agents and callers will actually use during execution.

At the end of this sprint, OmnethDB should support:

- live semantic recall
- full scoring by similarity, confidence, trust, recency, and space weight
- layered profile assembly for agent context injection
- raw-cosine candidate search for relation authoring
- predictable multi-space ranking

## In Scope

Backlog items:

- `BL-026` Specify Recall, Profile, And Candidate Search Contracts
- `BL-027` Implement Recall Over Live Corpus
- `BL-028` Implement Scoring Engine
- `BL-029` Implement GetProfile Layered Retrieval
- `BL-030` Implement FindCandidates
- `BL-031` Verify Retrieval, Scoring, Profile, And Candidate Search

Primary milestone capabilities:

- `CAP-12` Recall Live Knowledge For Mid-Task Queries
- `CAP-13` Assemble A Two-Layer Context Profile
- `CAP-14` Search Candidate Memories For Relation Authoring
- `CAP-15` Query Across Multiple Spaces With Controlled Weighting

Primary UAT alignment:

- `UAT-14`
- `UAT-15`
- `UAT-16`
- `UAT-17`
- `UAT-18`

## Out Of Scope

The following remain outside Milestone 4:

- lineage inspection and graph traversal APIs
- audit query APIs
- optimistic locking and concurrent-write validation
- embedding migration workflow
- operator backup/restore flow

## Workstreams

### Workstream A. Retrieval Contract

Objective:
Define and implement the live-corpus retrieval behavior that distinguishes OmnethDB from historical or graph-expanded search.

Tasks:

- `BL-026`
- `BL-027`

Expected outputs:

- concrete `Recall` contract
- filtering against live corpus only
- non-traversal of relations during retrieval
- deterministic ranking pipeline inputs

### Workstream B. Scoring And Multi-Space Ranking

Objective:
Implement the product scoring model exactly as specified in the architecture.

Tasks:

- `BL-028`

Expected outputs:

- cosine-based ranking
- confidence weighting
- trust weighting resolved from current policy
- episodic recency decay
- multi-space weight support with request overrides

### Workstream C. Context Assembly And Candidate Search

Objective:
Support both agent initialization flows and relation-authoring support flows without conflating their semantics.

Tasks:

- `BL-029`
- `BL-030`
- `BL-031`

Expected outputs:

- `GetProfile` with static and episodic layers
- inclusion of derived memories in the static layer
- orphan filtering support
- `FindCandidates` with raw cosine only
- verification that retrieval and candidate search behave differently by design

## Suggested Execution Order

Recommended order inside the sprint:

1. `BL-026`
2. `BL-027`
3. `BL-028`
4. `BL-029`
5. `BL-030`
6. `BL-031`

Rationale:

- freeze the behavioral differences between retrieval APIs first
- implement live recall before layering on score semantics
- build profile and candidate search on top of a stable retrieval and scoring core
- verify all retrieval modes together so contract differences stay explicit

## Suggested Ticket Breakdown

If this sprint is turned into tickets, a good minimum split is:

1. recall live-corpus filtering and no-graph-traversal behavior
2. scoring engine and trust/recency factors
3. multi-space merge and weight overrides
4. `GetProfile` layering and top-k limits
5. `FindCandidates` raw similarity search
6. retrieval verification suite

## Definition Of Done

Milestone 4 sprint is done only when all of the following are true:

1. `Recall` returns only latest, unforgotten, unexpired memories from requested spaces.
2. Retrieval does not implicitly traverse `Updates`, `Extends`, or `Derives`.
3. Final score reflects cosine, confidence, trust, recency, and space weights.
4. `GetProfile` returns independently capped static and episodic layers.
5. The static layer includes derived memories.
6. `FindCandidates` uses raw cosine only and can explicitly include historical records.
7. Retrieval and candidate search have separate, test-proven semantics.

## Sprint Risks

### Risk 1. Smuggling Graph Semantics Into Retrieval

Why it matters:
It is tempting to surface related memories automatically, but that would violate a central architectural contract.

Mitigation:

- keep retrieval pipeline independent from relation traversal code
- test that related-but-irrelevant memories are not surfaced automatically

### Risk 2. Treating Scoring As An Implementation Detail

Why it matters:
The scoring formula is part of the product promise. Small shortcuts will alter acceptance behavior.

Mitigation:

- implement each score factor explicitly
- design verification cases that isolate confidence, trust, recency, and space-weight effects

### Risk 3. Letting FindCandidates Drift Toward Recall

Why it matters:
If `FindCandidates` starts using the same ranking semantics as `Recall`, callers lose a critical authoring tool.

Mitigation:

- keep `FindCandidates` raw-cosine-only
- verify behavioral divergence from `Recall` with the same candidate corpus

### Risk 4. Blurring Corpus Limits And Profile Limits

Why it matters:
Confusing storage governance and retrieval shaping leads to subtle user-facing bugs.

Mitigation:

- test independent top-k behavior in `GetProfile`
- keep profile-layer limits separate from write-time limit enforcement

## Review Checklist

Before closing the sprint, review:

- can historical or forgotten memories ever leak into `Recall`?
- do derived memories show up in the static profile layer?
- does lowering actor trust immediately affect ranking?
- do run-scoped and project-scoped spaces merge predictably under weighting?
- is `FindCandidates` still useful for relation authoring rather than context retrieval?

## Next Step After This Sprint

If Milestone 4 closes successfully, the next planning artifact should be:

- `SPRINT_PLAN_M5.md`

That sprint should focus on inspection APIs, auditability, and concurrency safety.
