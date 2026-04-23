# Feature Tests

Feature tests for business logic patterns and rules.

## Test Files

- `integration_logic.go` - Extracted business logic functions
- `integration_logic_test.go` - Integration logic tests

## Functions Tested

- **Scoring**: `CalculateMatchScore()` - Skills, experience, education weighting
- **Deduplication**: `CheckCaseDuplicate()` - 5-minute case dedupe window
- **Restrictions**: `IsWithinRestrictionWindow()` - 168-hour restriction
- **Files**: `ValidateFileSize()`, `ValidateChunkSize()`, `IsAllowedMimeType()`
- **Chunk Assembly**: `IsWithinChunkAssemblyWindow()`

## Running Tests

```bash
go test ./internal/httpserver/... -v
```

## Integration

These functions are integrated into production handlers:
- `recruitment.go` uses `CalculateMatchScore()`
- `caseledger.go` uses `CheckCaseDuplicate()`

See testing-improvement-guide.md for full context.