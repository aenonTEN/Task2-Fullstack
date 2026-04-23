# Unit Tests

Unit tests for utility functions and pure business logic.

## Test Files

- `unit_test.go` - Data structures and response types
- `business_test.go` - Scoring algorithms, restrictions, RBAC
- `router_test.go` - Router and endpoint tests
- `recruitment_logic_test.go` - Extracted scoring logic tests

## Extracted Functions

The following pure functions are tested:

- `CalculateMatchScore()` - Candidate scoring algorithm
- `IsWithinRestrictionWindow()` - 168-hour restriction window
- `CheckWithinDeduplicationWindow()` - 5-minute case dedupe window
- `ValidateFileSize()` / `ValidateChunkSize()` - File size validation
- `IsAllowedMimeType()` - File type whitelist

## Running Tests

```bash
go test ./internal/httpserver/... -v
```

## Coverage

These tests exercise the extracted business logic functions used by production handlers.