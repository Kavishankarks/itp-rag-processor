# RAG Processing Pipeline Documentation

## Overview

The RAG (Retrieval-Augmented Generation) Processing Pipeline is a powerful feature that automatically processes course curriculum documents through multiple stages to create an enriched, searchable knowledge base.

## Pipeline Stages

The pipeline consists of 6 sequential stages:

```
1. Parse      → Extract curriculum structure
2. Search     → Web search for additional context (optional)
3. Normalize  → Clean and deduplicate content
4. Chunk      → Split into manageable pieces
5. Embed      → Generate semantic embeddings
6. Store      → Save to vector database
```

### Stage Details

#### 1. Parse Stage
- Extracts course title, modules, and topics
- Supports multiple input formats:
  - **JSON**: Structured curriculum data
  - **YAML**: Configuration-style format
  - **Markdown**: Human-readable format

#### 2. Search Stage (Optional)
- Performs web searches for each topic
- Scrapes and extracts content from search results
- Uses DuckDuckGo search API
- Configurable number of results per topic (default: 5)
- Adds enriched context to original curriculum content

#### 3. Normalize Stage (Optional)
- Cleans HTML tags and special characters
- Removes duplicate content using fuzzy matching
- Standardizes text formatting
- Filters out very short or empty content

#### 4. Chunk Stage
- Splits content into overlapping chunks
- Configurable chunk size (default: 500 characters)
- Configurable overlap (default: 50 characters)
- Preserves context between chunks

#### 5. Embed Stage
- Generates semantic embeddings using sentence-transformers
- Uses `all-MiniLM-L6-v2` model (384 dimensions)
- Batch processing for efficiency

#### 6. Store Stage
- Saves documents and chunks to PostgreSQL
- Creates vector embeddings for semantic search
- Links topics back to pipeline run for tracking

---

## API Endpoints

### Start Pipeline
**POST** `/api/v1/pipeline/start`

Start a new RAG processing pipeline.

**Request Body:**
```json
{
  "curriculum": {
    "title": "Introduction to Machine Learning",
    "modules": [
      {
        "name": "Supervised Learning",
        "description": "Learn about supervised learning algorithms",
        "topics": [
          "Linear Regression",
          "Logistic Regression",
          "Decision Trees",
          "Random Forests"
        ]
      },
      {
        "name": "Unsupervised Learning",
        "description": "Explore unsupervised learning techniques",
        "topics": [
          "K-Means Clustering",
          "Hierarchical Clustering",
          "PCA",
          "t-SNE"
        ]
      }
    ]
  },
  "config": {
    "web_search_enabled": true,
    "search_results_per_topic": 5,
    "chunk_size": 500,
    "chunk_overlap": 50,
    "normalize": true,
    "search_engine": "duckduckgo"
  }
}
```

**Response:**
```json
{
  "id": 1,
  "curriculum_title": "Introduction to Machine Learning",
  "status": "pending",
  "current_stage": "parse",
  "progress": 0,
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2025-01-15T10:00:00Z"
}
```

### Get Pipeline Status
**GET** `/api/v1/pipeline/:id/status`

Check the current status and progress of a pipeline run.

**Response:**
```json
{
  "id": 1,
  "status": "processing",
  "current_stage": "embed",
  "progress": 75,
  "stages": {
    "parse": "completed",
    "search": "completed",
    "normalize": "completed",
    "chunk": "completed",
    "embed": "in_progress",
    "store": "pending"
  },
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2025-01-15T10:05:00Z"
}
```

### Get Pipeline Results
**GET** `/api/v1/pipeline/:id/results`

Retrieve the documents and chunks created by a completed pipeline.

**Response:**
```json
{
  "pipeline_run": {
    "id": 1,
    "curriculum_title": "Introduction to Machine Learning",
    "status": "completed",
    "progress": 100
  },
  "documents": [
    {
      "id": 101,
      "title": "Linear Regression",
      "content": "...",
      "doc_type": "curriculum_topic",
      "chunks": [
        {
          "id": 501,
          "chunk_text": "...",
          "chunk_index": 0
        }
      ]
    }
  ],
  "total_chunks": 42
}
```

### Cancel Pipeline
**POST** `/api/v1/pipeline/:id/cancel`

