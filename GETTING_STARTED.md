# Getting Started Guide

## Quick Start with Docker (Recommended)

The easiest way to get the application running is with Docker Compose:

```bash
# Clone/navigate to project directory
cd pgsql-analyzer

# Start all services (PostgreSQL, Backend, Frontend)
docker-compose up

# Access the application
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080/api
```

The first run will:
1. Create PostgreSQL database with schema
2. Build and start the Go backend API
3. Build and start the TypeScript React frontend

**Mbox sync (Docker):** To use "Sync mbox files", add `.mbox` files to the project’s `data/` folder (created on first run), then click Sync mbox files in the UI. Or use **Upload Mbox** to upload a file; it is stored in the same directory.

## Local Development Setup

### Prerequisites

- Go 1.21+
- Node.js 18+
- PostgreSQL 14+

### Backend Setup

```bash
cd backend

# Download dependencies
go mod download

# Run the server
go run main.go
```

The backend API will start on `http://localhost:8080`

### Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Start development server
npm start
```

The frontend will start on `http://localhost:3000`

### Database Setup

If running locally without Docker, create the PostgreSQL database:

```bash
psql -U postgres

CREATE DATABASE pgsql_analyzer;
\c pgsql_analyzer

# Backend will auto-run migrations on startup
```

## Configuration

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Key variables:
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` - Database connection
- `API_PORT`, `API_HOST` - API server settings
- `MAIL_IMAP_HOST`, `MAIL_IMAP_PORT`, `MAIL_USERNAME`, `MAIL_PASSWORD` - Mail server (optional)
- `REACT_APP_API_URL` - Frontend API endpoint

## Project Structure

```
.
├── backend/                 # Go API server
│   ├── main.go             # Entry point
│   ├── config/             # Configuration
│   ├── models/             # Data models
│   ├── db/                 # Database setup
│   ├── parser/             # Mail parser
│   ├── analyzer/           # Thread analysis logic
│   ├── api/                # API routes
│   ├── go.mod & go.sum     # Go dependencies
│   └── .gitignore
├── frontend/               # React TypeScript UI
│   ├── src/
│   │   ├── components/     # React components
│   │   ├── api/           # API client
│   │   ├── App.tsx        # Main app component
│   │   └── index.tsx      # React entry point
│   ├── public/            # Static files
│   ├── package.json
│   └── tsconfig.json
├── docker-compose.yml      # Docker orchestration
├── Dockerfile.backend      # Backend container
├── Dockerfile.frontend     # Frontend container
├── README.md              # Project overview
└── .env.example           # Environment template
```

## API Endpoints

### Threads

- `GET /api/threads?status=in-progress&limit=50` - List threads with optional filtering
- `GET /api/threads/{id}` - Get thread details
- `GET /api/threads/{id}/messages` - Get all messages in thread

### Statistics

- `GET /api/stats` - Get overall statistics and thread status breakdown

### Administration

- `POST /api/sync` - Manually trigger mail synchronization

## Thread Status Classification

Threads are automatically classified into four categories:

1. **In Progress** - Active development detected
   - Has patch keywords ("patch", "diff", "commit")
   - Has review activity ("review", "LGTM", "approved")
   - Recent activity (last message < 7 days)

2. **Discussion** - Under discussion, no active work
   - No patch keywords detected
   - Recent activity
   - Multiple contributors engaged

3. **Stalled** - No recent activity
   - No messages for 7-30 days
   - May pick up later

4. **Abandoned** - Long-term inactivity
   - No messages for 30+ days
   - Few messages overall (< 5)

## Troubleshooting

### Docker Issues

```bash
# Clean up containers and volumes
docker-compose down -v

# Rebuild from scratch
docker-compose build --no-cache

# View logs
docker-compose logs -f backend
docker-compose logs -f frontend
```

### Database Connection

If you get "connection refused" errors:

1. Ensure PostgreSQL is running: `docker-compose ps postgres`
2. Check database credentials in `.env`
3. Verify network: `docker network ls`

### Frontend Not Connecting to Backend

1. Check `REACT_APP_API_URL` in `.env`
2. Verify backend is running: `curl http://localhost:8080/api/health`
3. Check browser console for CORS errors

## Performance Considerations

- Thread analysis runs on message insert
- Indexes on `thread_id`, `status`, `last_message_at` for fast queries
- Update message limit to `limit=50` for large datasets
- Stats cache refreshed every 30 seconds in UI

## Next Steps

1. Configure mail credentials to enable mail sync
2. Run initial sync: POST `/api/sync`
3. View threads in dashboard
4. Filter by status to find work to pick up

## Support

For issues or questions, refer to the PostgreSQL mailing list archives or contribute improvements to this tool!
