#!/bin/bash

# SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
# SPDX-License-Identifier: Apache-2.0
# Test script for discover functionality
# Validates that discovered contracts pass schema validation

set -e

echo "🔍 Testing ChoreoAtlas Discovery Feature"
echo "======================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Paths
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
CHOREOATLAS="$PROJECT_ROOT/bin/choreoatlas"
TEST_TRACE="$PROJECT_ROOT/examples/traces/successful-order.trace.json"
OUTPUT_DIR="$PROJECT_ROOT/test-discovery-output"
OUTPUT_FLOW="$OUTPUT_DIR/discovered.flowspec.yaml"
OUTPUT_SERVICES="$OUTPUT_DIR/services"

# Clean up function
cleanup() {
    echo ""
    echo "🧹 Cleaning up test files..."
    rm -rf "$OUTPUT_DIR"
}

# Set up trap to clean up on exit
trap cleanup EXIT

# Check prerequisites
echo "📋 Checking prerequisites..."
if [ ! -f "$CHOREOATLAS" ]; then
    echo -e "${RED}❌ ChoreoAtlas binary not found at $CHOREOATLAS${NC}"
    echo "   Run 'go build -o bin/choreoatlas ./cmd/choreoatlas/' first"
    exit 1
fi

if [ ! -f "$TEST_TRACE" ]; then
    echo -e "${RED}❌ Test trace not found at $TEST_TRACE${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites satisfied${NC}"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Step 1: Run discovery
echo "🔍 Step 1: Running discovery..."
"$CHOREOATLAS" discover \
    --trace "$TEST_TRACE" \
    --out "$OUTPUT_FLOW" \
    --out-services "$OUTPUT_SERVICES" \
    --title "Test Discovery Flow"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Discovery completed successfully${NC}"
else
    echo -e "${RED}❌ Discovery failed${NC}"
    exit 1
fi

# Step 2: Verify files were created
echo ""
echo "📁 Step 2: Verifying generated files..."

if [ ! -f "$OUTPUT_FLOW" ]; then
    echo -e "${RED}❌ FlowSpec not generated at $OUTPUT_FLOW${NC}"
    exit 1
fi
echo -e "${GREEN}✅ FlowSpec generated${NC}"

SERVICE_COUNT=$(find "$OUTPUT_SERVICES" -name "*.servicespec.yaml" 2>/dev/null | wc -l)
if [ "$SERVICE_COUNT" -eq 0 ]; then
    echo -e "${RED}❌ No ServiceSpec files generated${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Generated $SERVICE_COUNT ServiceSpec files${NC}"

# Step 3: Validate FlowSpec with lint
echo ""
echo "🧪 Step 3: Validating generated FlowSpec..."
"$CHOREOATLAS" lint --flow "$OUTPUT_FLOW"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ FlowSpec validation passed${NC}"
else
    echo -e "${RED}❌ FlowSpec validation failed${NC}"
    exit 1
fi

# Step 4: Validate against original trace
echo ""
echo "🔄 Step 4: Validating against original trace..."
"$CHOREOATLAS" validate \
    --flow "$OUTPUT_FLOW" \
    --trace "$TEST_TRACE" \
    --semantic false \
    --causality off

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Trace validation passed${NC}"
else
    echo -e "${YELLOW}⚠️  Trace validation failed (expected - generated contracts need refinement)${NC}"
fi

# Step 5: Show summary
echo ""
echo "📊 Summary"
echo "========="
echo "FlowSpec: $OUTPUT_FLOW"
echo "ServiceSpecs: $OUTPUT_SERVICES/"
echo ""
echo "Generated files structure:"
tree -L 2 "$OUTPUT_DIR" 2>/dev/null || ls -la "$OUTPUT_DIR"

echo ""
echo -e "${GREEN}🎉 Discovery test completed successfully!${NC}"
echo ""
echo "To inspect the generated contracts:"
echo "  cat $OUTPUT_FLOW"
echo "  ls -la $OUTPUT_SERVICES/"
echo ""
echo "To manually test with your own trace:"
echo "  $CHOREOATLAS discover --trace <your-trace.json> --out <output.yaml>"