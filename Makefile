.PHONY: help setup-python run-embedding run-go-api test-api docker-up docker-down clean

help:
	@echo "Document Hub - Development Commands"
	@echo ""
	@echo "  make setup-python    - Set up Python virtual environment and install dependencies"
	@echo "  make run-embedding   - Run the Python embedding service"
	@echo "  make run-go-api      - Run the Go API server"
	@echo "  make test-api        - Test the API endpoints"
	@echo "  make docker-up       - Start all services with Docker Compose"
	@echo "  make docker-down     - Stop all Docker services"
	@echo "  make clean           - Clean up build artifacts"

setup-python:
	@echo "Setting up Python environment..."
	cd embedding-service && python3 -m venv venv
	cd embedding-service && ./venv/bin/pip install -r requirements.txt
	@echo "Python environment ready!"

run-embedding:
	@echo "Starting Python embedding service on port 8001..."
	cd embedding-service && python3 -m app.main

run-go-api:
	@echo "Starting Go API server on port 8000..."
	cd go-api && export PATH="/opt/homebrew/opt/postgresql@17/bin:$$PATH" && go run cmd/api/main.go

test-api:
	@echo "Testing API endpoints..."
	@echo "\n1. Health check (Go API):"
	curl -s http://localhost:8000/health | jq .
	@echo "\n2. Health check (Python Embedding Service):"
	curl -s http://localhost:8001/health | jq .
	@echo "\n3. List documents:"
	curl -s http://localhost:8000/api/v1/documents | jq .

docker-up:
	@echo "Starting all services with Docker Compose..."
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	sleep 10
	@echo "Services are up!"
	docker-compose ps

docker-down:
	@echo "Stopping all Docker services..."
	docker-compose down

clean:
	@echo "Cleaning up..."
	rm -rf embedding-service/venv
	rm -rf embedding-service/__pycache__
	rm -rf embedding-service/app/__pycache__
	cd go-api && go clean
	@echo "Clean complete!"
make swagger

swagger:
	@echo "Generating Swagger documentation..."
	cd go-api && ~/go/bin/swag init -g cmd/api/main.go --output docs
