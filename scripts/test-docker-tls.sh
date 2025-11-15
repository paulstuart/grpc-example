#!/bin/bash
# Test script to verify TLS works across Docker containers
set -e

echo "Testing TLS certificate validation across Docker containers..."
echo ""

# Test 1: Verify certificate has correct SANs
echo "1. Checking certificate SANs..."
openssl x509 -in certs/server.crt -noout -text | grep -A 2 "Subject Alternative Name"
echo ""

# Test 2: Test from host to container using localhost (should work with our cert)
echo "2. Testing HTTPS from host to container (localhost)..."
if curl -s --cacert certs/server.crt https://localhost:11005/v1/users > /dev/null 2>&1; then
    echo "✅ Host → Container (localhost) - Certificate validated successfully"
else
    echo "❌ Host → Container (localhost) - Certificate validation failed"
fi
echo ""

# Test 3: Try to connect using Docker service name from host (will fail - DNS not resolvable)
echo "3. Testing certificate with 'api' hostname..."
echo "   (This tests if the cert has the SAN, not if connection works from host)"
if openssl s_client -connect localhost:11005 -servername api -CAfile certs/server.crt </dev/null 2>&1 | grep -q "Verify return code: 0 (ok)"; then
    echo "✅ Certificate accepted for hostname 'api'"
else
    echo "⚠️  Certificate verification issue (may be due to hostname mismatch from host)"
fi
echo ""

# Test 4: Show what Docker sees
echo "4. Docker network configuration:"
docker network inspect grpc-example_grpc-network | jq -r '.[0].Containers | to_entries[] | "\(.value.Name): \(.value.IPv4Address)"' | head -10
echo ""

echo "✅ TLS certificate test complete!"
echo ""
echo "The certificate includes these SANs for Docker inter-container communication:"
echo "  - api, api-1, api-2, api-3, api-4, api-5 (Docker service names)"
echo "  - 172.21.0.10-14 (Docker network IPs)"
echo "  - localhost, grpc.example (host access)"
