# Integration Tests

Integration tests for RBAC, tenant isolation, and handlers.

## Test Files

- `integration_test.go` - RBAC and tenant isolation tests

## Test Coverage

- **RBAC**: `role_user` denied (403), `role_admin` allowed
- **Tenant Isolation**: Cross-institution access denied
- **Case Deduplication**: 5-minute window logic
- **Chunk Assembly**: Upload timeout logic

## Running Tests

```bash
go test ./internal/httpserver/... -v -cover
```

## Key Points

- Tests use mocked Gin context for isolated testing
- No database dependency for unit/integration tests
- Production handlers integrate extracted functions for audit value