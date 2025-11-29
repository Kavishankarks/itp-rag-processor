# Document Hub

A high-performance documentation platform that aggregates and indexes documentation with intelligent search capabilities using full-text and semantic vector search.

## Architecture

Document Hub uses a microservice architecture for optimal performance:

```
┌─────────────────────────────────────────────────┐
│                   Client/User                    │
└───────────────────┬─────────────────────────────┘
                    │
                    ▼
        ┌───────────────────────┐
        │   Go API Server       │ ◄─── Fast, efficient API layer
        │   (Fiber Framework)   │
        │   Port: 8000          │
        └───┬──────────────┬────┘
            │              │
            │              └──────────────┐
            ▼                             ▼
┌───────────────────────┐     ┌─────────────────────────┐
│  PostgreSQL 17        │     │ Python ML Service       │
│  + pgvector 0.8.1     │     │ (FastAPI)               │
│  + pg_trgm            │     │ Port: 8001              │
│  Port: 5432           │     │                         │
│                       │     │ - Sentence Transformers │
│ - Documents           │     │ - Text Chunking         │
│ - Embeddings          │     │ - Embedding Generation  │
│ - Full-text Search    │     └─────────────────────────┘
│ - Vector Similarity   │
└───────────────────────┘
```

### Why This Architecture?

1. **Go API Layer**: High performance, low latency for CRUD operations and search
2. **Python ML Service**: Best-in-class ML libraries for embedding generation
3. **PostgreSQL + pgvector**: Single source of truth with both full-text and vector search

## Features

### v1 Features

- **Web User Interface**
  - Modern, responsive design with Tailwind CSS
  - Real-time search with 3 modes (full-text, semantic, hybrid)
  - Document CRUD operations (Create, Read, Update, Delete)
  - Interactive document viewer
  - Real-time statistics dashboard
  - Mobile-friendly interface

- **Document Management**
  - Create, read, update, delete documentation
  - **Duplicate prevention**: Unique title constraint prevents duplicate documents
  - Auto-chunking of large documents
  - Metadata and tagging support
  - Source URL tracking

- **Intelligent Search**
  - **Full-text search**: PostgreSQL `pg_trgm` + `ts_vector` for keyword matching
  - **Semantic search**: pgvector cosine similarity for meaning-based search
  - **Hybrid search**: Combines both with weighted scoring (40% full-text + 60% semantic)
  - Ranked results with relevance scores
  - Context snippets in search results

- **Vector Embeddings**
  - Automatic embedding generation using sentence-transformers
  - Model: `all-MiniLM-L6-v2` (384 dimensions)
  - Efficient HNSW indexing for fast similarity search

- **API Documentation**
  - Interactive Swagger UI at `/swagger/index.html`
  - OpenAPI 3.0 specification
  - Test endpoints directly from browser

### 🆕 RAG Processing Pipeline

The **RAG (Retrieval-Augmented Generation) Processing Pipeline** is a powerful new feature that automatically processes course curricula through multiple stages:

- **Automated Processing**: Parse curriculum → Web Search → Normalize → Chunk → Embed → Store
- **Web Search Integration**: Enriches topics with additional context from DuckDuckGo
- **Text Normalization**: Cleans, deduplicates, and standardizes content
- **Flexible Input**: Supports JSON, YAML, and Markdown formats
- **Progress Tracking**: Real-time status monitoring for each pipeline stage
- **Async Processing**: Non-blocking execution with status endpoints

**Quick Example:**
```bash
# Start a pipeline
./test-pipeline.sh

# Or manually:
curl -X POST http://localhost:8000/api/v1/pipeline/start \
  -H "Content-Type: application/json" \
  -d '{
    "curriculum": {
      "title": "Machine Learning Basics",
      "modules": [{
        "name": "Supervised Learning",
        "topics": ["Linear Regression", "Decision Trees"]
      }]
    },
    "config": {
      "web_search_enabled": true,
      "search_results_per_topic": 5
    }
  }'
```

📖 **[View Full Pipeline Documentation →](PIPELINE.md)**

## Quick Start

### Prerequisites

- Docker and Docker Compose
- OR: Go 1.22+, Python 3.11+, PostgreSQL 17

### Option 1: Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/kavishankarks/itp-rag-processor.git
cd itp-rag-processor

# Start all services
docker-compose up -d

# Check service health
curl http://localhost:8000/health  # Go API
curl http://localhost:8001/health  # Python Embedding Service
```

### Option 2: Local Development

#### 1. Start PostgreSQL 17 with pgvector

```bash
# Using Homebrew (macOS)
brew install postgresql@17 pgvector
brew services start postgresql@17

