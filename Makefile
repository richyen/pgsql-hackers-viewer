.PHONY: help docker-up docker-down docker-logs backend-run frontend-run backend-test docker-build clean

help:
	@echo "PostgreSQL Mailing List Thread Analyzer"
	@echo ""
	@echo "Available commands:"
	@echo "  docker-up          Start all services with Docker Compose"
	@echo "  docker-down        Stop all services"
	@echo "  docker-logs        View Docker logs"
	@echo "  docker-build       Build Docker images"
	@echo "  backend-run        Run backend locally (requires PostgreSQL)"
	@echo "  frontend-run       Run frontend locally"
	@echo "  backend-test       Run backend tests"
	@echo "  clean              Clean build artifacts and containers"

docker-up:
	docker-compose up

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-build:
	docker-compose build --no-cache

backend-run:
	cd backend && go run main.go

frontend-run:
	cd frontend && npm install && npm start

backend-test:
	cd backend && go test ./...

clean:
	docker-compose down -v
	rm -rf backend/vendor
	rm -rf frontend/node_modules
	rm -rf frontend/build

install-hooks:
	@echo "Setting up git hooks (optional)"
	@mkdir -p .git/hooks
	@echo "#!/bin/bash" > .git/hooks/pre-commit
	@echo "cd backend && go fmt ./..." >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit

dev-deps:
	@echo "Checking development dependencies..."
	@which docker-compose > /dev/null || echo "Please install Docker Compose"
	@which go > /dev/null || echo "Please install Go 1.21+"
	@which node > /dev/null || echo "Please install Node.js 18+"
	@echo "All dependencies checked!"