Cancel a running pipeline.

**Response:**
```json
{
  "message": "Pipeline cancelled successfully"
}
```

### List Pipelines
**GET** `/api/v1/pipelines`

List all pipeline runs with pagination.

**Query Parameters:**
- `skip` (int): Number of records to skip (default: 0)
- `limit` (int): Max records to return (default: 20, max: 100)
- `status` (string): Filter by status (pending, processing, completed, failed)

**Response:**
```json
{
  "total": 10,
  "skip": 0,
  "limit": 20,
  "results": [
    {
      "id": 1,
      "curriculum_title": "Introduction to Machine Learning",
      "status": "completed",
      "progress": 100,
      "created_at": "2025-01-15T10:00:00Z"
    }
  ]
}
```

---

## Input Formats

### JSON Format
```json
{
  "title": "Course Title",
  "modules": [
    {
      "name": "Module Name",
      "description": "Optional description",
      "topics": ["Topic 1", "Topic 2"]
    }
  ]
}
```

### YAML Format
```yaml
title: Course Title
modules:
  - name: Module Name
    description: Optional description
    topics:
      - Topic 1
      - Topic 2
```

### Markdown Format
```markdown
# Course Title

## Module: Module Name

- Topic 1
- Topic 2
- Topic 3

## Module: Another Module

- Topic A
- Topic B
```

---

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `web_search_enabled` | boolean | false | Enable web search enrichment |
| `search_results_per_topic` | int | 5 | Number of search results per topic |
| `chunk_size` | int | 500 | Size of each text chunk (characters) |
| `chunk_overlap` | int | 50 | Overlap between chunks (characters) |
| `normalize` | boolean | true | Enable text normalization |
| `search_engine` | string | "duckduckgo" | Search engine to use |

---

## Usage Examples

### Example 1: Basic Pipeline (No Web Search)

```bash
curl -X POST http://localhost:8000/api/v1/pipeline/start \
  -H "Content-Type: application/json" \
  -d '{
    "curriculum": {
      "title": "Python Basics",
      "modules": [{
        "name": "Getting Started",
        "topics": ["Variables", "Data Types", "Control Flow"]
      }]
    },
    "config": {
      "web_search_enabled": false
    }
  }'
```

### Example 2: Full Pipeline with Web Search

```bash
curl -X POST http://localhost:8000/api/v1/pipeline/start \
  -H "Content-Type: application/json" \
  -d '{
    "curriculum": {
      "title": "Advanced JavaScript",
      "modules": [{
        "name": "Async Programming",
        "topics": ["Promises", "Async/Await", "Event Loop"]
      }]
    },
    "config": {
      "web_search_enabled": true,
      "search_results_per_topic": 3,
      "normalize": true
    }
  }'
```

### Example 3: Check Pipeline Status

```bash
# Get pipeline status
curl http://localhost:8000/api/v1/pipeline/1/status

# Poll until complete
watch -n 5 'curl -s http://localhost:8000/api/v1/pipeline/1/status | jq .progress'
```

### Example 4: Search Processed Documents

Once the pipeline is complete, you can search the processed documents:

```bash
curl "http://localhost:8000/api/v1/search?q=what%20are%20promises&type=hybrid&limit=5"
```

---

## Architecture

### Component Flow

```
┌──────────────────┐
│   Go API Server  │
│   (Port 8000)    │
└─────────┬────────┘
          │
          ├─► Pipeline Orchestrator
          │   ├─► Parser (JSON/YAML/Markdown)
          │   └─► Database (PostgreSQL + pgvector)
          │
          └─► Embedding Client
              │
              ↓
┌──────────────────────────────┐
│   Python Embedding Service   │
│      (Port 8001)             │
├──────────────────────────────┤
│ • Embedding Generation       │
│ • Text Chunking              │
│ • Text Normalization         │
│ • Web Search (DuckDuckGo)    │
└──────────────────────────────┘
```

### Database Tables

**pipeline_runs**
- Tracks pipeline execution
- Stores configuration and progress
- Links to curriculum topics

**curriculum_topics**
- Individual topics from curriculum
- Original and enriched content
- Links to created documents

**documents** (existing)
- Final processed documents
- Searchable content

