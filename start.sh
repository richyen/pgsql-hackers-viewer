#!/bin/bash
# Quick setup script for PostgreSQL Mailing List Thread Analyzer

echo "PostgreSQL Mailing List Thread Analyzer"
echo "========================================"
echo ""

# Check for Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    echo "   https://docs.docker.com/get-docker/"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    echo "   https://docs.docker.com/compose/install/"
    exit 1
fi

echo "✅ Docker and Docker Compose found"
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo "⚠️  .env file not found. Creating from template..."
    cp .env.example .env
    echo "✅ Created .env file. Please update with your settings if needed."
else
    echo "✅ .env file found"
fi

echo ""
echo "Starting services..."
echo ""

# Start services
docker-compose up

echo ""
echo "Services started!"
echo ""
echo "Access the application:"
echo "  Frontend: http://localhost:3000"
echo "  Backend:  http://localhost:8080/api"
echo "  Database: localhost:5432"
echo ""
echo "To stop: Press Ctrl+C or run 'docker-compose down'"
