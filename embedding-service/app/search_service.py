"""
Web Search Service for enriching curriculum topics with additional context.
Supports DuckDuckGo search with web scraping capabilities.
"""

import asyncio
import re
from typing import List, Dict, Optional
from urllib.parse import quote_plus
import httpx
from bs4 import BeautifulSoup
import logging

logger = logging.getLogger(__name__)


class SearchService:
    """Service for performing web searches and extracting content."""

    def __init__(self, timeout: int = 30):
        """
        Initialize the search service.

        Args:
            timeout: HTTP request timeout in seconds
        """
        self.timeout = timeout
        self.user_agent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

    async def search_duckduckgo(
        self,
        query: str,
        max_results: int = 5
    ) -> List[Dict[str, str]]:
        """
        Search DuckDuckGo and return results.

        Args:
            query: Search query
            max_results: Maximum number of results to return

        Returns:
            List of search results with title, url, and snippet
        """
        try:
            # Use DuckDuckGo lite HTML version (easier to parse)
            encoded_query = quote_plus(query)
            url = f"https://lite.duckduckgo.com/lite/?q={encoded_query}"

            async with httpx.AsyncClient(timeout=self.timeout) as client:
                headers = {"User-Agent": self.user_agent}
                response = await client.get(url, headers=headers)
                response.raise_for_status()

                soup = BeautifulSoup(response.text, 'html.parser')
                results = []

                # Parse search results from DuckDuckGo lite
                result_tables = soup.find_all('tr')

                for table_row in result_tables[:max_results * 2]:  # Get more rows to filter
                    link = table_row.find('a', class_='result-link')
                    snippet_td = table_row.find('td', class_='result-snippet')

                    if link and snippet_td:
                        title = link.get_text(strip=True)
                        href = link.get('href', '')
                        snippet = snippet_td.get_text(strip=True)

                        if href and not href.startswith('/'):
                            results.append({
                                "title": title,
                                "url": href,
                                "snippet": snippet
                            })

                            if len(results) >= max_results:
                                break

                logger.info(f"Found {len(results)} results for query: {query}")
                return results

        except Exception as e:
            logger.error(f"Error searching DuckDuckGo: {e}")
            return []

    async def extract_content_from_url(
        self,
        url: str,
        max_length: int = 5000
    ) -> Optional[str]:
        """
        Extract text content from a URL.

        Args:
            url: URL to fetch and extract content from
            max_length: Maximum content length to return

        Returns:
            Extracted text content or None if failed
        """
        try:
            async with httpx.AsyncClient(
                timeout=self.timeout,
                follow_redirects=True
            ) as client:
                headers = {"User-Agent": self.user_agent}
                response = await client.get(url, headers=headers)
                response.raise_for_status()

                # Only process HTML content
                content_type = response.headers.get('content-type', '')
                if 'text/html' not in content_type.lower():
                    logger.warning(f"Skipping non-HTML content: {content_type}")
                    return None

                soup = BeautifulSoup(response.text, 'html.parser')

                # Remove script and style elements
                for script in soup(['script', 'style', 'header', 'footer', 'nav']):
                    script.decompose()

                # Get text content
                text = soup.get_text(separator=' ', strip=True)

                # Clean up whitespace
                text = re.sub(r'\s+', ' ', text)

                # Truncate if too long
                if len(text) > max_length:
                    text = text[:max_length] + "..."

                logger.info(f"Extracted {len(text)} characters from {url}")
                return text

        except Exception as e:
            logger.error(f"Error extracting content from {url}: {e}")
            return None

    async def search_and_extract(
        self,
        query: str,
        max_results: int = 5,
        extract_content: bool = True
    ) -> List[Dict[str, any]]:
        """
        Search and optionally extract content from results.

        Args:
            query: Search query
            max_results: Maximum number of results
            extract_content: Whether to extract full content from URLs

        Returns:
            List of enriched search results
        """
        # Perform search
        results = await self.search_duckduckgo(query, max_results)

        if not extract_content or not results:
            return results

        # Extract content from URLs in parallel
        async def fetch_and_add_content(result):
            content = await self.extract_content_from_url(result['url'])
            result['extracted_content'] = content
            return result

        tasks = [fetch_and_add_content(result) for result in results]
        enriched_results = await asyncio.gather(*tasks, return_exceptions=True)

        # Filter out failed extractions (exceptions)
        enriched_results = [
            r for r in enriched_results
            if not isinstance(r, Exception)
        ]

        return enriched_results

    async def enrich_topic(
        self,
        topic_name: str,
        max_results: int = 5
    ) -> Dict[str, any]:
        """
        Enrich a curriculum topic with web search results.

        Args:
            topic_name: Name of the topic to enrich
            max_results: Maximum number of search results

        Returns:
            Dictionary with search results and combined content
        """
        # Create a better search query
        search_query = f"{topic_name} tutorial explanation guide"

        results = await self.search_and_extract(
            search_query,
            max_results=max_results,
            extract_content=True
        )

        # Combine all extracted content
        combined_content = f"Topic: {topic_name}\n\n"

        for idx, result in enumerate(results, 1):
            combined_content += f"\n[Source {idx}: {result['title']}]\n"
            combined_content += f"URL: {result['url']}\n"

            if result.get('extracted_content'):
                combined_content += f"{result['extracted_content']}\n"
            else:
                combined_content += f"{result['snippet']}\n"

            combined_content += "\n" + "-" * 80 + "\n"

        return {
            "topic_name": topic_name,
            "search_query": search_query,
            "results": results,
            "combined_content": combined_content,
            "result_count": len(results)
        }


# Global instance
_search_service: Optional[SearchService] = None


def get_search_service() -> SearchService:
    """Get or create the global search service instance."""
    global _search_service
    if _search_service is None:
        _search_service = SearchService()
    return _search_service
