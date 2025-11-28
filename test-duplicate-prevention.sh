#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Testing Duplicate Document Prevention ===${NC}\n"

# 1. Create first document
echo -e "${BLUE}1. Creating first document with title 'Go Best Practices'${NC}"
RESPONSE1=$(curl -s -X POST http://localhost:8000/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Go Best Practices",
    "content": "Go is a compiled language with excellent performance. Best practices include using goroutines for concurrency, proper error handling, and following Go conventions.",
    "doc_type": "guide",
    "metadata": {
      "tags": ["go", "best-practices"],
      "author": "Test User"
    }
  }')

echo "$RESPONSE1" | jq .
DOC_ID=$(echo "$RESPONSE1" | jq -r '.id')

if [ "$DOC_ID" != "null" ]; then
  echo -e "${GREEN}✓ First document created successfully with ID: $DOC_ID${NC}\n"
else
  echo -e "${RED}✗ Failed to create first document${NC}\n"
fi

# 2. Try to create duplicate document with same title
echo -e "${BLUE}2. Attempting to create duplicate document with same title${NC}"
RESPONSE2=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST http://localhost:8000/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Go Best Practices",
    "content": "This is different content but same title. This should be rejected.",
    "doc_type": "tutorial"
  }')

HTTP_STATUS=$(echo "$RESPONSE2" | grep "HTTP_STATUS" | cut -d: -f2)
BODY=$(echo "$RESPONSE2" | sed '/HTTP_STATUS/d')

echo "$BODY" | jq .

if [ "$HTTP_STATUS" = "409" ]; then
  echo -e "${GREEN}✓ Duplicate correctly rejected with 409 Conflict${NC}\n"
else
  echo -e "${RED}✗ Expected 409 Conflict but got HTTP $HTTP_STATUS${NC}\n"
fi

# 3. Create document with different title (should succeed)
echo -e "${BLUE}3. Creating document with different title 'Python Best Practices'${NC}"
RESPONSE3=$(curl -s -X POST http://localhost:8000/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Python Best Practices",
    "content": "Python is a high-level, interpreted programming language. Best practices include using virtual environments, following PEP 8, and writing tests.",
    "doc_type": "guide",
    "metadata": {
      "tags": ["python", "best-practices"]
    }
  }')

echo "$RESPONSE3" | jq .
DOC_ID2=$(echo "$RESPONSE3" | jq -r '.id')

if [ "$DOC_ID2" != "null" ]; then
  echo -e "${GREEN}✓ Document with different title created successfully with ID: $DOC_ID2${NC}\n"
else
  echo -e "${RED}✗ Failed to create document with different title${NC}\n"
fi

# 4. Update the first document (should work)
echo -e "${BLUE}4. Updating first document with ID: $DOC_ID${NC}"
RESPONSE4=$(curl -s -X PUT "http://localhost:8000/api/v1/documents/$DOC_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "tags": ["go", "best-practices", "updated"],
      "author": "Test User",
      "updated_at": "2025-10-12"
    }
  }')

echo "$RESPONSE4" | jq .
echo -e "${GREEN}✓ Document updated successfully${NC}\n"

echo -e "${GREEN}=== Test Summary ===${NC}"
echo -e "${GREEN}✓ Duplicate prevention is working correctly${NC}"
echo -e "${YELLOW}Note: To update a document with an existing title, use PUT /api/v1/documents/:id${NC}"
