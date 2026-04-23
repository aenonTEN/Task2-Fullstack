# TASK-2-Fullstack Audit Revision 01: Initial Static Review

**Verdict:** ❌ **FAIL**

## Key Findings

### 1. Blocker: Inverted Expiry Logic
`compliance.go:49` (checkQualificationExpiry) incorrectly returns `true` (active) if the expiry date has passed. This corrupts the entire compliance module's integrity.

### 2. High: Hardcoded Cryptography
`recruitment.go:18-19` uses a hardcoded 32-byte AES key and a fixed 12-byte nonce. This makes PII encryption deterministic and easily reversible by any attacker with access to the source code.

### 3. High: RBAC Documentation Mismatch
The README (added in commit `52b468b`) claims that all write operations require `role_admin`, but the `router.go` implementation does not apply any role-based middleware to these routes.

### 4. High: Protocol Mismatch
The Angular frontend attempts to send multipart binary chunks, while the Go backend expects JSON-wrapped hex strings. Chunked uploads will fail at runtime.

### 5. Medium: Non-Functional Test Runner
`run_tests.sh` contains invalid bash syntax (dual `else` clauses in a single `if` block), making automated testing impossible.

### 6. Low: UX/Engineering Detail
Frontend components currently render raw JSON dumps instead of using the provided design system (`styles.css`).
