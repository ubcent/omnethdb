# OmnethDB Memory Policy

Use the `omnethdb` MCP server as durable memory for repository knowledge.

Your goal is not to save everything.
Your goal is to preserve the smallest set of high-value memories that improves future task performance without contaminating retrieval.

## Default Memory Workflow

Before doing substantial work:

1. Call `memory_profile` for the repository space.
2. If the profile is sparse, call `memory_recall` with a task-specific query.
3. Treat recalled memories as working hypotheses until confirmed against the codebase.

After finishing meaningful work:

1. Decide whether anything learned will matter in a future session.
2. Write no more than 1-3 memories unless the task truly changed multiple independent facts.
3. Prefer updating existing knowledge over adding parallel duplicates.

## Space Discipline

Use one stable `space_id` per repository:

```text
repo:<org>/<repo>
```

Do not spread one repository's knowledge across multiple spaces unless there is an explicit isolation reason.

## Memory Kind Rules

### `episodic`

Use for:

- concrete incidents
- one-off debugging observations
- task-local outcomes that may later support a broader pattern

Examples:

- "deploy failed on 2026-03-31 because migration smoke test was skipped"
- "auth e2e failed because the seeded user was missing"

Do not use `episodic` for long-lived project truths.

### `static`

Use for:

- durable repository facts
- architectural invariants
- operational rules that should be recalled in future tasks

Examples:

- "payments use signed cursor pagination"
- "all write paths must pass `IfLatestID` on contested updates"

Only write `static` when the fact is verified from code, tests, docs, or an authoritative human instruction.

### `derived`

Use for:

- a real synthesis across multiple current memories
- patterns or rules not explicitly stated in one source alone

Only create a `derived` memory when:

- there are at least two valid current sources
- the sources are `episodic` or `static`
- you can provide a concrete rationale

Do not create a `derived` memory from one source or from vague intuition.

## Update Rules

Before writing a new top-level `static` fact, ask:

- does a memory already exist for this invariant?
- did the fact change, or am I just restating it?

If the fact changed:

- use `memory_recall` to find the current fact
- use `memory_lineage` if needed
- write the new fact with `update_id`
- include `if_latest_id` when you know the current latest memory ID

Do not create sibling duplicates for the same fact.

## Confidence Rules

Use confidence to express epistemic certainty about the fact itself:

- `0.9-1.0` for directly verified code or explicit human instruction
- `0.7-0.8` for strong but not exhaustive verification
- `0.4-0.6` for tentative observations

Do not inflate confidence just because the memory feels useful.

## What Not To Store

Do not write memories for:

- temporary shell outputs
- obvious facts already implied by the current diff
- speculative guesses
- duplicated restatements of existing current knowledge
- private or sensitive content that should not persist

## End-Of-Task Checklist

Before writing to OmnethDB, confirm:

- this information will likely matter in a future task
- the chosen kind matches the semantics
- this is not duplicating an existing latest fact
- if it is a changed fact, I am using `Updates`
- if it is `Derived`, I have at least two sources and a rationale

If any answer is weak, do not write the memory.
