# OmnethDB V1 Release UAT Harness

This document defines the repeatable execution process for the release-blocking `v1` acceptance suite.

## Harness Principle

The release harness for `v1` is the automated Go test suite plus the release traceability matrix.

We are deliberately not building a second parallel acceptance harness that duplicates behavior already covered by the verification suite. The repeatable release process is:

1. run the automated suite
2. map release-blocking UAT scenarios to their evidence-bearing tests
3. record pass/fail outcomes in the release evidence sheet
4. issue a release recommendation from recorded evidence

## Harness Command

Primary release command:

```bash
go test ./...
```

## Release-Blocking Scenario Set

The current release-blocking UAT set is:

- `UAT-01`
- `UAT-05`
- `UAT-07`
- `UAT-10`
- `UAT-12`
- `UAT-13`
- `UAT-14`
- `UAT-15`
- `UAT-16`
- `UAT-19`
- `UAT-20`
- `UAT-21`
- `UAT-24`
- `UAT-25`
- `UAT-27`

The traceability mapping for these scenarios lives in [RELEASE_TRACEABILITY_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_TRACEABILITY_V1.md).

## Repeatable Execution Steps

1. Ensure the workspace is on the intended `v1` candidate state.
2. Run `go test ./...`.
3. If the suite is green, mark automated evidence-backed scenarios as `pass`.
4. If the suite is red, mark affected scenarios as `fail` or `partial` and record the blocking defect.
5. Review the recorded outcomes in [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md).
6. Produce a release recommendation in [RELEASE_RECOMMENDATION_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_RECOMMENDATION_V1.md).

## Evidence Rules

- Do not treat green tests as implied sign-off. Record them.
- Do not silently waive failing scenarios.
- Do not replace scenario evidence with a narrative summary.
- Record residual risks explicitly even when all scenarios pass.

## Result

This harness is intentionally simple, auditable, and repeatable. That is the correct `v1` standard for an embedded system where correctness matters more than ceremony.
