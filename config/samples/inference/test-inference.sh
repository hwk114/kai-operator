#!/bin/bash

set -e

URL="${1:-}"
MODEL="${2:-Qwen/Qwen2.5-0.5B-Instruct}"

if [ -z "$URL" ]; then
    echo "Usage: $0 <external_url> [model_name]"
    echo "Example: $0 http://localhost:8080/llama-cpp/llama-cpp-minimal"
    echo ""
    echo "This script tests inference service with the given external URL."
    echo "It uses the OpenAI-compatible API format."
    exit 1
fi

echo "Testing inference service at: $URL"
echo "Using model: $MODEL"
echo ""

echo "=== Testing /v1/models endpoint ==="
curl -s -w "\nHTTP Status: %{http_code}\n" "$URL/v1/models" | head -50

echo ""
echo "=== Testing /v1/chat/completions endpoint ==="
curl -s -w "\nHTTP Status: %{http_code}\n" "$URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d "{
    \"model\": \"$MODEL\",
    \"messages\": [
      {\"role\": \"user\", \"content\": \"What is 1+1?\"}
    ],
    \"max_tokens\": 50
  }"

echo ""
echo "=== Testing /v1/completions endpoint ==="
curl -s -w "\nHTTP Status: %{http_code}\n" "$URL/v1/completions" \
  -H "Content-Type: application/json" \
  -d "{
    \"model\": \"$MODEL\",
    \"prompt\": \"What is 1+1?\",
    \"max_tokens\": 50
  }"

echo ""
echo "=== Test completed ==="