**document_chunks** (existing)
- Text chunks with embeddings
- Vector similarity search

---

## Error Handling

The pipeline is designed to be resilient:

1. **Web Search Failures**: If a search fails for a topic, the pipeline continues with original content
2. **Normalization Errors**: Falls back to original content if normalization fails
3. **Partial Failures**: Each topic is processed independently
4. **Status Tracking**: Full error messages stored in `error_message` field

### Error Response Example

```json
{
  "id": 1,
  "status": "failed",
  "current_stage": "search",
  "progress": 20,
  "error_message": "Failed to connect to search service",
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2025-01-15T10:02:00Z"
}
```

---

## Performance Considerations

### Pipeline Execution Time

Approximate times for a 10-topic curriculum:

| Configuration | Time |
|--------------|------|
| No web search | 10-30 seconds |
| With web search (5 results/topic) | 2-5 minutes |
| Large curriculum (50+ topics) | 5-15 minutes |

### Optimization Tips

1. **Disable web search** for faster processing if you have complete content
2. **Reduce search results** per topic (e.g., 3 instead of 5)
3. **Adjust chunk size** based on your content (larger chunks = fewer embeddings)
4. **Process in batches** for very large curriculums

---

## Monitoring

### Check Service Health

```bash
# Go API
curl http://localhost:8000/health

# Embedding Service
curl http://localhost:8001/health
```

### View Logs

```bash
# Docker logs
docker logs -f doc_hub_api
docker logs -f doc_hub_embedding_service

# Or if running locally
# Go API logs will show in terminal
# Python service logs will show in terminal
```

---

## Best Practices

1. **Start Small**: Test with a small curriculum (3-5 topics) first
2. **Monitor Progress**: Use the status endpoint to track execution
3. **Error Recovery**: Review error messages and adjust configuration
4. **Search Queries**: Pipeline results work best with hybrid search
5. **Content Quality**: Web search works best with well-defined, specific topics

---

## Troubleshooting

### Pipeline Stuck at "processing"

```bash
# Check service health
curl http://localhost:8001/health

# Check logs
docker logs doc_hub_embedding_service
```

### Web Search Not Working

- Ensure embedding service has internet access
- Check if DuckDuckGo is accessible
- Try reducing `search_results_per_topic`

### Poor Search Results

- Enable web search for richer content
- Adjust `chunk_size` (try 300-700)
- Use hybrid search type for best results

### Database Errors

```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Check connection
psql -h localhost -U postgres -d doc_hub
```

---

## Next Steps

After processing your curriculum:

1. **Search Documents**: Use `/api/v1/search` endpoint
2. **Review Results**: Check `/api/v1/pipeline/:id/results`
3. **Integrate with RAG**: Use document chunks as context for LLM prompts
4. **Build UI**: Create a frontend to visualize pipeline progress

---

## API Integration Example (JavaScript)

```javascript
// Start pipeline
async function startPipeline(curriculum, config) {
  const response = await fetch('http://localhost:8000/api/v1/pipeline/start', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ curriculum, config })
  });
  return await response.json();
}

// Poll status
async function waitForCompletion(pipelineId) {
  while (true) {
    const status = await fetch(
      `http://localhost:8000/api/v1/pipeline/${pipelineId}/status`
    ).then(r => r.json());

    console.log(`Progress: ${status.progress}% - ${status.current_stage}`);

    if (status.status === 'completed') {
      return await fetch(
        `http://localhost:8000/api/v1/pipeline/${pipelineId}/results`
      ).then(r => r.json());
    }

    if (status.status === 'failed') {
      throw new Error(status.error_message);
    }

    await new Promise(resolve => setTimeout(resolve, 2000));
  }
}

// Usage
const curriculum = {
  title: "Web Development",
  modules: [{
    name: "Frontend",
    topics: ["HTML", "CSS", "JavaScript"]
  }]
};

const run = await startPipeline(curriculum, {
  web_search_enabled: true,
  search_results_per_topic: 5
});

const results = await waitForCompletion(run.id);
console.log(`Processed ${results.total_chunks} chunks`);
```

---

## Support

For issues or questions:
- Check the logs: `docker logs doc_hub_api`
- Review error messages in pipeline status
- Check the main README.md for setup instructions
