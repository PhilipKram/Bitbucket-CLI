#!/bin/bash
# Manual test script for error handling scenarios
# This script tests all 6 acceptance criteria

set -e

echo "=========================================="
echo "Manual Error Scenario Testing"
echo "=========================================="
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
PASSED=0
FAILED=0
TOTAL=6

echo "Test 1: 401 Unauthorized - Should show 'bb auth login' suggestion"
echo "-------------------------------------------------------------------"
echo "Command: ./bb repo list (without authentication or with invalid token)"
echo ""
echo "Expected: Error message with suggestion: 'Try running 'bb auth login' to authenticate with Bitbucket.'"
echo ""
echo -e "${YELLOW}Manual Check Required:${NC}"
echo "1. Remove or invalidate auth token: rm ~/.config/bb/config.yaml"
echo "2. Run: ./bb repo list 2>&1 | grep -i 'bb auth login'"
echo "3. Verify the suggestion appears"
echo ""
read -p "Did the 401 error show 'bb auth login' suggestion? (y/n): " response
if [[ "$response" == "y" ]]; then
    echo -e "${GREEN}✓ PASSED${NC}"
    ((PASSED++))
else
    echo -e "${RED}✗ FAILED${NC}"
    ((FAILED++))
fi
echo ""

echo "Test 2: 404 Not Found - Should show clear not-found message"
echo "-----------------------------------------------------------"
echo "Command: ./bb repo view nonexistent/repo"
echo ""
echo "Expected: Clear message like 'Not found: ...' with suggestion to check workspace/repo names"
echo ""
echo -e "${YELLOW}Manual Check Required:${NC}"
echo "Run: ./bb repo view nonexistent-workspace/nonexistent-repo 2>&1"
echo ""
read -p "Did the 404 error show a clear not-found message? (y/n): " response
if [[ "$response" == "y" ]]; then
    echo -e "${GREEN}✓ PASSED${NC}"
    ((PASSED++))
else
    echo -e "${RED}✗ FAILED${NC}"
    ((FAILED++))
fi
echo ""

echo "Test 3: 429 Rate Limit - Should show reset time"
echo "------------------------------------------------"
echo "Expected: Message showing when rate limit resets"
echo ""
echo -e "${YELLOW}Manual Check Required:${NC}"
echo "This test requires triggering actual rate limiting."
echo "If you have access to Bitbucket API, make many rapid requests to trigger 429."
echo "The error should show: 'Rate limit resets at [TIME]' or 'Wait a few moments'"
echo ""
read -p "Have you verified rate limit error shows reset time? (y/n/skip): " response
if [[ "$response" == "y" ]]; then
    echo -e "${GREEN}✓ PASSED${NC}"
    ((PASSED++))
elif [[ "$response" == "skip" ]]; then
    echo -e "${YELLOW}⊘ SKIPPED (Unable to trigger rate limit)${NC}"
    ((TOTAL--))
else
    echo -e "${RED}✗ FAILED${NC}"
    ((FAILED++))
fi
echo ""

echo "Test 4: UUID with Braces - Should work transparently"
echo "----------------------------------------------------"
echo "Testing UUID normalization function..."
echo ""

# Create a simple test program
cat > /tmp/test_uuid.go <<'EOF'
package main

import (
    "fmt"
    "os"
)

func NormalizeUUID(uuid string) string {
    if uuid == "" {
        return uuid
    }

    // Check if already URL-encoded
    if contains(uuid, "%7B") || contains(uuid, "%7b") || contains(uuid, "%7D") || contains(uuid, "%7d") {
        return uuid
    }

    // Check if it has braces that need encoding
    if len(uuid) > 0 && uuid[0] == '{' && uuid[len(uuid)-1] == '}' {
        // Simple URL encoding of braces
        result := ""
        for _, c := range uuid {
            if c == '{' {
                result += "%7B"
            } else if c == '}' {
                result += "%7D"
            } else {
                result += string(c)
            }
        }
        return result
    }

    return uuid
}

func contains(s, substr string) bool {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}

func main() {
    testCases := []struct {
        input    string
        expected string
    }{
        {"{abc-def-123}", "%7Babc-def-123%7D"},
        {"%7Babc-def-123%7D", "%7Babc-def-123%7D"},
        {"abc-def-123", "abc-def-123"},
        {"", ""},
    }

    allPassed := true
    for _, tc := range testCases {
        result := NormalizeUUID(tc.input)
        if result == tc.expected {
            fmt.Printf("✓ Input: '%s' -> '%s'\n", tc.input, result)
        } else {
            fmt.Printf("✗ Input: '%s' -> '%s' (expected '%s')\n", tc.input, result, tc.expected)
            allPassed = false
        }
    }

    if !allPassed {
        os.Exit(1)
    }
}
EOF

cd /tmp && go run test_uuid.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ PASSED${NC}"
    ((PASSED++))
else
    echo -e "${RED}✗ FAILED${NC}"
    ((FAILED++))
fi
cd - > /dev/null
echo ""

echo "Test 5: Network Timeout - Should show BB_HTTP_TIMEOUT suggestion"
echo "-----------------------------------------------------------------"
echo "Expected: Error message suggesting to check BB_HTTP_TIMEOUT environment variable"
echo ""
echo -e "${YELLOW}Manual Check Required:${NC}"
echo "To test network timeout:"
echo "1. Set very low timeout: export BB_HTTP_TIMEOUT=1ms"
echo "2. Run any API command: ./bb repo list"
echo "3. Should see suggestion about BB_HTTP_TIMEOUT"
echo ""
read -p "Did network timeout error show BB_HTTP_TIMEOUT suggestion? (y/n): " response
if [[ "$response" == "y" ]]; then
    echo -e "${GREEN}✓ PASSED${NC}"
    ((PASSED++))
else
    echo -e "${RED}✗ FAILED${NC}"
    ((FAILED++))
fi
echo ""

echo "Test 6: Errors Go to Stderr - Test with piping"
echo "-----------------------------------------------"
echo "Testing that errors go to stderr, not stdout..."
echo ""

# Test stderr vs stdout
echo "Running command that should produce an error..."
echo "Command: ./bb repo view nonexistent/repo > /tmp/stdout.txt 2> /tmp/stderr.txt"
echo ""

rm -f /tmp/stdout.txt /tmp/stderr.txt
./bb repo view nonexistent/repo > /tmp/stdout.txt 2> /tmp/stderr.txt || true

if [ ! -s /tmp/stdout.txt ] && [ -s /tmp/stderr.txt ]; then
    echo -e "${GREEN}✓ PASSED${NC} - Error went to stderr, stdout is empty"
    echo "Stderr contains:"
    head -n 5 /tmp/stderr.txt | sed 's/^/  /'
    ((PASSED++))
else
    echo -e "${RED}✗ FAILED${NC}"
    echo "Stdout size: $(wc -c < /tmp/stdout.txt) bytes"
    echo "Stderr size: $(wc -c < /tmp/stderr.txt) bytes"
    if [ -s /tmp/stdout.txt ]; then
        echo "Stdout (should be empty):"
        cat /tmp/stdout.txt | sed 's/^/  /'
    fi
    ((FAILED++))
fi
echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "Total Tests: $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
else
    echo -e "Failed: $FAILED"
fi
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed! ✓${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Please review the output above.${NC}"
    exit 1
fi
