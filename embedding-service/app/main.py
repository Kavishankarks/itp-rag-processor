import os
from typing import List
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from dotenv import load_dotenv

from app.embedding_service import EmbeddingService
from app.search_service import get_search_service
from app.normalize_service import get_normalize_service
from app.conversion_service import get_conversion_service
from fastapi import UploadFile, File
import tempfile
import shutil

# Load environment variables
load_dotenv()

# Global services
embedding_service = None
search_service = None
normalize_service = None
conversion_service = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: Initialize services
    global embedding_service, search_service, normalize_service, conversion_service
    model_name = os.getenv("EMBEDDING_MODEL", "all-MiniLM-L6-v2")
    embedding_service = EmbeddingService(model_name)
    search_service = get_search_service()
    normalize_service = get_normalize_service()
    conversion_service = get_conversion_service()
    print(f"Embedding service initialized with model: {model_name}")
    print("Search service initialized")
    print("Normalize service initialized")
    print("Conversion service initialized")
    yield
    # Shutdown: cleanup if needed
    print("Shutting down services")


# Create FastAPI app
app = FastAPI(
    title=os.getenv("APP_NAME", "Document Hub Embedding Service"),
    version="1.0.0",
    lifespan=lifespan,
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# Request/Response models
class EmbeddingRequest(BaseModel):
    texts: List[str]


class EmbeddingResponse(BaseModel):
    embeddings: List[List[float]]
    model: str
    dimension: int


class ChunkRequest(BaseModel):
    text: str
    chunk_size: int = 500
    chunk_overlap: int = 50


class ChunkResponse(BaseModel):
    chunks: List[str]


class HealthResponse(BaseModel):
    status: str
    service: str
    model: str


class SearchRequest(BaseModel):
    query: str
    max_results: int = 5
    extract_content: bool = True


class SearchResponse(BaseModel):
    results: List[dict]
    result_count: int


class EnrichTopicRequest(BaseModel):
    topic_name: str
    max_results: int = 5


class EnrichTopicResponse(BaseModel):
    topic_name: str
    search_query: str
    results: List[dict]
    combined_content: str
    result_count: int


class NormalizeRequest(BaseModel):
    text: str
    clean_html_tags: bool = True


class NormalizeResponse(BaseModel):
    normalized_text: str
    metadata: dict


class NormalizeBatchRequest(BaseModel):
    texts: List[str]
    deduplicate: bool = True
    clean_html_tags: bool = True


class NormalizeBatchResponse(BaseModel):
    normalized_texts: List[str]
    original_count: int
    normalized_count: int


class ConvertResponse(BaseModel):
    markdown: str
    filename: str


# Health check endpoint
@app.get("/health", response_model=HealthResponse)
async def health_check():
    return HealthResponse(
        status="ok",
        service="embedding-service",
        model=embedding_service.model_name if embedding_service else "not loaded",
    )


# Generate embeddings endpoint
@app.post("/api/v1/embeddings", response_model=EmbeddingResponse)
async def generate_embeddings(request: EmbeddingRequest):
    if not embedding_service:
        raise HTTPException(status_code=503, detail="Embedding service not initialized")

    if not request.texts:
        raise HTTPException(status_code=400, detail="No texts provided")

    try:
        embeddings = embedding_service.encode(request.texts)
        return EmbeddingResponse(
            embeddings=embeddings,
            model=embedding_service.model_name,
            dimension=embedding_service.dimension,
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to generate embeddings: {str(e)}")


# Chunk text endpoint
@app.post("/api/v1/chunk", response_model=ChunkResponse)
async def chunk_text(request: ChunkRequest):
    if not embedding_service:
        raise HTTPException(status_code=503, detail="Embedding service not initialized")

    if not request.text:
        raise HTTPException(status_code=400, detail="No text provided")

    try:
        chunks = embedding_service.chunk_text(
            request.text,
            chunk_size=request.chunk_size,
            chunk_overlap=request.chunk_overlap,
        )
        return ChunkResponse(chunks=chunks)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to chunk text: {str(e)}")


# Search endpoints
@app.post("/api/v1/search", response_model=SearchResponse)
async def search_web(request: SearchRequest):
    if not search_service:
        raise HTTPException(status_code=503, detail="Search service not initialized")

    if not request.query:
        raise HTTPException(status_code=400, detail="No query provided")

    try:
        results = await search_service.search_and_extract(
            query=request.query,
            max_results=request.max_results,
            extract_content=request.extract_content,
        )
        return SearchResponse(results=results, result_count=len(results))
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to perform search: {str(e)}")


@app.post("/api/v1/enrich-topic", response_model=EnrichTopicResponse)
async def enrich_topic(request: EnrichTopicRequest):
    if not search_service:
        raise HTTPException(status_code=503, detail="Search service not initialized")

    if not request.topic_name:
        raise HTTPException(status_code=400, detail="No topic name provided")

    try:
        result = await search_service.enrich_topic(
            topic_name=request.topic_name,
            max_results=request.max_results,
        )
        return EnrichTopicResponse(**result)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to enrich topic: {str(e)}")


# Normalization endpoints
@app.post("/api/v1/normalize", response_model=NormalizeResponse)
async def normalize_text(request: NormalizeRequest):
    if not normalize_service:
        raise HTTPException(status_code=503, detail="Normalize service not initialized")

    if not request.text:
        raise HTTPException(status_code=400, detail="No text provided")

    try:
        normalized = normalize_service.normalize_text(
            text=request.text,
            clean_html_tags=request.clean_html_tags,
        )
        metadata = normalize_service.extract_metadata(normalized)
        return NormalizeResponse(normalized_text=normalized, metadata=metadata)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to normalize text: {str(e)}")


@app.post("/api/v1/normalize-batch", response_model=NormalizeBatchResponse)
async def normalize_batch(request: NormalizeBatchRequest):
    if not normalize_service:
        raise HTTPException(status_code=503, detail="Normalize service not initialized")

    if not request.texts:
        raise HTTPException(status_code=400, detail="No texts provided")

    try:
        normalized = normalize_service.normalize_batch(
            texts=request.texts,
            deduplicate=request.deduplicate,
            clean_html_tags=request.clean_html_tags,
        )
        return NormalizeBatchResponse(
            normalized_texts=normalized,
            original_count=len(request.texts),
            normalized_count=len(normalized),
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to normalize batch: {str(e)}")


# Conversion endpoints
@app.post("/api/v1/convert", response_model=ConvertResponse)
async def convert_document(file: UploadFile = File(...)):
    if not conversion_service:
        raise HTTPException(status_code=503, detail="Conversion service not initialized")

    # Create a temporary file to save the uploaded content
    with tempfile.NamedTemporaryFile(delete=False, suffix=os.path.splitext(file.filename)[1]) as temp_file:
        shutil.copyfileobj(file.file, temp_file)
        temp_path = temp_file.name

    try:
        markdown_content = conversion_service.convert_file(temp_path)
        return ConvertResponse(
            markdown=markdown_content,
            filename=file.filename
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to convert document: {str(e)}")
    finally:
        # Clean up the temporary file
        if os.path.exists(temp_path):
            os.remove(temp_path)


if __name__ == "__main__":
    import uvicorn

    port = int(os.getenv("APP_PORT", "8002"))
    print(f"Starting server on port {port}")
    print(f"swagger docs at http://localhost:{port}/docs")
    print(f"swagger redoc at http://localhost:{port}/redoc")
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=port,
        reload=os.getenv("DEBUG", "false").lower() == "true",
    )
