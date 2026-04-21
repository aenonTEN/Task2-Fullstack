# Backend Static Architecture Audit Report

## 1. Verdict
**Conclusion: Fail**
While the backend successfully implements many of the Prompt's structural requirements (Go/Gin, local offline design, basic authentication, file system attachments, and case ledger numbering), it fails on critical security boundaries (missing Role-Based Access Control on business routes) and test maintainability (tests are hard-coupled to a live database and immediately panic).

## 2. Scope and Static Verification Boundary
- **Reviewed:** Statically analyzed the backend codebase located at `repo/backend`, including Gin handlers, routing, authentication, and testing infrastructure.
- **Not Reviewed:** Database runtime state, frontend interactions, or actual Docker container execution.
- **Not Executed:** The Go server and tests were not run.
- **Manual Verification Required:** Real file system performance for the chunked attachment uploads and actual query performance of the scoring engine.

## 3. Repository / Requirement Mapping Summary
- **Core Business Goal:** Intranet Go/Gin API supporting recruitment, compliance, and case operations with RBAC, offline attachments, and audit trails.
- **Core Flows & Constraints:**
  - **Auth & Session:** 8hr tokens, bcrypt, >=8 chars. Mapped to `internal/httpserver/auth.go`.
  - **Recruitment:** Deduplication, explainable scoring. Mapped to `internal/httpserver/recruitment.go`.
  - **Case Handling:** 5-min dedupe, unique IDs. Mapped to `internal/httpserver/caseledger.go`.
  - **Compliance:** 30-day auto-expiry, 168h purchase rules, attachments. Mapped to `internal/httpserver/compliance.go`.
- **Finding:** Most flows are mapped out, but compliance restrictions lack the required "prescription attachment" check, and all feature endpoints lack explicit role validation.

## 4. Section-by-section Review

### 1. Hard Gates
- **1.1 Documentation and static verifiability: Partial Pass.** `docker-compose.yml` and `migrate.sh` exist, but there is no specific `README.md` for the backend explaining the configuration (e.g., `DB_DSN` requirements) or how to run tests.
- **1.2 Prompt alignment: Pass.** The delivered project uses Go, Gin, and MySQL as requested, and implements the required domains.

### 2. Delivery Completeness
- **2.1 Core requirement coverage: Partial Pass.** The 168h rolling purchase restriction is implemented (`complianceCheckRestriction`), but the requirement "requiring prescription attachments" for controlled medications is completely ignored in the restriction logic.
- **2.2 End-to-end project shape: Partial Pass.** The project provides real SQL integration rather than mocks, but lacks basic documentation.

### 3. Engineering and Architecture Quality
- **3.1 Structure and modularity: Fail.** The architecture is extremely tightly coupled. SQL queries (`db.QueryRowContext`, `db.ExecContext`) are hardcoded directly into the Gin HTTP handlers. There is no separation between the Transport layer, Service layer, and Repository layer.
- **3.2 Maintainability and extensibility: Fail.** The monolithic 150-line handlers (e.g., `recruitmentCreateCandidate`) make it very difficult to extend business logic or test components in isolation.

### 4. Engineering Details and Professionalism
- **4.1 Frontend/Backend engineering quality: Partial Pass.** Data encryption (AES) and masking (`maskPhone`) are professionally handled. However, multi-step operations (like merging candidates or completing chunked uploads) do not use SQL Transactions (`sql.Tx`), leaving the system vulnerable to partial failures and data corruption.
- **4.2 Product credibility: Partial Pass.** Handles advanced concepts like chunked file uploads and SHA256 deduplication well (`attachments.go`), but the hard-coded scoring logic loading 200 candidate rows into memory (`recruitmentSearch`) is not production-ready.

### 5. Prompt Understanding and Requirement Fit
- **5.1 Business understanding: Partial Pass.** Correctly implements the nuanced unique numbering rule (`generateCaseNumber`) and duplicate blocking (`checkCaseDuplicateWindow`).

## 5. Issues / Suggestions (Severity-Rated)

### Issue 1: Missing Role-Based Access Control (RBAC) on Core Routes
- **Severity:** Blocker
- **Conclusion:** Fail
- **Evidence:** `internal/httpserver/router.go:88-151`
- **Impact:** While `RequireAuth` ensures the user is logged in, none of the `/recruitment`, `/compliance`, or `/cases` routes use the `requireRole` middleware. Any authenticated user within the institution can perform any action (e.g., a recruitment specialist could approve compliance qualifications).
- **Minimum Actionable Fix:** Apply `requireRole("role_business_specialist")` etc., to the respective routing groups.

### Issue 2: Tightly Coupled Tests Panic on Execution
- **Severity:** High
- **Conclusion:** Fail
- **Evidence:** `internal/httpserver/router_test.go:11`, `internal/httpserver/router.go:19-23`
- **Impact:** The `router_test.go` integration tests call `NewRouter()`, which immediately panics if the `DB_DSN` environment variable is not set and a live MySQL DB is not running. Tests cannot be run in isolation.
- **Minimum Actionable Fix:** Extract the database connection logic out of `NewRouter()` and inject the `*sql.DB` dependency so it can be mocked or pointed to a test DB.

