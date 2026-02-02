# Quick Reference

## Start Development

```bash
cd pgsql-analyzer
docker-compose up
# Access: http://localhost:3000
```

## Project Layout

| Path | Purpose |
|------|---------|
| `backend/main.go` | Go API entry point |
| `backend/api/routes.go` | REST API endpoints |
| `backend/analyzer/analyzer.go` | Thread classification logic |
| `backend/parser/parser.go` | IMAP mail parser |
| `backend/models/` | Data structures |
| `backend/config/` | Configuration loader |
| `backend/db/` | Database setup and migrations |
| `frontend/src/App.tsx` | React main component |
| `frontend/src/api/client.ts` | API HTTP client |
| `frontend/src/components/` | React components |
| `docker-compose.yml` | Service orchestration |
| `Dockerfile.backend` | Go backend container |
| `Dockerfile.frontend` | React frontend container |
| `.env` | Environment variables (local dev) |
| `.env.example` | Environment template |

## API Quick Reference

### GET /api/health
Health check
```bash
curl http://localhost:8080/api/health
```

### GET /api/threads
List threads (with optional filtering)
```bash
curl "http://localhost:8080/api/threads"
curl "http://localhost:8080/api/threads?status=in-progress"
curl "http://localhost:8080/api/threads?limit=20"
```

### GET /api/threads/{id}
Get thread details
```bash
curl http://localhost:8080/api/threads/thread-id
```

### GET /api/threads/{id}/messages
Get messages in thread
```bash
curl http://localhost:8080/api/threads/thread-id/messages
```

### GET /api/stats
Get statistics
```bash
curl http://localhost:8080/api/stats
```

### POST /api/sync
Trigger IMAP mail sync
```bash
curl -X POST http://localhost:8080/api/sync
```

### POST /api/sync/mbox
Upload and parse mbox file
```bash
curl -X POST http://localhost:8080/api/sync/mbox \
  -F "file=@archive.mbox"
```

### POST /api/sync/mbox/all
Sync all stored mbox files
```bash
curl -X POST http://localhost:8080/api/sync/mbox/all
```

## Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | `postgres` |
| `DB_NAME` | Database name | `pgsql_analyzer` |
| `API_PORT` | API port | `8080` |
| `API_HOST` | API bind host | `0.0.0.0` |
| `MAIL_IMAP_HOST` | IMAP server | `imap.gmail.com` |
| `MAIL_IMAP_PORT` | IMAP port | `993` |
| `MAIL_USERNAME` | Email username | `user@gmail.com` |
| `MAIL_PASSWORD` | Email password | `app-password` |
| `DATA_DIR` | Mbox file storage directory | `./data` |
| `REACT_APP_API_URL` | Frontend API URL | `http://localhost:8080` |

## Common Commands

```bash
# Start all services
docker-compose up

# Stop services
docker-compose down

# View logs
docker-compose logs -f backend
docker-compose logs -f frontend

# Run backend locally
cd backend && go run main.go

# Run frontend locally
cd frontend && npm install && npm start

# Run backend tests
cd backend && go test ./...

# Format backend code
cd backend && go fmt ./...

# Build frontend for production
cd frontend && npm run build

# Remove all containers and volumes
docker-compose down -v
```

## Key Database Queries

```sql
-- View all threads
SELECT * FROM threads ORDER BY last_message_at DESC;

-- View threads by status
SELECT * FROM threads WHERE status = 'in-progress';

-- View messages in a thread
SELECT * FROM messages WHERE thread_id = '...' ORDER BY created_at;

-- Get statistics
SELECT 
    status,
    COUNT(*) as count
FROM threads
GROUP BY status;

-- Find stalled threads
SELECT * FROM threads 
WHERE status = 'stalled' AND last_message_at < NOW() - INTERVAL '7 days';

-- Find threads with most activity
SELECT * FROM threads 
ORDER BY message_count DESC 
LIMIT 10;
```

## Troubleshooting

### Port Already in Use
```bash
# Kill process using port 3000
lsof -ti:3000 | xargs kill -9

# Kill process using port 8080
lsof -ti:8080 | xargs kill -9

# Kill process using port 5432
lsof -ti:5432 | xargs kill -9
```

### Database Connection Refused
```bash
# Verify PostgreSQL is running
docker-compose ps postgres

# Check database logs
docker-compose logs postgres

# Recreate database
docker-compose down -v
docker-compose up
```

### Frontend Can't Connect to Backend
```bash
# Verify backend is running
curl http://localhost:8080/api/health

# Check REACT_APP_API_URL in .env
cat .env | grep REACT_APP_API_URL

# Check browser console for CORS errors
```

### Package Dependencies Missing
```bash
# Backend
cd backend && go mod download

# Frontend
cd frontend && npm install
```

## Development Workflow

1. **Feature**: Edit code in `backend/` or `frontend/`
2. **Test**: Code auto-reloads in Docker dev containers
3. **Verify**: Check API endpoints or UI in browser
4. **Commit**: `git add . && git commit -m "feature: description"`
5. **Push**: `git push origin feature-branch`

## Performance Tips

- Use `?limit=20` for large result sets
- Backend handles up to 50 threads per page
- Stats cache refreshed every 30 seconds
- Database indexes optimized for `status` and `last_message_at`

## Resources

- [Go Documentation](https://golang.org/doc/)
- [React Documentation](https://react.dev)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Docker Documentation](https://docs.docker.com/)
- [pgsql-hackers Archive](https://www.postgresql.org/list/pgsql-hackers/)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on contributing to this project.

## Additional Docs

- [README.md](README.md) - Project overview
- [GETTING_STARTED.md](GETTING_STARTED.md) - Setup guide
- [MBOX_GUIDE.md](MBOX_GUIDE.md) - Mbox file support
- [ARCHITECTURE.md](ARCHITECTURE.md) - System design
- [CONTRIBUTING.md](CONTRIBUTING.md) - Development guide
