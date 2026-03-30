# AGENTS.md

## Mission

This repository exists to build OmnethDB into a best-in-class memory system for autonomous agents.

The standard is not “good enough to demo.”

The standard is:

- semantic correctness
- explicit behavior over hidden magic
- sharp product judgment
- auditable state transitions
- operational simplicity without semantic compromise
- work that can stand up to scrutiny from strong engineers, not only enthusiastic users

We are not here to ship a generic memory demo, a vague agent cache, or a thin wrapper around embeddings.

We are here to build a serious memory primitive that deserves trust.

## What Good Looks Like

Good work in this repo makes the system:

- more correct
- more legible
- more testable
- more inspectable
- more aligned with the architecture
- more worthy of being called best-in-class

Work is not good because it “basically works.”

It is good when it:

- preserves invariants under edge cases
- keeps contracts explicit
- improves confidence in behavior, not just feature count
- reduces ambiguity for the next contributor
- raises the overall standard of the codebase

## Source Of Truth

Read these first:

1. [docs/ARCHITECTURE.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/ARCHITECTURE.md)
2. [docs/INDEX.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/INDEX.md)
3. the relevant sprint plan and ticket docs under [docs/tickets](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/tickets)

Do not invent behavior that conflicts with the architecture or planning stack.

If code, tests, and docs disagree, resolve the disagreement instead of quietly picking one and moving on.

## Working Principles

1. Preserve the product thesis.
OmnethDB is a versioned, governed, inspectable memory primitive. Do not quietly turn it into a generic vector store.

2. Keep retrieval semantics disciplined.
Live retrieval should stay predictable and explicit. Do not smuggle graph traversal or historical leakage into hot-path retrieval.

3. Treat invariants as product behavior.
Lineage rules, relation rules, write governance, and migration safety are not implementation details.

4. Prefer explicitness over cleverness.
If a flow is important, make it visible in code and tests. Hidden shortcuts create long-term semantic debt.

5. Keep planning and execution connected.
Tie work back to the docs stack: spec, capability, UAT, backlog, sprint, ticket.

6. Leave the repo clearer than you found it.
If you touch a tricky area, improve naming, structure, or docs enough that the next person can move faster.

7. Protect the standard.
Do not lower the bar because a shortcut feels faster. If something is under-specified, make it clearer. If something is brittle, make it sturdier. If something is sloppy, tighten it.

8. Optimize for trust, not cleverness theater.
This system should feel dependable. Choose designs that are explainable, testable, and durable.

9. Respect the distinction between hot path and cold path.
Fast retrieval behavior and inspection/audit behavior serve different jobs. Do not blur them for convenience.

10. Build like this repo will matter.
Because if we do this right, it will.

## Execution Expectations

When working in this repo:

- start from the relevant milestone and sprint plan
- check the matching ticket docs before implementing
- keep scope aligned to the active ticket
- update or add tests alongside behavior changes
- avoid speculative abstractions unless they clearly reduce future complexity
- prefer finishing a smaller slice correctly over touching a larger slice superficially
- if a change weakens explainability or invariant confidence, treat that as a real regression

## Quality Bar

Before considering work done, ask:

- does this align with the architecture?
- does it preserve invariants?
- is the behavior testable and tested?
- is the result understandable by the next contributor?
- does this move OmnethDB toward best-in-class quality, or only toward “done for now”?

If the answer to the last question is weak, the work probably needs another pass.

## What We Do Not Want

Do not introduce:

- fuzzy behavior that no one can explain precisely
- hidden coupling between retrieval and graph inspection
- “temporary” shortcuts that quietly become the product
- abstractions that exist only to look sophisticated
- weak tests that bless unclear semantics
- design drift away from the architecture because the local change was easier

## Default Attitude

Contribute with ambition.

Assume this repository is trying to set a standard, not follow one.

Write code, docs, and tests like they will be read by people deciding whether this system is serious.
