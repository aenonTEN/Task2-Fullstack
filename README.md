# Integrated Platform Architecture

An enterprise recruitment and case management system built with Angular frontend, Go/Gin backend, and MySQL database. Designed for offline intranet deployment.

## Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Angular UI     │────▶│  Go/Gin API     │────▶│  MySQL 8.4      │
│  (Port 4200)    │     │  (Port 8080)    │     │  (Port 3306)    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌─────────────────┐
                        │ Local File      │
                        │ Storage         │
                        │ (Attachments)   │
                        └─────────────────┘
```

## Tech Stack

- **Frontend**: Angular 17+, TypeScript, Standalone Components
- **Backend**: Go 1.24+, Gin Web Framework
- **Database**: MySQL 8.4
- **Authentication**: JWT tokens with bcrypt password hashing
- **Deployment**: Docker Compose for intranet deployment

## Features

### 1. Authentication & Authorization
- JWT-based session management with 8-hour TTL
- Role-based access control (RBAC)
- Data scope isolation by institution/department/team
- Session invalidation on logout
- Default admin user: `admin` / `password123`

### 2. Recruitment Module
- **Candidates**: Create, list, search candidates
- **Bulk Import**: Import candidates from JSON files
- **Search & Scoring**: Skills-based candidate matching (skills: 50pts, experience: 30pts, education: 20pts)
- Phone and ID number deduplication

### 3. Compliance Module
- **Qualifications**: Track candidate qualifications with expiry dates
- **Auto-Expiry**: Automatic qualification deactivation on expiry
- **Restrictions**: 168-hour (7-day) purchase restriction window
- **Check-on-Arrival**: Expiry validation on login and regulated operations

### 4. Case Ledger Module
- **Case Management**: Create, update, assign cases
- **Auto-Numbering**: Institution-prefixed case numbers with date-based serial
- **Deduplication**: 5-minute window to prevent duplicate case creation
- **History Tracking**: Full audit trail of case actions
- **Attachments**: File uploads with chunked transfer and SHA256 deduplication

### 5. Positions & Profiles
- **Job Positions**: Create and manage job openings
- **Qualification Profiles**: Define required skills, experience, and education

### 6. Tags System
- **Configurable Tags**: Create custom tags with colors
- **Assignment**: Assign tags to candidates or cases
- **Filtering**: Filter entities by assigned tags

### 7. Audit Logging
- **Immutable Records**: Append-only audit trail
- **Action Tracking**: All CRUD operations logged
- **Before/After Deltas**: Full change history

## Project Structure

```
repo/
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go          # Application entry point
│   ├── internal/
│   │   ├── httpserver/
│   │   │   ├── auth.go          # Authentication handlers
│   │   │   ├── authorization.go  # RBAC middleware
│   │   │   ├── attachments.go    # File upload handling
│   │   │   ├── audit.go         # Audit logging
│   │   │   ├── caseledger.go   # Case management
│   │   │   ├── compliance.go    # Compliance rules
│   │   │   ├── idempotency.go  # Idempotency keys
│   │   │   ├── positions.go     # Job positions
│   │   │   ├── recruitment.go   # Candidate management
│   │   │   ├── router.go       # Route definitions
│   │   │   ├── tags.go         # Tag management
│   │   │   └── unit_test.go    # Unit tests
│   │   └── persistence/
│   │       ├── mysql.go        # MySQL connection
│   │       ├── schema.go        # Database schema
│   │       └── store.go        # Data store
│   ├── Dockerfile
│   ├── go.mod
│   └── migrate.sh              # Database migration script
│
├── frontend/
│   ├── src/
│   │   ├── app/
│   │   │   ├── api.service.ts        # API communication
│   │   │   ├── app.component.ts      # Root component
│   │   │   ├── recruitment/          # Recruitment module
│   │   │   ├── compliance/           # Compliance module
│   │   │   ├── caseledger/           # Case ledger module
│   │   │   ├── audit/                # Audit log module
│   │   │   └── tags/                 # Tags management
│   │   ├── main.ts                   # Bootstrap
│   │   └── styles.css                # Global styles
│   ├── Dockerfile
│   ├── nginx.conf                    # Nginx configuration
│   └── package.json
│
├── docker-compose.yml                # Container orchestration
├── .env.example                      # Environment variables
└── README.md                         # This file
```

## API Endpoints

### Authentication
| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/v1/auth/login` | User login | No |
| POST | `/api/v1/auth/logout` | User logout | Yes |
| GET | `/api/v1/me` | Current user info | Yes |