# Create database
psql -h localhost -p 5432 -d postgres -c "CREATE DATABASE doc_hub;"
psql -h localhost -p 5432 -d doc_hub -c "CREATE EXTENSION vector; CREATE EXTENSION pg_trgm;"
```

#### 2. Start Python Embedding Service

```bash
cd embedding-service

# Create virtual environment
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt

# Run service
python -m app.main
# Service will be available at http://localhost:8001
```

#### 3. Start Go API Server

```bash
cd go-api

# Install dependencies
go mod download

# Run server
go run cmd/api/main.go
# API will be available at http://localhost:8000
```

## API Reference

### Interactive API Documentation (Swagger)

Once the Go API is running, visit:
- **Swagger UI**: http://localhost:8000/swagger/index.html
- **OpenAPI JSON**: http://localhost:8000/swagger/doc.json

This provides interactive documentation where you can test all endpoints directly from your browser.

### Document Endpoints

#### Create Document

```bash
POST /api/v1/documents
Content-Type: application/json

{
  "title": "Getting Started with PostgreSQL",
  "content": "PostgreSQL is a powerful, open source object-relational database system...",
  "source_url": "https://postgresql.org/docs/getting-started",
  "doc_type": "tutorial",
  "metadata": {
    "tags": ["database", "postgresql", "tutorial"],
    "author": "PostgreSQL Team"
  }
}
```

**Note:** Document titles must be unique. If a document with the same title already exists, you'll receive a `409 Conflict` response with the existing document's ID and a hint to update it instead.

#### Get Document

```bash
GET /api/v1/documents/{id}
```

#### List Documents

```bash
GET /api/v1/documents?skip=0&limit=20
```

#### Update Document

```bash
PUT /api/v1/documents/{id}
Content-Type: application/json

