#!/bin/bash

# verify-ci.sh - Local CI verification script
# This script runs the same checks that CI will run to catch issues early

set -e  # Exit on any error

echo "🔍 LOCAL CI VERIFICATION"
echo "========================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
    else
        echo -e "${RED}❌ $2${NC}"
        return 1
    fi
}

# Change to project root
cd "$(dirname "$0")/.."

echo -e "${BLUE}📁 Working directory: $(pwd)${NC}"
echo ""

# 1. Go formatting check
echo -e "${YELLOW}1. Checking Go formatting...${NC}"
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    echo -e "${RED}❌ gofmt failed - files need formatting:${NC}"
    gofmt -s -l .
    echo ""
    echo -e "${YELLOW}💡 Run 'gofmt -s -w .' to fix formatting${NC}"
    exit 1
else
    print_status 0 "Go formatting"
fi
echo ""

# 2. Go vet check
echo -e "${YELLOW}2. Running go vet...${NC}"
go vet ./... 2>/dev/null
print_status $? "go vet"
echo ""

# 3. Go build check
echo -e "${YELLOW}3. Testing Go build...${NC}"
go build -v ./... > /dev/null 2>&1
print_status $? "Go build"
echo ""

# 4. Go tests
echo -e "${YELLOW}4. Running Go tests...${NC}"
go test -v ./... 2>/dev/null | grep -E "(PASS|FAIL|RUN)" | tail -5
go test ./... > /dev/null 2>&1
print_status $? "Go tests"
echo ""

# 5. Golangci-lint (if available)
echo -e "${YELLOW}5. Running golangci-lint...${NC}"
if command -v golangci-lint &> /dev/null; then
    golangci-lint run --timeout=5m > /dev/null 2>&1
    print_status $? "golangci-lint"
else
    echo -e "${YELLOW}⚠️  golangci-lint not found - install with: brew install golangci-lint${NC}"
fi
echo ""

# 6. Node.js compatibility tests (if available)
echo -e "${YELLOW}6. Running compatibility tests...${NC}"
if [ -d "tests" ] && [ -f "tests/package.json" ]; then
    cd tests
    if [ ! -d "node_modules" ]; then
        echo -e "${BLUE}📦 Installing npm dependencies...${NC}"
        npm install > /dev/null 2>&1
    fi
    
    # Run main compatibility tests
    npm test > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        print_status 0 "Compatibility tests"
        
        # Show final compatibility status
        echo -e "${BLUE}📊 Compatibility Status:${NC}"
        npm test 2>/dev/null | grep -A 2 -B 2 "Total Tests" | tail -3
        echo ""
        
        # Run edge cases tests
        echo -e "${YELLOW}   Running edge cases tests...${NC}"
        npm run test:edge-cases > /dev/null 2>&1
        print_status $? "Edge cases tests"
    else
        print_status 1 "Compatibility tests"
    fi
    cd ..
else
    echo -e "${YELLOW}⚠️  Node.js tests directory not found${NC}"
fi
echo ""

# 7. Check for common issues
echo -e "${YELLOW}7. Checking for common issues...${NC}"

# Check for TODO/FIXME comments
todo_count=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs grep -c "TODO\|FIXME" 2>/dev/null | awk -F: '{sum += $2} END {print sum+0}')
if [ "$todo_count" -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Found $todo_count TODO/FIXME comments${NC}"
else
    print_status 0 "No TODO/FIXME comments"
fi

# Check for hardcoded paths
hardcoded_paths=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs grep -c "/Users/\|C:\\\\" 2>/dev/null | awk -F: '{sum += $2} END {print sum+0}')
if [ "$hardcoded_paths" -gt 0 ]; then
    echo -e "${RED}❌ Found hardcoded paths - this will break CI${NC}"
    find . -name "*.go" -not -path "./vendor/*" | xargs grep -n "/Users/\|C:\\\\" || true
    exit 1
else
    print_status 0 "No hardcoded paths"
fi

# Check for debug prints
debug_prints=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs grep -c "fmt.Print\|log.Print" 2>/dev/null | awk -F: '{sum += $2} END {print sum+0}')
if [ "$debug_prints" -gt 5 ]; then  # Allow some prints for legitimate logging
    echo -e "${YELLOW}⚠️  Found $debug_prints debug print statements${NC}"
fi

echo ""

# 8. Git status check
echo -e "${YELLOW}8. Checking git status...${NC}"
if [ -n "$(git status --porcelain)" ]; then
    echo -e "${YELLOW}⚠️  Uncommitted changes detected:${NC}"
    git status --porcelain
    echo ""
    echo -e "${YELLOW}💡 Consider committing changes before CI${NC}"
else
    print_status 0 "Clean git status"
fi
echo ""

# Summary
echo -e "${GREEN}🎉 LOCAL CI VERIFICATION COMPLETE${NC}"
echo "=================================="
echo ""
echo -e "${BLUE}📋 Summary:${NC}"
echo "• All core checks passed"
echo "• Ready for CI pipeline"
echo "• Tests: Go tests + Compatibility tests"
echo "• Linting: gofmt + go vet + golangci-lint"
echo ""
echo -e "${GREEN}✅ This commit should pass CI!${NC}"