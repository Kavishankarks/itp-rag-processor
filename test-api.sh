#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Document Hub API Test Suite ===${NC}\n"

# 1. Health checks
echo -e "${BLUE}1. Testing Health Endpoints${NC}"
echo "Go API:"
curl -s http://localhost:8000/health | jq . || echo -e "${RED}Failed${NC}"
echo ""
echo "Python Embedding Service:"
curl -s http://localhost:8001/health | jq . || echo -e "${RED}Failed${NC}"
echo ""

# 2. Create a document
echo -e "\n${BLUE}2. Creating a new document${NC}"
DOC_RESPONSE=$(curl -s -X POST http://localhost:8000/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{
    "title": "PostgreSQL Best Practices",
    "content": "PostgreSQL is a powerful, open source object-relational database system. Here are some best practices: 1. Use indexes wisely. 2. Optimize your queries. 3. Regular maintenance with VACUUM. 4. Use connection pooling. 5. Monitor performance metrics.",
    "source_url": "https://example.com/postgres-guide",
    "doc_type": "tutorial",
    "metadata": {
      "tags": ["database", "postgresql", "optimization"],
      "author": "DB Team"
    }
  }')

echo "$DOC_RESPONSE" | jq .'

DOC_ID=$(echo "$DOC_RESPONSE" | jq -r '.id')
echo -e "${GREEN}Created document with ID: $DOC_ID${NC}"

# 3. Get the document
echo -e "\n${BLUE}3. Retrieving document by ID${NC}"
curl -s "http://localhost:8000/api/v1/documents/$DOC_ID" | jq .

# 4. List all documents
echo -e "\n${BLUE}4. Listing all documents${NC}"
curl -s "http://localhost:8000/api/v1/documents?limit=5" | jq .

# 5. Full-text search
echo -e "\n${BLUE}5. Full-text search for 'postgresql optimization'${NC}"
curl -s "http://localhost:8000/api/v1/search?q=postgresql%20optimization&type=fulltext&limit=3" | jq .

# 6. Semantic search
echo -e "\n${BLUE}6. Semantic search for 'how to improve database performance'${NC}"
curl -s "http://localhost:8000/api/v1/search?q=how%20to%20improve%20database%20performance&type=semantic&limit=3" | jq .

# 7. Hybrid search
echo -e "\n${BLUE}7. Hybrid search for 'database best practices'${NC}"
curl -s "http://localhost:8000/api/v1/search?q=database%20best%20practices&type=hybrid&limit=3" | jq .

# 8. Update document
echo -e "\n${BLUE}8. Updating document${NC}"
curl -s -X PUT "http://localhost:8000/api/v1/documents/$DOC_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "tags": ["database", "postgresql", "optimization", "updated"],
      "author": "DB Team",
      "updated_by": "Test Script"
    }
  }' | jq .

echo -e "\n${GREEN}=== All tests completed! ===${NC}"
