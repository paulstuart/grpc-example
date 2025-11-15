#!/bin/bash
# Populate the gRPC server with test user data
# Usage: ./scripts/populate-users.sh [API_URL] [TOKEN_FILE]

# Work from repo root for consistency
cd $(dirname $0)/..

set -e

# Configuration
API_URL="${1:-https://localhost:10000}"
TOKEN_FILE="${2:-testtoken}"
USERS_JSON="scripts/test-users.json"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if token file exists
if [ ! -f "$TOKEN_FILE" ]; then
    echo -e "${RED}Error: Token file '$TOKEN_FILE' not found${NC}"
    echo "Please provide a valid JWT token file"
    exit 1
fi

# Read the token
TOKEN=$(cat "$TOKEN_FILE")

echo -e "${GREEN}Populating users from $USERS_JSON...${NC}"
echo "API URL: $API_URL"
echo "Token: ${TOKEN:0:20}..."
echo ""

# Function to add a single user
add_user() {
    local user_data="$1"
    local username=$(echo "$user_data" | jq -r '.username')

    echo -e "${YELLOW}Creating user: $username${NC}"

    response=$(curl -k -s -w "\n%{http_code}" -X POST "$API_URL/v1/users" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "$user_data")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 201 ]; then
        echo -e "${GREEN}✓ Created: $username (HTTP $http_code)${NC}"
        return 0
    else
        echo -e "${RED}✗ Failed: $username (HTTP $http_code)${NC}"
        echo "Response: $body"
        return 1
    fi
}

# Read users from JSON file and create them
if [ -f "$USERS_JSON" ]; then
    echo "Reading users from $USERS_JSON"

    success_count=0
    fail_count=0

    # Read each user from the JSON array
    while IFS= read -r user; do
        if add_user "$user"; then
            ((success_count++))
        else
            ((fail_count++))
        fi
        echo ""
    done < <(jq -c '.[]' "$USERS_JSON")

    echo -e "${GREEN}================================================${NC}"
    echo -e "${GREEN}Summary:${NC}"
    echo -e "  ${GREEN}Successfully created: $success_count${NC}"
    if [ $fail_count -gt 0 ]; then
        echo -e "  ${RED}Failed: $fail_count${NC}"
    fi
    echo ""

else
    echo -e "${RED}Error: Users JSON file not found: $USERS_JSON${NC}"
    echo "Please create the file or run this script from the project root"
    exit 1
fi

# Verify by listing users
echo -e "${YELLOW}Verifying users...${NC}"
curl -k -s "$API_URL/v1/users" \
    -H "Authorization: Bearer $TOKEN" | jq '.'
exit
echo -e "${GREEN}Done!${NC}"
