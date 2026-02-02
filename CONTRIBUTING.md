# Contributing to PostgreSQL Mailing List Thread Analyzer

## Overview

This project helps PostgreSQL contributors identify which mailing list threads have active development work. Contributions are welcome!

## Development Workflow

### 1. Local Setup

```bash
# Clone the repository
cd pgsql-analyzer

# Copy environment template
cp .env.example .env

# Start services with Docker
docker-compose up

# Or run locally
# Terminal 1: Backend
cd backend && go run main.go

# Terminal 2: Frontend  
cd frontend && npm install && npm start
```

### 2. Making Changes

#### Backend Changes

1. Modify files in `backend/` directory
2. Changes auto-reload in Docker dev containers
3. Test with: `go test ./...`
4. Format code: `go fmt ./...`

Key files:
- `models/`: Data structures
- `api/routes.go`: API endpoints
- `analyzer/analyzer.go`: Thread classification logic
- `parser/parser.go`: Mail parsing logic
- `db/db.go`: Database operations

#### Frontend Changes

1. Modify files in `frontend/src/` directory
2. Changes auto-reload in dev server
3. Components in `components/` directory
4. API client in `api/client.ts`

Key files:
- `App.tsx`: Main application component
- `components/`: React components
- `api/client.ts`: API communication

### 3. Code Style

**Go**:
```bash
go fmt ./...
go vet ./...
```

**TypeScript/React**:
- Use 2-space indentation
- Use `const` by default, `let` when needed
- Use functional components with hooks
- Add types to function parameters

## Feature Ideas

### High Priority
- [ ] Connect to real pgsql-hackers archive
- [ ] Implement IMAP mail sync
- [ ] Add search functionality
- [ ] Add date range filtering
- [ ] Export thread data (CSV/JSON)

### Medium Priority
- [ ] User authentication
- [ ] Saved thread collections
- [ ] Email notifications for new threads
- [ ] Thread statistics charts
- [ ] Advanced filtering (author, keyword)

### Low Priority
- [ ] Mobile app
- [ ] Browser extension
- [ ] Slack integration
- [ ] Calendar view
- [ ] Markdown email rendering

## Bug Fixes

Found a bug? Here's how to fix it:

1. **Describe the issue**: What's the expected vs actual behavior?
2. **Reproduce**: Provide steps to reproduce
3. **Locate**: Find which component is affected
4. **Fix**: Make minimal changes to fix the issue
5. **Test**: Verify the fix works

Common issues:
- **Frontend not connecting**: Check `REACT_APP_API_URL` in `.env`
- **Database errors**: Verify PostgreSQL is running and credentials are correct
- **Port conflicts**: Check if ports 3000, 8080, 5432 are already in use

## Testing

### Backend Testing

```bash
cd backend

# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./api -v

# Run with coverage
go test -cover ./...
```

### Frontend Testing

```bash
cd frontend

# Run tests (interactive)
npm test

# Run with coverage
npm test -- --coverage
```

### Manual Testing

1. **API Testing**: Use curl or Postman
   ```bash
   curl http://localhost:8080/api/health
   curl http://localhost:8080/api/threads
   curl http://localhost:8080/api/stats
   ```

2. **UI Testing**: Check all pages in browser
   - Thread list filters
   - Message thread display
   - Statistics panel
   - Sync functionality

## Documentation

When adding features, update:
- `README.md`: High-level overview
- `GETTING_STARTED.md`: Setup and usage
- `ARCHITECTURE.md`: System design
- Code comments: Explain complex logic
- API docs: If adding endpoints

### Documentation Standards

```go
// ExportedFunction explains what this exported function does.
// It takes these parameters and returns these values.
// Use it for this purpose.
func ExportedFunction(arg string) (result string, err error) {
    // ...
}
```

```typescript
/**
 * Component description.
 * @param param1 - Description of param1
 * @returns Description of return value
 */
export const MyComponent: React.FC<Props> = ({ param1 }) => {
    // ...
};
```

## Pull Request Process

1. **Create a branch**: `git checkout -b feature/my-feature`
2. **Make changes**: Commit with clear messages
3. **Test**: Verify all tests pass
4. **Update docs**: Update README, comments, etc.
5. **Create PR**: Describe changes and motivation

PR Template:
```markdown
## Description
Brief description of changes

## Motivation
Why is this change needed?

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation
- [ ] Performance improvement

## Testing
How was this tested?

## Screenshots (if UI change)
Add before/after screenshots

## Checklist
- [ ] Code follows style guidelines
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] No breaking changes
```

## Architecture Guidelines

### Adding a New API Endpoint

1. Add route in `api/routes.go`:
```go
router.HandleFunc("/api/newfeature", newFeatureHandler(db)).Methods("GET")
```

2. Create handler:
```go
func newFeatureHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Implementation
    }
}
```

3. Update frontend client in `api/client.ts`:
```typescript
export const threadAPI = {
    newFeature: () => api.get('/newfeature'),
}
```

4. Create React component to use it

### Adding a New Frontend Component

1. Create component in `src/components/MyComponent.tsx`
2. Add styles in `src/components/MyComponent.module.css`
3. Import and use in `App.tsx`
4. Export from component file

Example:
```typescript
interface MyComponentProps {
    prop1: string;
}

export const MyComponent: React.FC<MyComponentProps> = ({ prop1 }) => {
    return <div className={styles.container}>{prop1}</div>;
};
```

### Adding a New Database Migration

1. Edit `db/db.go` in `RunMigrations()` function
2. Add SQL schema changes
3. Create indexes for new columns
4. Test with `docker-compose down -v` then `docker-compose up`

## Performance Optimization

- Use database indexes for frequently queried columns
- Limit query results with pagination
- Cache stats data (refreshed every 30s)
- Use proper SQL query planning
- Optimize React re-renders with React.memo

## Debugging

### Backend Debugging

```bash
# View logs
docker-compose logs backend

# View database
docker exec -it pgsql_analyzer_db psql -U postgres -d pgsql_analyzer

# Query threads
SELECT * FROM threads LIMIT 10;

# View messages
SELECT * FROM messages WHERE thread_id = '...' ORDER BY created_at;
```

### Frontend Debugging

- Open browser DevTools (F12)
- Check Console tab for errors
- Check Network tab for API calls
- Use React DevTools extension

## Questions?

- Check existing documentation
- Look at similar code in the project
- Ask in issues or discussions
- Check PostgreSQL mailing list for context

## Code of Conduct

- Be respectful and inclusive
- Help others learn
- Review code thoughtfully
- Give constructive feedback

Thank you for contributing! 🚀
