#!/bin/bash

set -e

echo "=========================================="
echo "  Integrated Platform Test Runner"
echo "=========================================="
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track test results
BACKEND_PASSED=0
FRONTEND_PASSED=0
OVERALL_PASSED=0

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo -e "${YELLOW}Running tests from: $SCRIPT_DIR${NC}"
echo ""

# ------------------------------------------------------------------------------
# Backend Tests (Go)
# ------------------------------------------------------------------------------
echo -e "${YELLOW}>>> Running Backend Tests (Go)${NC}"
echo "--------------------------------------"

if [ -d "backend" ] && [ -f "backend/go.mod" ]; then
    cd backend
    
    echo "Installing dependencies..."
    if go mod download 2>/dev/null; then
        echo "  Dependencies installed"
    else
        echo -e "${RED}  Warning: Could not install dependencies (may already be cached)${NC}"
    fi
    
    echo ""
    echo "Running unit tests..."
    if go test -v -race -coverprofile=coverage.out ./internal/httpserver/... 2>&1; then
        echo -e "${GREEN}Backend tests: PASSED${NC}"
        BACKEND_PASSED=1
    else
        echo -e "${RED}Backend tests: FAILED${NC}"
        BACKEND_PASSED=0
    fi
    
    echo ""
    echo "Running coverage..."
    if [ -f coverage.out ]; then
        COVERAGE=$(go tool cover -func=coverage.out | grep -E "^total:" | awk '{print $NF}')
        echo "  Coverage: $COVERAGE"
    fi
    
    cd ..
else
    echo -e "${RED}Backend directory not found, skipping${NC}"
fi

echo ""

# ------------------------------------------------------------------------------
# Frontend Tests (Angular/Karma)
# ------------------------------------------------------------------------------
echo -e "${YELLOW}>>> Running Frontend Tests (Angular)${NC}"
echo "--------------------------------------"

if [ -d "frontend" ] && [ -f "frontend/package.json" ]; then
    cd frontend
    
    echo "Installing dependencies..."
    if npm install --legacy-peer-deps 2>/dev/null; then
        echo "  Dependencies installed"
        echo ""
        echo "Running unit tests..."
        if npm test -- --no-watch --browsers=ChromeHeadless 2>&1; then
            echo -e "${GREEN}Frontend tests: PASSED${NC}"
            FRONTEND_PASSED=1
        else
            echo -e "${RED}Frontend tests: FAILED${NC}"
            FRONTEND_PASSED=0
        fi
    else
        echo -e "${RED}  Error: Could not install npm dependencies${NC}"
        echo -e "${YELLOW}  Skipping frontend tests${NC}"
        FRONTEND_PASSED=0
    fi
    
    cd ..
else
    echo -e "${RED}Frontend directory not found, skipping${NC}"
fi

echo ""

# ------------------------------------------------------------------------------
# Summary
# ------------------------------------------------------------------------------
echo "=========================================="
echo "  Test Results Summary"
echo "=========================================="
echo ""

if [ $BACKEND_PASSED -eq 1 ]; then
    echo -e "Backend:  ${GREEN}PASSED${NC}"
else
    echo -e "Backend:  ${RED}FAILED${NC}"
fi

if [ $FRONTEND_PASSED -eq 1 ]; then
    echo -e "Frontend: ${GREEN}PASSED${NC}"
else
    echo -e "Frontend: ${RED}FAILED${NC}"
fi

echo ""

if [ $BACKEND_PASSED -eq 1 ] && [ $FRONTEND_PASSED -eq 1 ]; then
    OVERALL_PASSED=1
    echo -e "${GREEN}Overall: PASSED${NC}"
    exit 0
else
    OVERALL_PASSED=0
    echo -e "${RED}Overall: FAILED${NC}"
    exit 1
fi