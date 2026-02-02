# PostgreSQL Mailing List Analyzer - Project Instructions

A full-stack application for analyzing pgsql-hackers mailing list threads to identify which threads are actively being worked on versus still under discussion.

## Project Setup Checklist

- [x] Scaffold the project structure
- [x] Set up Go backend with mail parsing and API
- [x] Set up TypeScript frontend with React UI
- [x] Configure Docker and Docker Compose
- [x] Implement database schema and models
- [x] Create mailing list parser functionality
- [x] Build REST API endpoints
- [x] Create web dashboard UI
- [x] Test and verify full deployment

## Quick Start

```bash
cd pgsql-analyzer
docker-compose up
```

Access at:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080/api

## Project Structure

```
backend/          - Go API server (port 8080)
frontend/         - React TypeScript UI (port 3000)
docker-compose.yml - Service orchestration
README.md         - Project overview
GETTING_STARTED.md - Setup guide
ARCHITECTURE.md   - System design
CONTRIBUTING.md   - Development guide
```

## Key Features

- **Thread Analysis**: Categorizes threads by status (in-progress, discussion, stalled, abandoned)
- **Web Dashboard**: Real-time thread view with filtering and statistics
- **Mail Parser**: IMAP support for syncing mailing list messages
- **REST API**: Full-featured API for thread data and analytics
- **Docker Ready**: Single command deployment with Docker Compose

## Technical Stack

- **Backend**: Go 1.21, Gorilla Mux, PostgreSQL driver
- **Frontend**: TypeScript, React 18, Axios
- **Database**: PostgreSQL 15
- **Deployment**: Docker, Docker Compose
- **Mail**: IMAP with TLS

## Key Ports

- 3000: Frontend (React)
- 8080: Backend API (Go)
- 5432: PostgreSQL Database

## Database Tables

- `threads`: Mailing list threads with metadata
- `messages`: Individual email messages
- `thread_activities`: Activity metrics and classification

## Thread Status Classification

- **In Progress**: Has patches and review activity
- **Discussion**: Active but no development work
- **Stalled**: No activity for 7-30 days
- **Abandoned**: No activity for 30+ days

## Development Notes

- Frontend auto-reloads in Docker dev mode
- Backend auto-rebuilds when code changes
- Run `make help` for available development tasks
- See CONTRIBUTING.md for development workflow
- See ARCHITECTURE.md for system design details
