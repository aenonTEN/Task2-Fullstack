# Testing Depth And Practical Coverage Policy

## Coverage Targets (Where Applicable)
- Backend domain and policy modules: >= 90% line coverage.
- Frontend business logic modules (services/guards/interceptors/validators): >= 90% line coverage.
- Excluded from strict thresholds:
  - generated code
  - thin DTOs and constants-only modules
  - framework bootstrap wiring

## Depth By Test Type
- Unit tests:
  - deterministic business rule tests
  - edge-condition tests for dedupe windows, numbering, and restriction policies
- Integration tests:
  - API + DB + filesystem contracts
  - auth + scope permutations
  - upload/resume/retry behavior
- End-to-end tests:
  - role-based navigation and visibility
  - critical user workflows in recruitment, compliance, and case operations

## Practical Coverage Controls
- Per-module coverage gates in CI, not just global aggregate.
- High-risk modules (authorization, case numbering, restriction evaluation, audit append) require strict threshold enforcement.
- Mutation testing is recommended for core policy engines where tooling/runtime allows.
- Coverage reports are reviewed alongside defect escape trends to ensure meaningful, not superficial, coverage.
