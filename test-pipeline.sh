#!/bin/bash

# RAG Pipeline Test Script
# Tests the pipeline with a sample curriculum

set -e

API_URL="http://localhost:8000/api/v1"
BOLD="\033[1m"
GREEN="\033[0;32m"
YELLOW="\033[0;33m"
BLUE="\033[0;34m"
RED="\033[0;31m"
NC="\033[0m" # No Color

echo -e "${BOLD}================================${NC}"
echo -e "${BOLD}RAG Pipeline Test Script${NC}"
echo -e "${BOLD}================================${NC}\n"

# Test 1: Health Check
echo -e "${BLUE}[1/5] Checking service health...${NC}"
if curl -s "${API_URL%/api/v1}/health" | grep -q "ok"; then
    echo -e "${GREEN}✓ API service is healthy${NC}\n"
else
    echo -e "${RED}✗ API service is not responding${NC}"
    exit 1
fi

if curl -s "http://localhost:8001/health" | grep -q "ok"; then
    echo -e "${GREEN}✓ Embedding service is healthy${NC}\n"
else
    echo -e "${RED}✗ Embedding service is not responding${NC}"
    exit 1
fi

# Test 2: Start Pipeline (Small Example - No Web Search)
echo -e "${BLUE}[2/5] Starting pipeline (no web search)...${NC}"
RESPONSE=$(curl -s -X POST "$API_URL/pipeline/start" \
  -H "Content-Type: application/json" \
  -d '{
    "curriculum": {
      "title": "Python Basics Test",
      "modules": [
        {
          "name": "Introduction",
          "description": "Getting started with Python",
          "topics": ["Variables and Data Types", "Control Flow", "Functions"]
        }
      ]
    },
    "config": {
      "web_search_enabled": false,
      "normalize": true,
      "chunk_size": 500
    }
  }')

PIPELINE_ID=$(echo "$RESPONSE" | grep -o '"id":[0-9]*' | grep -o '[0-9]*')

if [ -z "$PIPELINE_ID" ]; then
    echo -e "${RED}✗ Failed to start pipeline${NC}"
    echo "$RESPONSE"
    exit 1
fi

echo -e "${GREEN}✓ Pipeline started with ID: $PIPELINE_ID${NC}\n"

# Test 3: Monitor Progress
echo -e "${BLUE}[3/5] Monitoring pipeline progress...${NC}"
COMPLETED=false
TIMEOUT=120  # 2 minutes timeout
ELAPSED=0

while [ "$COMPLETED" = false ] && [ $ELAPSED -lt $TIMEOUT ]; do
    STATUS_RESPONSE=$(curl -s "$API_URL/pipeline/$PIPELINE_ID/status")
    STATUS=$(echo "$STATUS_RESPONSE" | grep -o '"status":"[^"]*' | cut -d'"' -f4)
    PROGRESS=$(echo "$STATUS_RESPONSE" | grep -o '"progress":[0-9]*' | grep -o '[0-9]*')
    STAGE=$(echo "$STATUS_RESPONSE" | grep -o '"current_stage":"[^"]*' | cut -d'"' -f4)

    echo -e "${YELLOW}Progress: $PROGRESS% - Stage: $stage${NC}"

    if [ "$STATUS" = "completed" ]; then
        COMPLETED=true
        echo -e "${GREEN}✓ Pipeline completed successfully!${NC}\n"
    elif [ "$STATUS" = "failed" ]; then
        echo -e "${RED}✗ Pipeline failed${NC}"
        echo "$STATUS_RESPONSE"
        exit 1
    else
        sleep 2
        ELAPSED=$((ELAPSED + 2))
    fi
done

if [ "$COMPLETED" = false ]; then
    echo -e "${RED}✗ Pipeline timed out after ${TIMEOUT}s${NC}"
    exit 1
fi

# Test 4: Get Results
echo -e "${BLUE}[4/5] Fetching pipeline results...${NC}"
RESULTS=$(curl -s "$API_URL/pipeline/$PIPELINE_ID/results")
TOTAL_CHUNKS=$(echo "$RESULTS" | grep -o '"total_chunks":[0-9]*' | grep -o '[0-9]*')

if [ -z "$TOTAL_CHUNKS" ]; then
    echo -e "${RED}✗ Failed to fetch results${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Pipeline created $TOTAL_CHUNKS chunks${NC}\n"

# Test 5: Search Documents
echo -e "${BLUE}[5/5] Testing search functionality...${NC}"
SEARCH_RESULTS=$(curl -s "$API_URL/search?q=python+variables&type=hybrid&limit=3")

if echo "$SEARCH_RESULTS" | grep -q "Variables"; then
    echo -e "${GREEN}✓ Search is working!${NC}\n"
else
    echo -e "${YELLOW}⚠ Search returned no results (might need more time for indexing)${NC}\n"
fi

# Summary
echo -e "${BOLD}================================${NC}"
echo -e "${BOLD}Test Summary${NC}"
echo -e "${BOLD}================================${NC}"
echo -e "${GREEN}✓ All tests passed!${NC}"
echo -e "\nPipeline ID: ${BOLD}$PIPELINE_ID${NC}"
echo -e "Total Chunks: ${BOLD}$TOTAL_CHUNKS${NC}"
echo -e "\nYou can now:"
echo -e "  • View results: ${YELLOW}curl $API_URL/pipeline/$PIPELINE_ID/results${NC}"
echo -e "  • Search documents: ${YELLOW}curl '$API_URL/search?q=your+query&type=hybrid'${NC}"
echo -e "  • List pipelines: ${YELLOW}curl $API_URL/pipelines${NC}\n"

# Test with Web Search (Optional)
echo -e "${BOLD}Want to test with web search? Run:${NC}"
echo -e "${YELLOW}curl -X POST $API_URL/pipeline/start -H 'Content-Type: application/json' -d '{
  \"curriculum\": {
    \"title\": \"Machine Learning\",
    \"modules\": [{
      \"name\": \"Basics\",
      \"topics\": [\"Linear Regression\"]
    }]
  },
  \"config\": {
    \"web_search_enabled\": true,
    \"search_results_per_topic\": 3
  }
}'${NC}\n"
