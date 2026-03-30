# OmnethDB V1 Release Recommendation

Decision date:

- `2026-03-30`

Recommendation:

- `ship`

Basis:

- release-blocking UAT scenarios have explicit recorded outcomes in [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md)
- traceability from backlog and capabilities to release-blocking UAT scenarios is documented in [RELEASE_TRACEABILITY_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_TRACEABILITY_V1.md)
- the release harness is defined and repeatable in [RELEASE_UAT_HARNESS_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_UAT_HARNESS_V1.md)
- the automated release command `go test ./...` is green for the current repository state

## Residual Risk Posture

- acceptable for `v1`

Residual risks:

- no dedicated operator CLI exists yet; operator workflows are currently library- and file-layout-driven
- release evidence is scenario-aligned and automated, but not yet wrapped in a separate reporting tool
- performance, scale, and post-`v1` ergonomics remain outside this release decision

## What This Means

This recommendation does not claim that OmnethDB is finished forever. It claims that the defined `v1` contract is implemented, verified, and auditable enough to ship as a serious embedded memory primitive.
