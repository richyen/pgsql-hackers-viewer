# Architecture Overview

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Web Browser (Port 3000)                   │
│         React Frontend (TypeScript + React)                  │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │ Thread List  │  │ Message View │  │ Stats Panel  │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└─────────────────────────────────────────────────────────────┘
                          │
                  HTTP Requests (REST)
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Go Backend API (Port 8080)                       │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ API Routes Layer                                      │   │
│  │ - GET /api/threads                                    │   │
│  │ - GET /api/threads/{id}                              │   │
│  │ - GET /api/threads/{id}/messages                     │   │
│  │ - GET /api/stats                                      │   │
│  │ - POST /api/sync                                      │   │
│  └──────────────────────────────────────────────────────┘   │
│                          │                                    │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Business Logic Layer                                  │   │
│  │ - Thread Analyzer (classification)                    │   │
│  │ - Mail Parser (IMAP support)                          │   │
│  │ - Data Processing                                     │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                          │
                 SQL Queries (PostgreSQL)
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│         PostgreSQL Database (Port 5432)                       │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Database Schema                                       │   │
│  │                                                        │   │
│  │  Threads                                              │   │
│  │  ├─ id (PRIMARY KEY)                                  │   │
│  │  ├─ subject                                           │   │
│  │  ├─ first_author / first_author_email                │   │
│  │  ├─ created_at / updated_at / last_message_at        │   │
│  │  ├─ message_count / unique_authors                   │   │
│  │  └─ status (in-progress|discussion|stalled|abandoned) │   │
│  │                                                        │   │
│  │  Messages                                             │   │
│  │  ├─ id (PRIMARY KEY)                                  │   │
│  │  ├─ thread_id (FOREIGN KEY)                           │   │
│  │  ├─ message_id (UNIQUE, from IMAP)                    │   │
│  │  ├─ subject / author / author_email                   │   │
│  │  └─ created_at                                        │   │
│  │                                                        │   │
│  │  Thread Activities                                    │   │
│  │  ├─ thread_id (UNIQUE)                                │   │
│  │  ├─ message_count / unique_authors                    │   │
│  │  ├─ has_patch / has_review                            │   │
│  │  ├─ days_since_last_message                           │   │
│  │  └─ created_at / updated_at                           │   │
│  │                                                        │   │
│  │  Indexes: thread_id, status, last_message_at, created_at │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### 1. Mail Synchronization

```
IMAP Server (pgsql-hackers)
    │
    ▼
Mail Parser (Go)
    │ - Connects via IMAP TLS
    │ - Fetches messages since last sync
    │ - Extracts: subject, author, date, message-id
    │
    ▼
Database
    │ - Insert/update messages
    │ - Group by subject (thread)
    │ - Update thread metadata
    │
    ▼
Analyzer
    │ - Check for patch keywords
    │ - Check for review keywords
    │ - Calculate activity metrics
    │ - Classify thread status
    │
    ▼
Frontend (via API)
    │ - Display threads by status
    │ - Show statistics
```

### 2. Thread Classification Algorithm

```
For each thread:
    1. Count messages and unique authors
    2. Find last message date
    3. Scan message bodies for keywords
    
    If has_patch AND (has_review OR message_count > 3):
        status = "in-progress"
    
    Else if days_since_last_message > 30 AND message_count < 5:
        status = "abandoned"
    
    Else if days_since_last_message > 7:
        status = "stalled"
    
    Else:
        status = "discussion"
```

## Deployment Architecture

### Docker Compose Setup

```
┌──────────────────────────────────────────────┐
│           Docker Network (default)            │
│                                               │
│  ┌─────────────────────────────────────────┐ │
│  │ postgres:15 (pgsql_analyzer_db)         │ │
│  │ Volume: postgres_data                   │ │
│  │ Port: 5432 → localhost:5432             │ │
│  └─────────────────────────────────────────┘ │
│                    ▲                          │
│                    │ SQL                      │
│                    │                          │
│  ┌─────────────────────────────────────────┐ │
│  │ Go Backend (pgsql_analyzer_backend)     │ │
│  │ Build: Dockerfile.backend               │ │
│  │ Ports: 8080 → localhost:8080            │ │
│  │ Depends on: postgres (healthcheck)      │ │
│  │ Env: DB_HOST=postgres                   │ │
│  └─────────────────────────────────────────┘ │
│                    ▲                          │
│                    │ REST API                 │
│                    │                          │
│  ┌─────────────────────────────────────────┐ │
│  │ React Frontend (pgsql_analyzer_frontend)│ │
│  │ Build: Dockerfile.frontend              │ │
│  │ Ports: 3000 → localhost:3000            │ │
│  │ Depends on: backend                     │ │
│  │ Env: REACT_APP_API_URL=localhost:8080   │ │
│  └─────────────────────────────────────────┘ │
│                                               │
└──────────────────────────────────────────────┘
```

### Build Process

```
docker-compose up
    │
    ├─ Starts postgres:15 container
    │   │
    │   └─ Waits for healthcheck (pg_isready)
    │
    ├─ Builds and starts backend
    │   │ - Dockerfile.backend
    │   │   - golang:1.21-alpine (build)
    │   │   - alpine:3.18 (runtime)
    │   │ - Waits for postgres healthcheck
    │   │ - Runs: go run main.go
    │   │   - Auto-runs migrations
    │   │   - Starts API on :8080
    │
    └─ Builds and starts frontend
        │ - Dockerfile.frontend
        │   - node:18-alpine (build)
        │   - node:18-alpine (serve)
        │ - Runs: serve -s build -l 3000
        │   - Serves React build on :3000
```

## Technology Stack

| Layer | Technology | Version | Purpose |
|-------|-----------|---------|---------|
| Frontend | React | 18.2 | UI Components |
| Frontend Lang | TypeScript | 5.1 | Type Safety |
| Frontend HTTP | Axios | 1.4 | API Requests |
| Backend | Go | 1.21 | API Server |
| Backend HTTP | Gorilla Mux | 1.8 | Routing |
| Backend Mail | go-imap | 1.2.1 | IMAP Client |
| Backend Mail | go-message | 0.17 | Email Parsing |
| Database | PostgreSQL | 15 | Data Storage |
| Database Driver | pq | 1.10 | Go PostgreSQL |
| Container | Docker | - | Deployment |
| Orchestration | Docker Compose | 3.8 | Service Management |

## Performance Characteristics

- **API Response Time**: < 100ms for typical queries
- **Thread Analysis**: ~1-2ms per thread
- **Database Indexes**: Optimized for thread_id, status, last_message_at
- **Frontend Stats Refresh**: Every 30 seconds
- **Thread Limit**: 50 threads per page (configurable)

## Security Considerations

- CORS headers configured for local development
- IMAP over TLS for mail synchronization
- Password stored in environment variables (not in code)
- SQL queries use parameterized statements (no SQL injection)
- No authentication required (add if deploying publicly)

## Scalability Notes

For large deployments with 10,000+ threads:

1. **Database**: Add connection pooling (pq supports this)
2. **Pagination**: Implement cursor-based pagination
3. **Caching**: Add Redis for stats caching
4. **Mail Sync**: Use background job queue (e.g., Bull.js)
5. **Frontend**: Implement virtual scrolling for large lists
6. **API**: Add rate limiting and request validation
