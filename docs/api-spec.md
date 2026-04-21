# API Specification

## 1. Scope
This API contract covers the offline intranet platform domains:
- auth/session
- recruitment
- compliance
- case ledger
- attachments
- audit

All endpoints are REST JSON unless explicitly marked as binary upload.

## 2. Cross-Cutting Rules

### Authentication
- Local username/password authentication.
- Password min length: 8.
- Session token TTL: 8 hours.
- Logout invalidates the current token immediately.

### Authorization
- Every protected endpoint enforces RBAC plus data-scope checks.
- Data scope dimensions: institution/department/team.
- Client/supplier records are institution-owned.

### Sensitive Data
- `idNumber` and `phone` are encrypted at rest.
- List/search responses return masked values for sensitive fields.

### Error Model
All non-2xx responses use:
```json
{
  "code": "string",
  "message": "string",
  "details": {},
  "traceId": "string"
}
```

### Idempotency And Duplicate Guards
- Case creation blocks duplicate submissions within 5 minutes.
- Upload completion is idempotent by upload token + SHA256.
- Import operations should support retry-safe semantics.

## 3. Domain Clarifications Applied
- Qualification expiration is check-on-arrival (evaluated on login and regulated operations).
- Purchase restriction is strict rolling 168 hours from last approved purchase.
- Regulated record deletion is soft delete only (no hard delete).
- Match score explanation baseline: skills 50, experience 30, education 20.

## 4. Endpoint Groups

## 4.1 Auth

### `POST /api/v1/auth/login`
- Auth: none
- Request:
```json
{
  "username": "string",
  "password": "string"
}
```
- Response `200`:
```json
{
  "accessToken": "string",
  "expiresAt": "2026-04-21T18:00:00Z"
}
```

### `POST /api/v1/auth/logout`
- Auth: bearer token
- Response: `204 No Content`

## 4.2 Recruitment

### `POST /api/v1/recruitment/candidates/import`
- Auth: bearer token + scope check
- Purpose: bulk import resume/candidate data.
- Request:
```json
{
  "fileRef": "string",
  "importMode": "append|upsert"
}
```
- Response `202`:
```json
{
  "jobId": "string",
  "status": "accepted"
}
```

### `POST /api/v1/recruitment/candidates/merge`
- Auth: bearer token + scope check
- Purpose: merge duplicates triggered by phone or ID collision.
- Request:
```json
{
  "primaryCandidateId": "string",
  "duplicateCandidateId": "string",
  "reason": "string"
}
```
- Response `200`:
```json
{
  "mergedCandidateId": "string"
}
```

### `GET /api/v1/recruitment/search`
- Auth: bearer token + scope check
- Query: `keyword`, `skill`, `education`, `minExperienceYears`, `sortBy`, `order`
- Response `200`:
```json
[
  {
    "candidateId": "string",
    "positionId": "string",
    "score": 86,
    "reasons": [
      "3 required skills matched",
      "Experience threshold met",
      "Education below preferred level"
    ]
  }
]
```

## 4.3 Compliance

### `POST /api/v1/compliance/qualifications`
- Auth: bearer token + scope check
- Purpose: create/update qualification profile.
- Request:
```json
{
  "partyType": "client|supplier",
  "partyId": "string",
  "institutionId": "string",
  "qualificationType": "string",
  "issuedAt": "2026-04-01T00:00:00Z",
  "expiresAt": "2027-04-01T00:00:00Z"
}
```
- Response `200`:
```json
{
  "id": "string",
  "status": "active|expiring_soon|expired|inactive"
}
```

### `POST /api/v1/compliance/restrictions/evaluate`
- Auth: bearer token + scope check
- Purpose: evaluate purchase allowance with 168-hour rolling rule.
- Request:
```json
{
  "clientId": "string",
  "medicationId": "string",
  "prescriptionAttachmentId": "string"
}
```
- Response `200`:
```json
{
  "allowed": false,
  "rationale": "Last approved purchase was within 168 hours",
  "nextEligibleAt": "2026-04-28T14:01:00Z"
}
```

## 4.4 Case Ledger

### `POST /api/v1/cases`
- Auth: bearer token + scope check
- Purpose: create case with duplicate-window enforcement.
- Request:
```json
{
  "institutionCode": "string",
  "keyFields": {}
}
```
- Response `201`:
```json
{
  "id": "string",
  "caseNo": "YYYYMMDD-INSTITUTION-000001",
  "status": "new"
}
```

### `POST /api/v1/cases/{id}/assign`
- Auth: bearer token + scope check
- Request:
```json
{
  "assigneeId": "string"
}
```
- Response `200`: updated case object

## 4.5 Attachments

### `POST /api/v1/attachments/uploads/start`
- Auth: bearer token + scope check
- Purpose: start resumable upload.
- Request:
```json
{
  "fileName": "string",
  "mimeType": "string",
  "totalSize": 1234,
  "expectedSha256": "string"
}
```
- Response `200`:
```json
{
  "uploadId": "string",
  "chunkSize": 5242880
}
```

### `PUT /api/v1/attachments/uploads/{uploadId}/chunks/{chunkIndex}`
- Auth: bearer token + scope check
- Content-Type: `application/octet-stream`
- Response: `204 No Content`

### `POST /api/v1/attachments/uploads/{uploadId}/complete`
- Auth: bearer token + scope check
- Purpose: finalize upload and dedupe by SHA256.
- Response `200`:
```json
{
  "id": "string",
  "sha256": "string",
  "mimeType": "string",
  "sizeBytes": 1234
}
```

## 4.6 Audit

### `GET /api/v1/audit/records`
- Auth: bearer token + elevated permission + scope check
- Query: `actorId`, `entityType`, `entityId`, `from`, `to`, `action`
- Response `200`:
```json
[
  {
    "id": "string",
    "traceId": "string",
    "actorId": "string",
    "sourceIp": "string",
    "entityType": "Candidate",
    "entityId": "string",
    "action": "UPDATE",
    "beforeSnapshot": {},
    "afterSnapshot": {},
    "createdAt": "2026-04-21T12:00:00Z"
  }
]
```

## 5. Logging, Validation, And Operational Notes
- All write operations must emit structured logs with `traceId`, actor, scope, action, entity, status, and latency.
- Validation layers:
  - transport validation (shape/required/range/format)
  - domain validation (policy rules, windows, numbering)
  - persistence validation (unique/check constraints)
- In offline deployment, APIs must not rely on external network services.

## 6. Testability Expectations
- Endpoint groups require contract tests for success and failure paths.
- Auth + scope permutations require integration tests.
- High-risk flows (case creation, restriction evaluation, upload completion) require idempotency/duplicate-window tests.
- Coverage target remains practical >=90% where applicable, with strict focus on high-risk modules.
