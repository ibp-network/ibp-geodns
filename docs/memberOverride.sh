#!/bin/bash
# This requires JQ and Curl installed on the local server

# Configuration
AUTH_KEY="your_auth_key_here"
# Membername must match member name (stored in key) here: https://github.com/ibp-network/config/blob/main/members_professional.json
DETAILS="Membername"
SERVERS=("http://dns-01.dotters.network:8080/api" "http://dns-02.dotters.network:8080/api" "http://dns-03.dotters.network:8080/api")

# Function to send the request
send_request() {
    local METHOD=$1
    local SERVER=$2

    # Create JSON payload
    JSON_PAYLOAD=$(jq -n \
                  --arg method "$METHOD" \
                  --arg details "$DETAILS" \
                  --arg authkey "$AUTH_KEY" \
                  '{method: $method, details: $details, authkey: $authkey}')

    # Send request and capture the response
    RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" -d "$JSON_PAYLOAD" "$SERVER")
    
    # Print the response
    echo "Response from $SERVER:"
    echo "$RESPONSE"
    echo
}

# Check if the argument is provided
if [ -z "$1" ]; then
    echo "Usage: $0 {enable|disable}"
    exit 1
fi

# Determine the method based on the argument
if [ "$1" == "enable" ]; then
    METHOD="enableMember"
elif [ "$1" == "disable" ]; then
    METHOD="disableMember"
else
    echo "Invalid argument: $1. Use 'enable' or 'disable'."
    exit 1
fi

# Iterate over the servers and send the request
for SERVER in "${SERVERS[@]}"; do
    send_request "$METHOD" "$SERVER"
done
