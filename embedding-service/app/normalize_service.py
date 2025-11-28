"""
Text Normalization Service for cleaning and standardizing text content.
Handles HTML cleaning, deduplication, and format standardization.
"""

import re
from typing import List, Dict, Set
from difflib import SequenceMatcher
import html
import logging

logger = logging.getLogger(__name__)


class NormalizeService:
    """Service for normalizing and cleaning text content."""

    def __init__(
        self,
        similarity_threshold: float = 0.85,
        min_text_length: int = 50
    ):
        """
        Initialize the normalization service.

        Args:
            similarity_threshold: Threshold for fuzzy duplicate detection (0-1)
            min_text_length: Minimum text length to keep (in characters)
        """
        self.similarity_threshold = similarity_threshold
        self.min_text_length = min_text_length

    def clean_html(self, text: str) -> str:
        """
        Remove HTML tags and decode HTML entities.

        Args:
            text: Input text with potential HTML

        Returns:
            Cleaned text
        """
        # Decode HTML entities
        text = html.unescape(text)

        # Remove HTML tags
        text = re.sub(r'<[^>]+>', '', text)

        return text

    def clean_markdown(self, text: str) -> str:
        """
        Clean markdown formatting while preserving content.

        Args:
            text: Input text with markdown

        Returns:
            Cleaned text
        """
        # Remove markdown links but keep link text
        text = re.sub(r'\[([^\]]+)\]\([^\)]+\)', r'\1', text)

        # Remove markdown images
        text = re.sub(r'!\[([^\]]*)\]\([^\)]+\)', '', text)

        # Remove markdown headers (#)
        text = re.sub(r'^#+\s+', '', text, flags=re.MULTILINE)

        # Remove bold/italic markers
        text = re.sub(r'[*_]{1,3}([^*_]+)[*_]{1,3}', r'\1', text)

        # Remove code blocks
        text = re.sub(r'```[^`]*```', '', text, flags=re.DOTALL)
        text = re.sub(r'`([^`]+)`', r'\1', text)

        return text

    def clean_whitespace(self, text: str) -> str:
        """
        Normalize whitespace and remove excessive blank lines.

        Args:
            text: Input text

        Returns:
            Text with normalized whitespace
        """
        # Replace multiple spaces with single space
        text = re.sub(r' +', ' ', text)

        # Replace multiple newlines with maximum 2
        text = re.sub(r'\n{3,}', '\n\n', text)

        # Remove trailing/leading whitespace
        text = text.strip()

        return text

    def remove_special_chars(self, text: str) -> str:
        """
        Remove or replace special characters.

        Args:
            text: Input text

        Returns:
            Cleaned text
        """
        # Remove control characters
        text = re.sub(r'[\x00-\x08\x0b-\x0c\x0e-\x1f\x7f-\x9f]', '', text)

        # Replace common problematic characters
        replacements = {
            '\u2018': "'",  # Left single quote
            '\u2019': "'",  # Right single quote
            '\u201c': '"',  # Left double quote
            '\u201d': '"',  # Right double quote
            '\u2013': '-',  # En dash
            '\u2014': '-',  # Em dash
            '\u2026': '...',  # Ellipsis
        }

        for old, new in replacements.items():
            text = text.replace(old, new)

        return text

    def normalize_text(self, text: str, clean_html_tags: bool = True) -> str:
        """
        Apply all normalization steps to text.

        Args:
            text: Input text
            clean_html_tags: Whether to clean HTML tags

        Returns:
            Fully normalized text
        """
        if not text:
            return ""

        # Clean HTML if requested
        if clean_html_tags:
            text = self.clean_html(text)

        # Clean markdown
        text = self.clean_markdown(text)

        # Remove special characters
        text = self.remove_special_chars(text)

        # Normalize whitespace
        text = self.clean_whitespace(text)

        return text

    def calculate_similarity(self, text1: str, text2: str) -> float:
        """
        Calculate similarity between two texts using sequence matching.

        Args:
            text1: First text
            text2: Second text

        Returns:
            Similarity score between 0 and 1
        """
        # Normalize for comparison
        t1 = text1.lower().strip()
        t2 = text2.lower().strip()

        return SequenceMatcher(None, t1, t2).ratio()

    def deduplicate_texts(
        self,
        texts: List[str],
        return_indices: bool = False
    ) -> List[str] | Dict[str, any]:
        """
        Remove duplicate or highly similar texts.

        Args:
            texts: List of texts to deduplicate
            return_indices: If True, return dict with texts and removed indices

        Returns:
            Deduplicated list of texts or dict with details
        """
        if not texts:
            return [] if not return_indices else {"texts": [], "removed_indices": []}

        unique_texts = []
        removed_indices = []
        seen_signatures: Set[str] = set()

        for idx, text in enumerate(texts):
            # Skip very short texts
            if len(text) < self.min_text_length:
                removed_indices.append(idx)
                continue

            # Create a signature for exact duplicate detection
            signature = text.lower().strip()[:200]  # First 200 chars

            # Check exact duplicates
            if signature in seen_signatures:
                removed_indices.append(idx)
                continue

            # Check fuzzy duplicates against existing unique texts
            is_duplicate = False
            for existing in unique_texts:
                similarity = self.calculate_similarity(text, existing)
                if similarity >= self.similarity_threshold:
                    is_duplicate = True
                    removed_indices.append(idx)
                    logger.debug(
                        f"Removing duplicate (similarity: {similarity:.2f}): "
                        f"{text[:50]}..."
                    )
                    break

            if not is_duplicate:
                unique_texts.append(text)
                seen_signatures.add(signature)

        logger.info(
            f"Deduplicated {len(texts)} texts to {len(unique_texts)} "
            f"(removed {len(removed_indices)})"
        )

        if return_indices:
            return {
                "texts": unique_texts,
                "removed_indices": removed_indices,
                "original_count": len(texts),
                "deduplicated_count": len(unique_texts)
            }

        return unique_texts

    def normalize_batch(
        self,
        texts: List[str],
        deduplicate: bool = True,
        clean_html_tags: bool = True
    ) -> List[str]:
        """
        Normalize a batch of texts.

        Args:
            texts: List of texts to normalize
            deduplicate: Whether to remove duplicates
            clean_html_tags: Whether to clean HTML

        Returns:
            List of normalized texts
        """
        # Normalize each text
        normalized = [
            self.normalize_text(text, clean_html_tags=clean_html_tags)
            for text in texts
        ]

        # Filter out empty texts
        normalized = [t for t in normalized if t and len(t) >= self.min_text_length]

        # Deduplicate if requested
        if deduplicate:
            normalized = self.deduplicate_texts(normalized)

        return normalized

    def extract_metadata(self, text: str) -> Dict[str, any]:
        """
        Extract metadata from text.

        Args:
            text: Input text

        Returns:
            Dictionary with metadata
        """
        metadata = {
            "character_count": len(text),
            "word_count": len(text.split()),
            "line_count": len(text.split('\n')),
            "has_html": bool(re.search(r'<[^>]+>', text)),
            "has_urls": bool(re.search(r'https?://', text)),
            "has_code": bool(re.search(r'```|`[^`]+`', text)),
        }

        # Estimate reading time (average 200 words per minute)
        metadata["estimated_reading_time_minutes"] = max(
            1,
            round(metadata["word_count"] / 200)
        )

        return metadata


# Global instance
_normalize_service: NormalizeService | None = None


def get_normalize_service() -> NormalizeService:
    """Get or create the global normalization service instance."""
    global _normalize_service
    if _normalize_service is None:
        _normalize_service = NormalizeService()
    return _normalize_service
