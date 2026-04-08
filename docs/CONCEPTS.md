# OmnethDB Concepts

This document explains the product model in plain language.

If you remember only one sentence, remember this:

OmnethDB stores project knowledge with explicit semantics, so agents can retrieve the current truth instead of guessing from a pile of similar text.

## The Core Mental Model

A memory is not just text.
It is text plus meaning:

- what kind of knowledge it is
- who wrote it
- how confident they were
- whether it superseded something older
- whether it depends on source memories
- whether it was later forgotten

That extra structure is the whole point.

## Memory Kinds

### `static`

Stable project knowledge.

Examples:

- "payments use signed cursor pagination"
- "calendar functionality lives inside palantir"
- "refresh token rotation is intentionally disabled"

Use `static` for truths you want future agents to rely on.

### `episodic`

A concrete event, incident, or observation.

Examples:

- "deploy failed on 2026-04-07 because smoke test was skipped"
- "build broke after upgrading grpc-go"

Use `episodic` when recording what happened, not what is always true.

### `derived`

A synthesized pattern built from multiple source memories.

Examples:

- "all DB migrations require a smoke test before production deploy"

`derived` is stricter on purpose:

- it needs at least two source memories
- sources must be current at write time
- it requires a rationale
- it is not allowed to chain from another `derived` memory

## Lineage

When reality changes, OmnethDB does not overwrite the old fact silently.
It creates a new version in the same lineage.

That gives you two things at once:

- the latest truth for retrieval
- the old truth for audit and history

Think of lineage as "version history for one fact."

## `Updates` vs `Extends`

This distinction matters a lot.

### `Updates`

Use this when the new memory replaces the old one.

Example:

- old: "deploy takes 5 minutes"
- new: "deploy takes 12 minutes after auth middleware was added"

Only one memory in the lineage is latest.

### `Extends`

Use this when the new memory adds context without contradicting the old one.

Example:

- base fact: "payments use signed cursor pagination"
- extension: "admin exports still use offset pagination for CSV compatibility"

Both can remain valid at the same time.

## `Forget`

Forget does not mean delete from history.

It means:

- this memory is no longer live
- retrieval should stop treating it as active knowledge
- audit and inspection should still be able to explain what happened

This is important because silent deletion destroys trust.

## Retrieval vs Inspection

OmnethDB keeps these jobs separate.

### Retrieval

Used when an agent needs working knowledge now.

Tools:

- `recall`
- `profile`

Goal:

- return live, relevant knowledge
- prefer current truth
- avoid contaminating the answer with stale history

### Inspection

Used when a human or agent needs to understand structure or history.

Tools:

- `lineage`
- `related`
- `audit`
- `export`

Goal:

- explain how knowledge evolved
- inspect graph links
- audit lifecycle transitions

## Trust And Governance

Every memory has a `Confidence`.
That is the writer saying, "how sure am I?"

The space policy also has trust settings.
That is the system saying, "how much do I trust this actor?"

Those are different things.

- confidence describes the fact
- trust describes the writer

OmnethDB also governs who is allowed to write which kind of memory.

Default shape:

- humans and system actors can write `episodic`
- humans and system actors can write `static`
- humans and agents can write `derived`
- promotion is human-governed by default

The runtime config can tighten that policy.

## Spaces

A space is a memory namespace.

Typical example:

- `repo:company/app`

You can also have multiple spaces and query them together.
Space weights let you keep a noisy run-scoped space from drowning out a project-scoped one.

## Why This Is Better Than A Flat Vector Store

A flat vector store can tell you which notes are similar.
It cannot tell you, by itself:

- which fact superseded which
- which memory is live
- whether a statement is a stable rule or a one-off incident
- whether a synthesis is auditable
- whether a writer was allowed to create that memory kind

OmnethDB is built around those semantics directly.

## Good Usage

- read before writing
- update existing facts instead of restating them
- store stable truths as `static`
- store incidents as `episodic`
- derive only when a real pattern exists across multiple sources

## Bad Usage

- using OmnethDB like a dump of random notes
- writing speculative guesses as durable facts
- relying on relation traversal to "magically enrich" retrieval
- treating history as disposable

## Next Step

Read [SETUP.md](./SETUP.md) to configure a workspace correctly.