### Issue 3: Missing SQL Transactions
- **Severity:** High
- **Conclusion:** Fail
- **Evidence:** `internal/httpserver/attachments.go:268-305`
- **Impact:** When completing an attachment upload, the system renames a file, queries chunks, and inserts an attachment record sequentially without a database transaction. If the system crashes mid-way, it results in orphaned files or inconsistent database states.
- **Minimum Actionable Fix:** Use `db.BeginTx` for multi-step mutations across all handlers.

### Issue 4: Incomplete Compliance Purchase Restrictions
- **Severity:** Medium
- **Conclusion:** Fail
- **Evidence:** `internal/httpserver/compliance.go:221-280`
- **Impact:** The prompt explicitly requires "requiring prescription attachments" for restricted purchases. The applied restriction logic only checks the time window and completely ignores attachment validation.
- **Minimum Actionable Fix:** Add a `AttachmentID` field to the restriction apply request and verify it exists in the database.

## 6. Security Review Summary

- **Authentication Entry Points: Pass.** Passwords are required to be >= 8 chars and are hashed with bcrypt. 8-hour sessions are implemented using UUID tokens. (`auth.go:76-100`).
- **Route-level Authorization: Fail.** The `requireRole` middleware exists but is completely omitted from all business endpoints (Recruitment, Compliance, Cases).
- **Object-level / Tenant Isolation: Partial Pass.** `scope.InstitutionID` is checked across almost all queries (`WHERE institution_id = ?`), enforcing basic tenant isolation. However, Department and Team scopes are ignored.
- **Function-level Authorization: Missing.** Handlers do not verify if the actor has specific permissions to update/delete specific records.
- **Admin / Internal Protection: Pass.** `/admin/ping` and `/audit/records` correctly enforce `requireRole("role_admin")`.
- **Data Protection: Pass.** Phone and ID numbers are AES-GCM encrypted at rest (`recruitment.go:118`) and dynamically masked on read (`maskPhone`).

## 7. Tests and Logging Review

- **Unit Tests: Fail.** `unit_test.go` only covers trivial pure functions (e.g., `maskPhone`). HTTP handlers and business logic have 0% isolated coverage.
- **API / Integration Tests: Fail.** `router_test.go` is broken by design (panics without a live DB) and only tests login and readiness.
- **Logging Categories / Observability: Pass.** An append-only audit log is rigorously implemented via the `store.AppendAudit` and `auditAppendMiddleware`, capturing traces, actors, entities, and diffs (Before/After JSON).
- **Sensitive-Data Leakage Risk in Logs: Partial Pass.** The `candidateDetailItem` masks the `IDNumber` as `***ENCRYPTED***` before saving it to the audit log, preventing leaks.

## 8. Test Coverage Assessment (Static Audit)

### 8.1 Test Overview
- Unit tests (`unit_test.go`) and API tests (`router_test.go`) exist using the standard `testing` package and `httptest`.
- Evidence: `internal/httpserver/router_test.go`, `internal/httpserver/unit_test.go`.

### 8.2 Coverage Mapping Table

| Requirement / Risk Point | Mapped Test Case(s) | Key Assertion / Fixture | Coverage Assessment | Gap | Minimum Test Addition |
| --- | --- | --- | --- | --- | --- |
| Login / Auth Flow | `TestAuthLoginAndLogout` | `loginRes.Code != 200` | Insufficient | Fails without live DB | Mock DB layer, test invalid credentials |
| Token Expiration / Revocation | None | None | Missing | Session expiry untested | Test token `ExpiresAt` and `RevokedAt` logic |
| Case Duplicate Blocking (5 min) | None | None | Missing | Core business rule untested | Integration test checking 409 Conflict |
| Recruitment Deduplication | None | None | Missing | Core business rule untested | Test merging logic and skill accumulation |
| Attachment Chunked Upload | None | None | Missing | Complex FS logic untested | Mock FS, test SHA256 and chunk merging |

### 8.3 Security Coverage Audit
- **Authentication:** Insufficient. Only happy path login tested.
- **Route Authorization:** Missing. Tests do not verify that `requireRole` blocks unauthorized users.
- **Tenant / Data Isolation:** Missing. No tests simulate User A attempting to read User B's institution data.
- **Admin / Internal Protection:** Missing.

### 8.4 Final Coverage Judgment
- **Conclusion: Fail**
- **Boundary Explanation:** The tests theoretically exist but are completely inoperable in a standard CI environment due to hardcoded live-database dependencies. Furthermore, 0% of the core business logic, complex data merging, file system operations, and RBAC rules are covered. Severe defects can effortlessly bypass the current test suite.
