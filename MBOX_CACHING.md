# Mbox File Caching Strategy

## Overview

The application now supports different behaviors for handling mbox files in development vs production environments.

## Development Mode (Default)

**Behavior:**
- Downloads mbox files once and caches them locally in the `data/` directory
- On subsequent syncs, reuses existing cached files instead of re-downloading
- Files are never deleted, allowing for fast database reloads during development

**Benefits:**
- Faster development iteration cycles
- Reduced bandwidth usage
- No need to re-download static historical data repeatedly
- Database can be wiped and reloaded quickly

**Configuration:**
```yaml
environment:
  ENV: development  # or omit - defaults to development
```

## Production Mode

**Behavior:**
- Always downloads fresh mbox files from PostgreSQL.org
- Overwrites any existing files with the latest version
- Automatically deletes mbox files after successful ingestion into the database
- Ensures data is always up-to-date and minimizes disk usage

**Benefits:**
- Always has the latest data
- Minimal disk space usage
- No stale cached files

**Configuration:**
```yaml
environment:
  ENV: production
```

## Implementation Details

### Configuration

The mode is controlled by the `ENV` environment variable:
- `ENV=development` (default): Cache files, skip downloads if file exists
- `ENV=production`: Always download fresh, cleanup after ingestion

The configuration automatically sets `CleanupMboxFiles` flag based on the environment.

### File Management

**Dev Mode:**
1. Check if mbox file exists before downloading
2. If exists, log "Using cached mbox file" and use it
3. If missing, download as normal
4. Keep all files in `data/` directory

**Production Mode:**
1. Always download files (overwrites existing)
2. Parse and ingest into database
3. Delete mbox file after successful ingestion
4. Log cleanup actions

### Docker Compose Configuration

The `docker-compose.yml` includes the environment setting:

```yaml
backend:
  environment:
    ENV: development  # Change to 'production' for production deployments
```

## Usage Scenarios

### Scenario 1: Fresh Development Setup
```bash
# Start services (dev mode by default)
docker-compose up

# Sync mbox files - downloads all files
curl -X POST http://localhost:8080/api/sync/mbox/all

# Files are cached in ./data/
```

### Scenario 2: Database Reset in Dev
```bash
# Clear database
curl -X POST http://localhost:8080/api/reset

# Re-sync - uses cached files, very fast
curl -X POST http://localhost:8080/api/sync/mbox/all
```

### Scenario 3: Production Deployment
```yaml
# docker-compose.prod.yml
backend:
  environment:
    ENV: production  # Always download fresh, cleanup after
```

```bash
docker-compose -f docker-compose.prod.yml up
```

## Disk Space Considerations

### Development
- Mbox files accumulate over time (~10-50 MB per month)
- For 12 months: ~120-600 MB
- Files persist until manually deleted

### Production
- Only temporary storage during ingestion
- Files deleted immediately after processing
- Minimal disk footprint

## Manual Cache Management

To clear cached mbox files in development:

```bash
# Remove all cached mbox files
rm data/pgsql-hackers.*

# Remove specific month
rm data/pgsql-hackers.202501
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ENV` | `development` | Environment mode (`development` or `production`) |
| `DATA_DIR` | `./data` | Directory for storing mbox files |

The `CleanupMboxFiles` config flag is automatically derived from `ENV`:
- `ENV=development` → `CleanupMboxFiles=false`
- `ENV=production` → `CleanupMboxFiles=true`

## Logging

The application logs its caching behavior:

**Dev Mode:**
```
Dev mode: Using cached mbox files if available
Using cached mbox file: /app/data/pgsql-hackers.202501
```

**Production Mode:**
```
Production mode: Downloading fresh mbox files
Downloaded pgsql-hackers.202501 (15234567 bytes)
Cleaned up mbox file: /app/data/pgsql-hackers.202501
```

## Best Practices

1. **Development**: Keep `ENV=development` for fast iterations
2. **Production**: Always set `ENV=production` to ensure fresh data
3. **CI/CD**: Use production mode in automated deployments
4. **Testing**: Can use dev mode with pre-seeded cache for faster tests
5. **Disk Management**: In dev, periodically clean old cached files if disk space is limited
