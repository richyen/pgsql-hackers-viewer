# PostgreSQL Mailing List Thread Analyzer

A tool to parse and analyze the pgsql-hackers mailing list to identify which threads are actively being worked on versus those that are still under discussion or abandoned.

## Problem Statement

The PostgreSQL community uses the pgsql-hackers mailing list for development discussion, but there's no clear way to determine which threads have active development work, which are waiting for feedback, and which have been abandoned. This tool categorizes threads by activity status to help contributors avoid duplicating effort.

## Features

- **Mailing List Archive Parser**: Fetches and parses messages from pgsql-hackers archive
- **Mbox File Support**: Import messages from mbox files (standard mail archive format)
- **Sync Progress Tracking**: Real-time progress bar showing months synced and latest message date
- **Thread Activity Analysis**: Categorizes threads by status:
  - **In Progress**: Active development, patches being worked on
  - **Under Discussion**: Active discussion but no current patch work
  - **Stalled**: No recent activity (7-30 days)
  - **Abandoned**: No activity for 30+ days
- **Web Dashboard**: Real-time view of thread status and activity metrics
- **Message Viewing**: View full message content inline with expandable panels
- **Archive Links**: Direct links to postgresql.org archive for each message
- **Filtering & Search**: Filter by activity status with helpful tooltips
- **Help Documentation**: Built-in guide explaining classification criteria
- **Hot Reload**: Frontend auto-reloads on code changes in development mode

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

1. **Start the Application**: Run `docker-compose up` to start all services
2. **Sync Messages**: Click "Sync Mbox Files" to download archives from postgresql.org (downloads last 365 days initially)
3. **Monitor Progress**: Watch the progress bar to see sync status and latest message timestamp
4. **Browse Threads**: Use status filters to view threads by classification
5. **View Messages**: Click a thread to see messages, expand to read content, or click "View Archive" to see on postgresql.org
6. **Get Help**: Click the "? Help" button for detailed classification criteria
7. **View Dashboard**: All metrics update in real-time at http://localhost:3000

## Thread Classification

Threads are classified based on:

- **Message frequency**: How many messages in the past N days
- **Author variation**: How many different contributors
- **Keywords**: Detection of "patch", "review", "committed", "[PATCH]", etc.
- **Stall detection**: No replies for 7-30 days (stalled) or 30+ days (abandoned)
- **Review patterns**: Detection of code review activity ("LGTM", "looks good", "reviewed-by")
- **Resolution markers**: Detection of resolved/merged patches

Click the "? Help" button in the application for detailed classification criteria.

## API Endpoints

- `GET /api/threads` - List all threads with filtering
- `GET /api/threads/:id` - Get thread details
- `GET /api/threads/:id/messages` - Get all messages in thread (includes message body)
- `GET /api/stats` - Get overall statistics
- `GET /api/sync/progress` - Get current sync progress (months synced, latest message)
- `POST /api/sync/mbox` - Upload and parse mbox file
- `POST /api/sync/mbox/all` - Sync all mbox archives from postgresql.org
- `POST /api/reset` - Clear all data for fresh start

## New Features

### Mbox File Caching (Dev vs Production)
The application now supports different mbox file handling strategies:
- **Development Mode** (default): Downloads files once and caches them locally for fast database reloads
- **Production Mode**: Always downloads fresh files and deletes them after ingestion

See [MBOX_CACHING.md](MBOX_CACHING.md) for detailed documentation on caching behavior.

### Development Mode with Hot Reload
The frontend now runs in development mode with source code mounted as a volume. Any changes to frontend code will automatically reload in the browser without rebuilding the Docker image.

### Sync Progress Tracking
A real-time progress bar shows:
- Current month being synced
- Progress: X / Y months completed  
- Latest message fetched timestamp
- Whether a sync is currently active

### Message Content & Archive Links
- Click "Show Message Content" to expand full message text inline
- Click "View Archive" to open the message on postgresql.org
- All message bodies are now stored and viewable

### Interactive Help System
Click the "? Help" button to view:
- Detailed classification criteria for each status
- How the analyzer determines thread status
- Feature descriptions and usage tips
- Syncing strategies and recommendations

## Contributing

Contributions welcome! Please feel free to submit PRs or issues.

## License

MIT
