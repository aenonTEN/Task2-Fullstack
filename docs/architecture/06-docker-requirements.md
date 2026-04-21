# Docker Requirements

## Services
- `frontend`: serves Angular static build.
- `api`: Go/Gin API service.
- `db`: MySQL database.
- `migration`: one-shot migration job.

## Required Volumes
- `attachments_data`: persisted uploaded attachments.
- `chunk_data`: resumable temporary upload chunks.
- `export_data`: exported files and reports.
- `db_data`: SQL data files.

## Health Checks
- Frontend: static root response.
- API: readiness endpoint `/health/ready`.
- DB: `mysqladmin ping`.
- Migration: exit code must be zero before release deployment can continue.

## Intranet Constraints
- No external SaaS dependencies at runtime.
- All artifacts must be buildable from local registry mirrors where required.
- Deployment script must pre-create and validate writable storage paths.

## Operational Checklist
1. Copy `.env.example` to `.env` and set secure secrets.
2. Run `docker compose up --build -d`.
3. Verify service health and API readiness.
4. Execute smoke tests for auth, case creation, and attachment upload.
5. Validate backup and restore for DB and filesystem volumes.
