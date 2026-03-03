#!/bin/bash
# Manual verification script for clone command error handling
# Tests the three error scenarios for subtask-1-5

set -e

echo "=========================================="
echo "Clone Command Error Handling Verification"
echo "=========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}This script guides you through manual verification of clone error handling${NC}"
echo ""

echo "Test 1: Non-existent Repository Error"
echo "--------------------------------------"
echo "Command: ./bb repo clone nonexistent-workspace/nonexistent-repo"
echo ""
echo "Expected output should include:"
echo "  - Message: \"Repository 'nonexistent-workspace/nonexistent-repo' not found\""
echo "  - Suggestion: \"Check that the repository exists and you have permission to access it. Verify the workspace and repository names are correct.\""
echo ""
echo -e "${YELLOW}Run the command above and verify the error message${NC}"
read -p "Press Enter to continue..."
echo ""

echo "Test 2: Invalid Authentication Error"
echo "-------------------------------------"
echo "To test this scenario:"
echo "1. Back up your current config: cp ~/.config/bb/config.yaml ~/.config/bb/config.yaml.bak"
echo "2. Invalidate the token: echo 'access_token: invalid_token_12345' > ~/.config/bb/config.yaml"
echo "3. Run: ./bb repo clone <valid-workspace>/<valid-repo>"
echo "4. Restore config: mv ~/.config/bb/config.yaml.bak ~/.config/bb/config.yaml"
echo ""
echo "Expected output should include:"
echo "  - Message: \"Authentication failed\""
echo "  - Suggestion: \"Try running 'bb auth login' to authenticate with Bitbucket, or check that your access token is still valid.\""
echo ""
echo -e "${YELLOW}Follow the steps above and verify the error message${NC}"
read -p "Press Enter to continue..."
echo ""

echo "Test 3: Directory Already Exists Error"
echo "---------------------------------------"
echo "To test this scenario:"
echo "1. Create a test directory: mkdir test-clone-dir"
echo "2. Run: ./bb repo clone <valid-workspace>/<valid-repo> test-clone-dir"
echo "3. Clean up: rmdir test-clone-dir"
echo ""
echo "Expected output should include:"
echo "  - Message: \"Directory 'test-clone-dir' already exists\""
echo "  - Suggestion: \"Choose a different directory name or remove the existing directory before cloning.\""
echo ""
echo -e "${YELLOW}Follow the steps above and verify the error message${NC}"
read -p "Press Enter to continue..."
echo ""

echo "=========================================="
echo "Verification Complete"
echo "=========================================="
echo ""
echo "If all three error scenarios showed the expected helpful error messages,"
echo "the implementation is correct and ready to commit."
echo ""
