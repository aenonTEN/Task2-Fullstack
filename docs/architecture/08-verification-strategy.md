# Verification Strategy

## Requirement-To-Test Matrix

| Requirement Area | Verification Type | Minimum Evidence |
| --- | --- | --- |
| RBAC + data scope enforcement | Integration + API contract tests | Deny/allow cases across institution/department/team permutations |
| Resume import + dedupe merge | Integration + property tests | Duplicate by phone/ID merged correctly, no data loss |
| Explainable scoring | Unit + integration | Score within 0-100 and reason list generated |
| Qualification expiry/deactivation | Scheduled job test + integration | Expiring records highlighted, expired records deactivated |
| Purchase restrictions | Unit + integration | Prescription requirement and 7-day interval enforced |
| Case numbering + duplicate window | Unit + race/concurrency tests | Unique format and 5-minute duplicate rejection |
| Attachment chunk upload + SHA256 dedupe | Integration + resilience tests | Resumable upload and duplicate files re-linked by hash |
| Append-only audit logs | Integration + data integrity checks | No update/delete path, before/after values captured |

## Verification Stages
1. Static stage: lint, schema validation, OpenAPI contract checks.
2. Unit stage: domain logic and validation tests.
3. Integration stage: API + DB + filesystem behavior.
4. E2E stage: role-specific user journeys in isolated environment.
5. Release gate: coverage threshold + smoke tests in Docker intranet profile.

## Non-Functional Verification
- Offline startup verification with blocked external network egress.
- Recovery tests for DB restart and interrupted chunk uploads.
- Security checks for masking and encrypted-at-rest persistence paths.
