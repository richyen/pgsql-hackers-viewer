# PostgreSQL Mailing List Thread Analyzer

A tool to parse and analyze the pgsql-hackers mailing list to identify which threads are actively being worked on versus those that are still under discussion or abandoned.

## Problem Statement

The PostgreSQL community uses the pgsql-hackers mailing list for development discussion, but there's no clear way to determine which threads have active development work, which are waiting for feedback, and which have been abandoned. This tool categorizes threads by activity status to help contributors avoid duplicating effort.

## Features

- **Mailing List Archive Parser**: Fetches and parses messages from pgsql-hackers archive
- **Mbox File Support**: Import messages from mbox files (standard mail archive format)
- **Thread Activity Analysis**: Categorizes threads by status:
  - **In Progress**: Active development, patches being worked on
  - **Under Discussion**: Active discussion but no current patch work
  - **Stalled**: No recent activity or feedback
  - **Abandoned**: No activity for extended period
- **Web Dashboard**: Real-time view of thread status and activity metrics
- **Filtering & Search**: Filter by activity status, date range, author

## Tech Stack

- **Backend**: Go (mail parsing, analysis engine)
- **Frontend**: TypeScript + React (web dashboard)
- **Database**: PostgreSQL (thread metadata, activity logs)
- **Deployment**: Docker & Docker Compose

## Project Structure

```
.
├── backend/
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   ├── config/
│   ├── parser/
│   ├── analyzer/
│   ├── api/
│   └── models/
├── frontend/
│   ├── src/
│   ├── package.json
│   ├── tsconfig.json
│   └── public/
├── docker-compose.yml
├── Dockerfile.backend
├── Dockerfile.frontend
└── README.md
```

## Getting Started

### Prerequisites

- Docker & Docker Compose
- Or locally: Go 1.21+, Node.js 18+, PostgreSQL 14+

### Quick Start with Docker

```bash
docker-compose up
```

This will:
- Start PostgreSQL database
- Build and run the Go backend (API on :8080)
- Build and run the TypeScript frontend (UI on :3000)

### Local Development

#### Backend Setup

```bash
cd backend
go mod download
go run main.go
```

#### Frontend Setup

```bash
cd frontend
npm install
npm start
```

## Configuration

Create a `.env` file in the root directory:

```env
# Database
DATABASE_URL=postgres://user:password@localhost:5432/pgsql_analyzer
DB_HOST=postgres
DB_PORT=5432
DB_NAME=pgsql_analyzer
DB_USER=postgres
DB_PASSWORD=postgres

# Mail Server
MAIL_IMAP_HOST=imap.example.com
MAIL_IMAP_PORT=993
MAIL_USERNAME=your@email.com
MAIL_PASSWORD=your_password

# API
API_PORT=8080
API_HOST=0.0.0.0

# Frontend
REACT_APP_API_URL=http://localhost:8080
```

## Usage

1. **Fetch Messages**: The backend will automatically fetch new messages from pgsql-hackers archive on startup and periodically
2. **Analyze Threads**: Threads are automatically categorized based on activity patterns
3. **View Dashboard**: Open http://localhost:3000 to see thread status and metrics

## Thread Classification

Threads are classified based on:

- **Message frequency**: How many messages in the past N days
- **Author variation**: How many different contributors
- **Keywords**: Detection of "patch", "review", "committed", etc.
- **Stall detection**: No replies for 7+ days
- **Resolution markers**: Detection of resolved/merged patches

## API Endpoints

- `GET /api/threads` - List all threads with filtering
- `GET /api/threads/:id` - Get thread details
- `GET /api/threads/:id/messages` - Get all messages in thread
- `GET /api/stats` - Get overall statistics
- `POST /api/sync` - Manually trigger IMAP mail sync
- `POST /api/sync/mbox` - Upload and parse mbox file
- `POST /api/sync/mbox/all` - Sync all stored mbox files

## Contributing

Contributions welcome! Please feel free to submit PRs or issues.

## License

MIT
