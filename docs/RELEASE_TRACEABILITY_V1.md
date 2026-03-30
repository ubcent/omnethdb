# OmnethDB V1 Release Traceability Matrix

This document closes the release-facing traceability loop for `v1`.

It maps release-blocking UAT scenarios to:

- capabilities
- backlog items
- automated verification evidence
- release evidence recording points

Use this together with:

- [UAT_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/UAT_MAP_V1.md)
- [CAPABILITY_MAP_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/CAPABILITY_MAP_V1.md)
- [BACKLOG_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/BACKLOG_V1.md)
- [RELEASE_UAT_HARNESS_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_UAT_HARNESS_V1.md)
- [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md)

## Release-Blocking Matrix

| UAT | Capability | Backlog | Automated evidence | Release evidence |
| --- | --- | --- | --- | --- |
| `UAT-01` Bootstrap A New Space On First Write | `CAP-01` | `BL-001`, `BL-004`, `BL-005` | `TestEnsureSpaceBootstrapsAndPersistsConfig`, `TestRememberCreatesRootMemoryAsLatest` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-05` Supersede Current Knowledge With An Update | `CAP-04`, `CAP-19` | `BL-007`, `BL-011` | `TestRememberUpdateSwitchesLatestWithinLineage`, `TestGetLineageReturnsFullHistoryIncludingForgottenAndRevived` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-07` Create A Derived Memory From Valid Sources | `CAP-06`, `CAP-20` | `BL-009`, `BL-011` | `TestRememberCreatesDerivedMemoryWithProvenance` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-10` Forget A Live Memory | `CAP-08` | `BL-012` | `TestForgetLatestDeactivatesLineageWithoutRestoringPreviousVersion` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-12` Revive An Inactive Lineage | `CAP-10` | `BL-013` | `TestReviveCreatesNewLatestForInactiveLineage` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-13` Mark Derived Memories With Orphaned Sources | `CAP-11` | `BL-014` | `TestForgetMarksAffectedLiveDerivedMemoryAsOrphanedAndKeepsItLive`, `TestOrphanedSourceFlagIsNotClearedBySourceRevive` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-14` Recall Only Live Current Knowledge | `CAP-12` | `BL-016` | `TestRecallReturnsOnlyLiveCurrentKnowledge`, `TestRecallDoesNotTraverseRelations` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-15` Rank Recall Results Using Full Scoring Rules | `CAP-13` | `BL-017` | `TestRecallScoresByConfidenceAndTrust`, `TestRecallAppliesEpisodicRecencyAndSpaceWeights` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-16` Build A Two-Layer Agent Profile | `CAP-14` | `BL-018` | `TestGetProfileReturnsSeparateStaticAndEpisodicLayers`, `TestGetProfileUsesIndependentLayerLimits` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-19` Enforce Write Permissions By Kind | `CAP-17` | `BL-010` | `TestRememberRejectsUnauthorizedStaticWrite`, `TestRememberAllowsPromotionOnlyWithPromotePolicy` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-20` Apply Trust Policy Retroactively In Retrieval | `CAP-18` | `BL-017`, `BL-010` | `TestRecallUsesCurrentPolicyTrustAtQueryTime` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-21` Enforce Corpus Limits Independently From Profile Limits | `CAP-17` | `BL-010`, `BL-018` | `TestRememberRejectsRootWriteWhenCorpusLimitReached`, `TestRememberRejectsPromotionWhenStaticCorpusLimitReached`, `TestGetProfileUsesIndependentLayerLimits` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-24` Explain Corpus State Through Audit History | `CAP-21` | `BL-027`, `BL-028`, `BL-038` | `TestGetAuditLogReturnsSpaceScopedChronologicalHistory`, `TestInspectionAndAuditCanExplainCurrentCorpusState` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-25` Reject Conflicting Concurrent Updates | `CAP-22` | `BL-039`, `BL-041` | `TestRememberRejectsUpdateConflictWhenIfLatestIDIsStale`, `TestConcurrentUpdatesProduceOneWinnerAndOneConflict` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |
| `UAT-27` Migrate Embeddings Without Mixing Vector Spaces | `CAP-23` | `BL-042`, `BL-043`, `BL-044` | `TestBeginEmbeddingMigrationRejectsWritesButKeepsReadsAvailable`, `TestMigrateEmbeddingsReembedsLiveAndHistoricalMemories`, `TestMigrateEmbeddingsCanResumeFromPersistedMigratingState` | [RELEASE_EVIDENCE_V1.md](/Users/dmitrybondarchuk/Projects/my/omnethdb/docs/RELEASE_EVIDENCE_V1.md) |

## Evidence Model

For each release-blocking UAT scenario, release evidence must record:

- scenario ID
- outcome: `pass`, `fail`, or `partial`
- command or harness used
- automated evidence references
- residual risk or defect notes, if any

## Coverage Statement

The current `v1` release gate is intentionally built on automated scenario-aligned tests, not on a separate manual demo script. This keeps release evidence tied to repeatable execution rather than memory or presentation flow.
