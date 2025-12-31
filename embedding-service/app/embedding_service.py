from typing import List, Optional
import numpy as np
from sentence_transformers import SentenceTransformer
import os
from openai import OpenAI

class EmbeddingService:
    """Service for generating text embeddings using sentence-transformers"""

    def __init__(self, model_name: Optional[str] = None):
        """
        Initialize the embedding service with a specified model.

        Args:
            model_name: Name of the sentence-transformer model to use (optional)
        """
        self.model_name = model_name
        self.openai_client = None
        self.openai_api_key = os.getenv("OPENAI_API_KEY")
        self.model = None
        
        if self.openai_api_key:
            try:
                self.openai_client = OpenAI(api_key=self.openai_api_key)
                print("OpenAI client initialized")
            except Exception as e:
                print(f"Failed to initialize OpenAI client: {e}")

        # Only load local model if model_name is provided AND it's not an OpenAI model
        # (Assuming OpenAI models start with text-embedding-)
        if model_name and not model_name.startswith("text-embedding-"):
            print(f"Loading local embedding model: {model_name}...")
            self.model = SentenceTransformer(model_name)
            self.dimension = os.getenv("EMBEDDING_DIMENSION", 384)
            print(f"Model loaded successfully. Embedding dimension: {self.dimension}")
        else:
            print("No local model loaded.")
            # Default dimension if not loaded - this might be wrong if we use OpenAI
            # but usually the client asks for dimension after getting embeddings
            self.dimension = 1536 if model_name and model_name.startswith("text-embedding-3-small") else 0

    def encode(self, texts: List[str], model: Optional[str] = None) -> List[List[float]]:
        """
        Generate embeddings for a list of texts.

        Args:
            texts: List of text strings to encode
            model: Optional model name to use. If None, uses default.

        Returns:
            List of embedding vectors (each vector is a list of floats)
        """
        if not texts:
            return []

        # Check if OpenAI model is requested
        if model and model.startswith("text-embedding-3") or (model is None and self.model_name.startswith("text-embedding-3")):
            if not self.openai_client:
                raise ValueError("OpenAI client not initialized. Check OPENAI_API_KEY.")
            
            target_model = model or self.model_name
            # If default model name was not open ai but request was, we use request.
            # If default was open ai, we can use it.
            
            try:
                # OpenAI batch limit is usually around 2048 texts, but let's be safe
                # Simple implementation directly calling OpenAI
                response = self.openai_client.embeddings.create(
                    input=texts,
                    model=target_model
                )
                return [data.embedding for data in response.data]
            except Exception as e:
                print(f"OpenAI embedding failed: {e}")
                raise e

        if not self.model:
             raise ValueError("No local model loaded. Please configure EMBEDDING_MODEL or use an OpenAI model.")

        # Generate embeddings
        embeddings = self.model.encode(texts, convert_to_numpy=True)

        # Convert numpy arrays to lists
        return embeddings.tolist()

    def chunk_text(
        self, text: str, chunk_size: int = os.getenv("CHUNK_SIZE", 500), chunk_overlap: int = os.getenv("CHUNK_OVERLAP", 50)
    ) -> List[str]:
        """
        Split text into overlapping chunks using recursive splitting to preserve semantic structure.
        
        Args:
            text: Text to chunk
            chunk_size: Size of each chunk in characters
            chunk_overlap: Number of characters to overlap between chunks

        Returns:
            List of text chunks
        """
        if not text:
            return []
            
        # Default separators from largest to smallest semantic unit
        separators = ["\n\n", "\n", " ", ""]
        return self._recursive_split(text, chunk_size, chunk_overlap, separators)

    def _recursive_split(self, text: str, chunk_size: int, chunk_overlap: int, separators: List[str]) -> List[str]:
        """Recursive helper to split text by separators."""
        final_chunks = []
        
        # Get appropriate separator
        separator = separators[-1]
        new_separators = []
        for i, sep in enumerate(separators):
            if sep == "":
                separator = ""
                break
            if sep in text:
                separator = sep
                new_separators = separators[i + 1:]
                break
        
        # Split text
        if separator:
            splits = text.split(separator)
        else:
            splits = list(text) # Individual characters
            
        # Merge splits into chunks
        final_chunks = []
        current_chunk = []
        current_length = 0
        
        for split in splits:
            split_len = len(split)
            
            # If a single split is strictly larger than chunk_size, we need to recurse on it
            if split_len > chunk_size:
                # First, if we have a current_chunk accumulating, verify if we should save it
                if current_chunk:
                    final_chunks.append(separator.join(current_chunk))
                    current_chunk = []
                    current_length = 0
                
                # Recurse on this large split with the next separators
                if new_separators:
                    sub_chunks = self._recursive_split(split, chunk_size, chunk_overlap, new_separators)
                    final_chunks.extend(sub_chunks)
                else:
                    # If no more separators, we have to hard cut it (or keep it as is if it's just big)
                    # For safety, let's hard cut if it's really too big, or just append if it's the last resort
                    # Here we treat it as a chunk (might slightly exceed limit if we ran out of separators)
                    final_chunks.append(split)
                    
                continue

            # Check if adding this split would exceed chunk_size
            # We add len(separator) if it's not the first element
            sep_len = len(separator) if current_length > 0 else 0
            
            if current_length + sep_len + split_len > chunk_size:
                # Current chunk is full, save it
                if current_chunk:
                    doc = separator.join(current_chunk)
                    final_chunks.append(doc)
                    
                    # Handle overlap for the next chunk
                    # We want to keep some trailing elements for overlap
                    # This is a simple overlap strategy: keep elements that fit within overlap size
                    overlap_chunk = []
                    overlap_len = 0
                    
                    for i in range(len(current_chunk) - 1, -1, -1):
                        element = current_chunk[i]
                        element_len = len(element)
                        # Estimate added length (plus separator)
                        added_len = element_len + (len(separator) if overlap_len > 0 else 0)
                        
                        if overlap_len + added_len > chunk_overlap:
                            break
                        
                        overlap_chunk.insert(0, element)
                        overlap_len += added_len
                    
                    current_chunk = overlap_chunk
                    current_length = overlap_len
            
            # Add the current split to the accumulator
            current_chunk.append(split)
            current_length += split_len + (len(separator) if len(current_chunk) > 1 else 0)
        
        # Add any remaining chunk
        if current_chunk:
            final_chunks.append(separator.join(current_chunk))
            
        return final_chunks

    def get_dimension(self) -> int:
        """Get the dimension of the embedding vectors"""
        return self.dimension
