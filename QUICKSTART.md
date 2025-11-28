# Quick Start Guide

Updated with Web UI instructions!

## Prerequisites

- PostgreSQL 17 with pgvector (already installed)
- Go 1.22+ (for local development)
- Python 3.11+ (for local development)
- Docker & Docker Compose (for containerized deployment)

## Local Development Setup

### 1. Start PostgreSQL 17

PostgreSQL 17 with pgvector is already running at `localhost:5432`.

### 2. Start Python Embedding Service

**Terminal 1:**

```bash
cd embedding-service
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python -m app.main
```

Service available at `http://localhost:8001`

### 3. Start Go API Server

**Terminal 2:**

```bash
cd go-api
export PATH="/opt/homebrew/opt/postgresql@17/bin:$PATH"
go run cmd/api/main.go
```

### 4. Access the Application

**🎉 Open your browser to: http://localhost:8000**

Available endpoints:
- **Web UI**: http://localhost:8000 (NEW!)
- **API**: http://localhost:8000/api/v1/*
- **Swagger**: http://localhost:8000/swagger/index.html
- **Health**: http://localhost:8000/health

## Using the Web UI

### Search Documents

1. Enter your search query in the search box
2. Choose search type:
   - **Hybrid** (recommended): Best of both worlds
   - **Semantic**: Meaning-based, AI-powered
   - **Full-text**: Traditional keyword search
3. View ranked results with relevance scores

### Add Documents

1. Click "+ Add Document" button
2. Fill in:
   - Title (required)
   - Content (required)
   - Source URL, type, tags (optional)
3. Document is automatically chunked and embedded

### Manage Documents

- **View**: Click any document card
- **Edit**: Click pencil icon
- **Delete**: Click trash icon

## Using the API (cURL)

### Create Document

```bash
curl -X POST http://localhost:8000/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Getting Started with Go",
    "content": "Go is a statically typed, compiled programming language...",
    "doc_type": "tutorial",
    "metadata": {
      "tags": ["go", "programming"]
    }
  }'
```

### Search

```bash
# Hybrid search (recommended)
curl "http://localhost:8000/api/v1/search?q=go%20programming&type=hybrid"

# Semantic search
curl "http://localhost:8000/api/v1/search?q=how%20to%20learn%20Go&type=semantic"

# Full-text search
curl "http://localhost:8000/api/v1/search?q=golang&type=fulltext"
```

## Docker Deployment

```bash
# Start all services
docker-compose up -d

# Access application
open http://localhost:8000
```

## Troubleshooting

### PostgreSQL not running
```bash
brew services start postgresql@17
```

### Python service fails
```bash
cd embedding-service
source venv/bin/activate
pip install --upgrade -r requirements.txt
```

### Go API can't connect
```bash
# Check services
curl http://localhost:8001/health  # Python service
psql -h localhost -p 5432 -d doc_hub -c "SELECT 1;"  # PostgreSQL
```

### Web UI shows "API Offline"
- Make sure Go API is running on port 8000
- Check console for CORS errors
- Verify API_BASE_URL in `frontend/js/app.js`

## Next Steps

1. **Add your documentation** using the web UI or API
2. **Try different search modes** to see how they compare
3. **Explore Swagger docs** at http://localhost:8000/swagger/index.html
4. **Check the README** for advanced features

For more details, see [README.md](README.md).
