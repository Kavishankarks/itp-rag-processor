from typing import List
import numpy as np
from sentence_transformers import SentenceTransformer
import os

class EmbeddingService:
    """Service for generating text embeddings using sentence-transformers"""

    def __init__(self, model_name: str = "all-MiniLM-L6-v2"):
        """
        Initialize the embedding service with a specified model.

        Args:
            model_name: Name of the sentence-transformer model to use
        """
        self.model_name = model_name
        print(f"Loading embedding model: {model_name}...")
        self.model = SentenceTransformer(model_name)
        self.dimension = os.getenv("EMBEDDING_DIMENSION", 384)
        print(f"Model loaded successfully. Embedding dimension: {self.dimension}")

    def encode(self, texts: List[str]) -> List[List[float]]:
        """
        Generate embeddings for a list of texts.

        Args:
            texts: List of text strings to encode

        Returns:
            List of embedding vectors (each vector is a list of floats)
        """
        if not texts:
            return []

        # Generate embeddings
        embeddings = self.model.encode(texts, convert_to_numpy=True)

        # Convert numpy arrays to lists
        return embeddings.tolist()

    def chunk_text(
        self, text: str, chunk_size: int = os.getenv("CHUNK_SIZE", 500), chunk_overlap: int = os.getenv("CHUNK_OVERLAP", 50)
    ) -> List[str]:
        """
        Split text into overlapping chunks.

        Args:
            text: Text to chunk
            chunk_size: Size of each chunk in characters
            chunk_overlap: Number of characters to overlap between chunks

        Returns:
            List of text chunks
        """
        if not text:
            return []

        chunks = []
        start = 0
        text_length = len(text)

        while start < text_length:
            # Find the end position for this chunk
            end = start + chunk_size

            # If this is not the last chunk, try to break at a sentence boundary
            if end < text_length:
                # Look for sentence endings within the next 100 characters
                search_start = end
                search_end = min(end + 100, text_length)
                sentence_endings = [
                    i
                    for i, char in enumerate(text[search_start:search_end], start=search_start)
                    if char in ".!?\n"
                ]

                if sentence_endings:
                    end = sentence_endings[0] + 1

            # Extract the chunk
            chunk = text[start:end].strip()
            if chunk:
                chunks.append(chunk)

            # Move to the next chunk with overlap
            start = end - chunk_overlap if end < text_length else text_length

        return chunks

    def get_dimension(self) -> int:
        """Get the dimension of the embedding vectors"""
        return self.dimension
