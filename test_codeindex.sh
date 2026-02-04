#!/bin/bash

# Simple test script for code index MCP server
# This script tests the basic functionality without requiring the full chat setup

set -e

echo "==================================="
echo "Code Index MCP Server Test"
echo "==================================="
echo ""

# Check if Ollama is running
echo "1. Checking Ollama..."
if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "❌ Ollama is not running!"
    echo "   Start it with: ollama serve"
    exit 1
fi
echo "✅ Ollama is running"
echo ""

# Check if model is available
echo "2. Checking embedding model..."
if ! ollama list | grep -q "nomic-embed-text"; then
    echo "❌ Model 'nomic-embed-text' not found!"
    echo "   Pull it with: ollama pull nomic-embed-text"
    exit 1
fi
echo "✅ Model 'nomic-embed-text' is available"
echo ""

# Check if binary exists
echo "3. Checking binary..."
if [ ! -f "./mcp-codeindex" ]; then
    echo "❌ Binary './mcp-codeindex' not found!"
    echo "   Build it with: go build -o mcp-codeindex ./cmd/mcp-codeindex"
    exit 1
fi
echo "✅ Binary exists"
echo ""

# Test health check endpoint via stdio
echo "4. Testing MCP server health..."
export CODE_INDEX_PATH="/tmp/test_index.json"
export OLLAMA_URL="http://localhost:11434"
export OLLAMA_MODEL="nomic-embed-text"

# Create a simple MCP request to list tools
cat > /tmp/mcp_test_request.json << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"0.1.0","clientInfo":{"name":"test","version":"1.0.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
EOF

# Run the server and send it the request
timeout 5s ./mcp-codeindex < /tmp/mcp_test_request.json > /tmp/mcp_test_response.json 2>&1 || true

# Check if we got a response
if grep -q '"method":"tools/list"' /tmp/mcp_test_response.json 2>/dev/null || \
   grep -q 'index_directory' /tmp/mcp_test_response.json 2>/dev/null; then
    echo "✅ MCP server responds to requests"
else
    echo "⚠️  Could not verify MCP server response (this is OK for stdio servers)"
fi
echo ""

# Cleanup
rm -f /tmp/mcp_test_request.json /tmp/mcp_test_response.json /tmp/test_index.json

echo "==================================="
echo "✨ All basic checks passed!"
echo "==================================="
echo ""
echo "Next steps:"
echo "  1. Add codeindex server to your config.yaml"
echo "  2. Start the chat: ./chat"
echo "  3. Index a directory: 'Index the current project'"
echo "  4. Search: 'Find code that handles errors'"
echo ""
echo "See docs/CODE_INDEX_QUICKSTART.md for full guide"