### Recruitment
| Method | Endpoint | Description | Role Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/recruitment/candidates` | List candidates | Yes |
| POST | `/api/v1/recruitment/candidates` | Create candidate | role_admin |
| POST | `/api/v1/recruitment/bulk` | Bulk import | role_admin |
| GET | `/api/v1/recruitment/search` | Search candidates | Yes |

### Compliance
| Method | Endpoint | Description | Role Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/compliance/qualifications` | List qualifications | Yes |
| POST | `/api/v1/compliance/qualifications` | Create qualification | role_admin |
| POST | `/api/v1/compliance/qualifications/expire` | Auto-expire | role_admin |
| POST | `/api/v1/compliance/qualifications/reactivate` | Reactivate | role_admin |
| POST | `/api/v1/compliance/restrictions/check` | Check restriction | Yes |
| POST | `/api/v1/compliance/restrictions` | Apply restriction | role_admin |

### Case Ledger
| Method | Endpoint | Description | Role Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/cases` | List cases | Yes |
| POST | `/api/v1/cases` | Create case | role_admin |
| PATCH | `/api/v1/cases/:id/status` | Update status | role_admin |
| POST | `/api/v1/cases/:id/assign` | Assign case | role_admin |
| GET | `/api/v1/cases/:id/history` | Case history | Yes |
| GET | `/api/v1/cases/:id/attachments` | Case attachments | Yes |

### Attachments
| Method | Endpoint | Description | Role Required |
|--------|----------|-------------|---------------|
| POST | `/api/v1/attachments/init` | Init upload | role_admin |
| POST | `/api/v1/attachments/:uploadId/chunk` | Upload chunk | role_admin |
| POST | `/api/v1/attachments/complete` | Complete upload | role_admin |
| GET | `/api/v1/attachments/:id/download` | Download file | Yes |

### Tags
| Method | Endpoint | Description | Role Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/tags` | List tags | Yes |
| POST | `/api/v1/tags` | Create tag | role_admin |
| DELETE | `/api/v1/tags/:id` | Delete tag | role_admin |
| POST | `/api/v1/tags/assign` | Assign tags | role_admin |
| GET | `/api/v1/tags/entity` | Get entity tags | Yes |

### Positions
| Method | Endpoint | Description | Role Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/positions` | List positions | Yes |
| POST | `/api/v1/positions` | Create position | role_admin |
| POST | `/api/v1/positions/:id/close` | Close position | role_admin |

### Profiles
| Method | Endpoint | Description | Role Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/profiles/qualifications` | List profiles | Yes |
| POST | `/api/v1/profiles/qualifications` | Create profile | role_admin |

### Audit
| Method | Endpoint | Description | Role Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/audit/records` | List audit records | role_admin |

## Getting Started

### Prerequisites
- Docker and Docker Compose
- Go 1.24+ (for local development)
- Node.js 22+ (for frontend development)

### Quick Start with Docker

1. Clone the repository and navigate to the project:
   ```bash
   cd repo
   ```

2. Start all services:
   ```bash
   docker compose up -d
   ```

3. Access the application:
   - Frontend: http://localhost:4200
   - API: http://localhost:8080
   - Database: localhost:3306

### Local Development

#### Backend
```bash
cd backend
go mod download
go run ./cmd/api
```

#### Frontend
```bash
cd frontend
npm install
npm start
```

### Environment Variables

Create a `.env` file based on `.env.example`:

```env
# API
API_PORT=8080

