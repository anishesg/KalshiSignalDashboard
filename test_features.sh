#!/bin/bash

# Test script for Kalshi Signal Dashboard
# This tests all API endpoints and verifies the system works

BASE_URL="http://localhost:8080"
API_URL="${BASE_URL}/api/v1"

echo "=== Testing Kalshi Signal Dashboard ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

test_endpoint() {
    local method=$1
    local endpoint=$2
    local description=$3
    
    echo -n "Testing $description... "
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "${API_URL}${endpoint}")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "${API_URL}${endpoint}")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        echo -e "${GREEN}✓${NC} (HTTP $http_code)"
        if [ -n "$body" ] && [ "$body" != "null" ]; then
            echo "  Response: $(echo "$body" | head -c 100)..."
        fi
        return 0
    else
        echo -e "${RED}✗${NC} (HTTP $http_code)"
        echo "  Error: $body"
        return 1
    fi
}

# Test health endpoint
test_endpoint "GET" "/health" "Health Check"

# Test categories endpoint
test_endpoint "GET" "/categories" "Categories List"

# Test markets endpoint
test_endpoint "GET" "/markets" "Markets List"

# Test signals endpoint
test_endpoint "GET" "/signals" "Signals List"

# Test alerts endpoint
test_endpoint "GET" "/alerts" "Alerts List"

# Test opportunities scanner
test_endpoint "GET" "/scanner/opportunities" "Opportunities Scanner"

# Test no-arb scanner
test_endpoint "GET" "/scanner/noarb" "No-Arb Scanner"

echo ""
echo "=== Testing Frontend Build ==="

if [ -d "dashboard/dist" ] && [ -f "dashboard/dist/index.html" ]; then
    echo -e "${GREEN}✓${NC} Frontend build exists"
    echo "  Files in dist: $(ls -1 dashboard/dist | wc -l | tr -d ' ') files"
else
    echo -e "${RED}✗${NC} Frontend build missing"
    echo "  Run: cd dashboard && npm run build"
fi

echo ""
echo "=== Summary ==="
echo "Backend should be running on: ${BASE_URL}"
echo "Frontend dev server: http://localhost:3000"
echo ""
echo "To start backend: go run main.go"
echo "To start frontend: cd dashboard && npm run dev"