{
  "title": "Updated Title",
  "metadata": {
    "tags": ["updated"]
  }
}
```

#### Delete Document

```bash
DELETE /api/v1/documents/{id}
```

### Search Endpoints

#### Full-Text Search

```bash
GET /api/v1/search?q=postgresql%20query%20optimization&type=fulltext&limit=10
```

#### Semantic Search

```bash
GET /api/v1/search?q=how%20to%20improve%20database%20performance&type=semantic&limit=10
```

#### Hybrid Search (Recommended)

```bash
GET /api/v1/search?q=postgresql%20best%20practices&type=hybrid&limit=10
```

Response:

```json
{
  "query": "postgresql best practices",
  "search_type": "hybrid",
  "results": [
    {
      "document": {
        "id": 1,
        "title": "PostgreSQL Performance Tuning",
        "content": "...",
        "source_url": "...",
        "doc_type": "guide",
        "metadata": {},
        "created_at": "2025-10-12T19:00:00Z",
        "updated_at": "2025-10-12T19:00:00Z"
      },
      "score": 0.87,
      "snippet": "...PostgreSQL performance can be significantly improved by following these best practices..."
    }
  ],
  "count": 1
}
```

## Project Structure

```
itp-rag-processor/
├── go-api/                      # Go API Service
│   ├── cmd/api/
│   │   └── main.go              # Entry point
│   ├── internal/
│   │   ├── handlers/
│   │   │   ├── handlers.go      # CRUD handlers
│   │   │   └── search.go        # Search handlers
│   │   ├── models/
│   │   │   └── document.go      # Data models
│   │   ├── database/
│   │   │   └── database.go      # DB connection & migrations
│   │   └── embedding_client/
│   │       └── client.go        # Python service client
│   ├── Dockerfile
│   ├── go.mod
│   └── .env
│
├── embedding-service/           # Python ML Service
│   ├── app/
│   │   ├── main.py              # FastAPI app
│   │   ├── embedding_service.py # Embedding logic
│   │   └── __init__.py
│   ├── Dockerfile
│   ├── requirements.txt
│   └── .env
│
├── docker-compose.yml           # Service orchestration
└── README.md
```

## Configuration

### Environment Variables

#### Go API (.env)

```bash
DATABASE_URL=postgres://user@localhost:5432/doc_hub?sslmode=disable
EMBEDDING_SERVICE_URL=http://localhost:8001
APP_PORT=8000
APP_NAME=Document Hub API
DEBUG=true
```

#### Python Embedding Service (.env)

```bash
EMBEDDING_MODEL=all-MiniLM-L6-v2
EMBEDDING_DIMENSION=384
CHUNK_SIZE=500
CHUNK_OVERLAP=50
APP_PORT=8001
APP_NAME=Document Hub Embedding Service
DEBUG=true
```

## Performance Characteristics

- **Go API**: ~1-3ms response time for CRUD operations
- **Full-text search**: ~10-50ms for typical queries
- **Semantic search**: ~20-100ms depending on corpus size
- **Hybrid search**: ~30-120ms (combines both)
- **Embedding generation**: ~50-200ms per document (chunking + embedding)

## Database Schema

### Documents Table

```sql
CREATE TABLE documents (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  source_url TEXT,
  doc_type VARCHAR(100),
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Unique index to prevent duplicate titles
CREATE UNIQUE INDEX idx_documents_title_unique ON documents (title);

-- Full-text search index
CREATE INDEX idx_documents_content_gin ON documents
  USING gin(to_tsvector('english', content));
```

### Document Chunks Table

```sql
CREATE TABLE document_chunks (
  id SERIAL PRIMARY KEY,
  document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  chunk_text TEXT NOT NULL,
  chunk_index INTEGER NOT NULL,
  embedding vector(384),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_chunks_text_gin ON document_chunks
  USING gin(to_tsvector('english', chunk_text));

CREATE INDEX idx_chunks_embedding_hnsw ON document_chunks
  USING hnsw(embedding vector_cosine_ops);
```

## Future Roadmap

### v2 Features (Planned)

- GitHub integration for auto-ingesting repos
- Support for multiple embedding models
- API documentation scraping
- Real-time document updates
- Advanced filtering and faceted search
- User authentication and multi-tenancy
- Document versioning
- Analytics and usage tracking

### v3 Features (Vision)

- Agentic workflows
- IDE plugins (VS Code, JetBrains)
- Slack/Discord bots
- Contextual code suggestions
- Documentation quality scoring
- Multi-language support
- Cloud deployment templates (AWS, GCP, Azure)

## Development

### Running Tests

```bash
# Go tests
cd go-api
go test ./...

# Python tests
cd embedding-service
pytest
```

### Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License

## Support

For issues and questions, please open an issue on GitHub.

 I've created a modern, responsive web interface for Document Hub with all CRUD operations and intelligent search. Here's what was built:

  Frontend Structure

  frontend/
  ├── index.html           # Main UI with modals
  ├── css/
  │   └── style.css        # Custom styling
  └── js/
      └── app.js           # All API interactions & UI logic

  Features Implemented

  1. Search Interface
  - Real-time search with 3 modes:
    - Hybrid (Recommended) - Best results
    - Semantic - AI-powered meaning search
    - Full-text - Traditional keyword search
  - Search on Enter key
  - Relevance scoring displayed

  2. Document Management
  - ✅ Create - Modal form with validation
  - ✅ Read - Full document viewer modal
  - ✅ Update - Edit any document
  - ✅ Delete - With confirmation
  - Auto-chunking and embedding on save

  3. Dashboard
  - Total documents count
  - Search results counter
  - API status indicator
  - Document cards with tags

  4. Modern UI
  - Tailwind CSS for responsive design
  - Font Awesome icons
  - Smooth transitions and animations
  - Mobile-friendly layout
  - Clean, professional design

  How to Access

  Start the Go API:
  cd go-api
  go run cmd/api/main.go

  Open browser to:
  http://localhost:8000

  The Go API automatically serves the frontend at the root path!

  Available URLs

  - Web UI: http://localhost:8000
  - API: http://localhost:8000/api/v1/*
  - Swagger: http://localhost:8000/swagger/index.html
  - Health: http://localhost:8000/health

  Key Features

  1. No Build Step Required - Pure HTML/CSS/JS
  2. CDN Dependencies - Tailwind CSS & Font Awesome loaded via CDN
  3. CORS Configured - API properly configured for cross-origin requests
  4. Error Handling - User-friendly error messages
  5. Success Notifications - Toast-style notifications
  6. Loading States - Spinner while fetching data
  7. Responsive - Works on desktop, tablet, and mobile

  Usage Examples

  Search:
  - Enter "PostgreSQL optimization" → See ranked results
  - Switch between search types to compare
  - View relevance scores and snippets

  Add Document:
  - Click "+ Add Document"
  - Fill title and content (required)
  - Add tags like: postgresql, database, performance
  - Document is automatically chunked and embedded

  Edit/Delete:
  - Click pencil icon to edit
  - Click trash icon to delete

  Now restart your Go API server and open http://localhost:8000 in your browser to see the beautiful web interface!