# Database
DB_DSN=root:root@tcp(localhost:3306)/eaglepoint?parseTime=true
DB_PORT=3306
DB_NAME=eaglepoint
DB_ROOT_PASSWORD=root

# Frontend
FRONTEND_PORT=4200
```

## Database Schema

### Core Tables

- **users**: User accounts with password hashes
- **sessions**: JWT tokens with expiration
- **candidates**: Job candidates with skills, education, experience
- **qualifications**: Candidate certifications with expiry dates
- **restrictions**: Purchase restrictions per candidate
- **cases**: Case records with numbering and status
- **case_history**: Case action audit trail
- **attachments**: File metadata with SHA256 hashes
- **attachment_chunks**: Chunked upload tracking
- **positions**: Job openings
- **qualification_profiles**: Required qualifications per position
- **tags**: Configurable tags
- **candidate_tags**: Tag assignments to candidates
- **case_tags**: Tag assignments to cases
- **audit_records**: Immutable audit log

## Security Features

1. **Password Hashing**: bcrypt with configurable cost factor
2. **JWT Tokens**: 8-hour TTL with secure storage
3. **Role-Based Access**: Middleware enforces role requirements
4. **Data Scope Isolation**: Institution-level data separation
5. **Input Validation**: Request body validation on all endpoints
6. **SQL Injection Prevention**: Parameterized queries
7. **File Type Whitelist**: Approved MIME types only
8. **SHA256 Deduplication**: Hash-based file deduplication

## Business Rules

### Qualification Expiry (AMB-02)
- Expiry checks occur on login and regulated operations
- Expired qualifications are auto-deactivated

### Restriction Window (AMB-03, AMB-07)
- 168-hour (7-day) rolling window for purchases
- Uses elapsed seconds, not calendar boundaries (DST-neutral)

### Case Serial Numbers (AMB-10)
- Format: `{INST}-{YYYYMMDD}-{SEQ}`
- Atomic allocation per institution per day

### Soft Delete (AMB-04)
- Regulated entities use soft delete only
- Records hidden from active views but queryable by audit

### Score Policy (AMB-05)
- Baseline scoring: Skills 50, Experience 30, Education 20
- Search results include human-readable score explanations

## Testing

### Backend Tests
```bash
cd backend
go test -v ./internal/httpserver/...
```

### Frontend Tests
```bash
cd frontend
npm test
```

### Docker Build Test
```bash
docker compose build
```

## Deployment

### Production Checklist

1. Set `GIN_MODE=release`
2. Configure strong database passwords
3. Set appropriate `BCRYPT_COST` (default: 10)
4. Configure `TOKEN_TTL` (default: 8h)
5. Set `EXPIRY_THRESHOLD_DAYS` (default: 30)
6. Configure file storage backup
7. Set up log rotation
8. Enable SSL/TLS termination at load balancer

### Docker Deployment

```bash
# Production build
docker compose -f docker-compose.yml build

# Run with production settings
docker compose up -d
```

### Health Checks

- **API Readiness**: `GET /api/v1/health/ready`
- **Database**: Checked automatically by readiness endpoint
- **Storage**: Configurable write path validation

## Troubleshooting

### Common Issues

1. **DB Connection Failed**
   - Check `DB_DSN` environment variable
   - Verify MySQL is running and accessible

2. **Attachment Upload Fails**
   - Check storage volume permissions
   - Verify `storageBasePath` is writable

3. **Authentication Errors**
   - Verify default credentials: `admin` / `password123`
   - Check token expiration

4. **Role Permission Denied (403)**
   - Ensure user has `role_admin` for write operations
   - Check role assignment in database

## License

Internal use only - Enterprise deployment
