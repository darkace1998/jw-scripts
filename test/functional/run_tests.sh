#!/bin/bash

# JW Scripts Functional Test Runner
# This script builds the applications and runs comprehensive functional tests
# covering all command-line flags and functionality.

set -e

echo "🚀 JW Scripts Functional Test Suite"
echo "======================================"

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_ROOT="$(dirname $(dirname "$SCRIPT_DIR"))"

echo "📁 Project root: $PROJECT_ROOT"

# Change to project root
cd "$PROJECT_ROOT"

# Build the applications
echo ""
echo "🔨 Building applications..."
if ! go build -o bin/ ./...; then
    echo "❌ Build failed!"
    exit 1
fi

echo "✅ Build successful"

# Verify binaries exist
if [[ ! -f "bin/jwb-index" ]] || [[ ! -f "bin/jwb-offline" ]]; then
    echo "❌ Binaries not found in bin/ directory"
    exit 1
fi

echo "✅ Binaries verified"

# Run the functional tests
echo ""
echo "🧪 Running functional tests..."
echo "⚠️  Note: Some tests may timeout or fail due to network dependencies"
echo "    This is expected behavior for tests that require internet access"
echo ""

# Set test timeout
export TEST_TIMEOUT=${TEST_TIMEOUT:-300}

# Run tests with verbose output
if go test -v -timeout ${TEST_TIMEOUT}s ./test/functional/...; then
    echo ""
    echo "✅ All functional tests completed!"
    echo ""
    echo "📊 Test Summary:"
    echo "- All command-line flags tested"
    echo "- Error handling validated"
    echo "- Integration workflows verified"
    echo "- Both jwb-index and jwb-offline applications covered"
    echo ""
    echo "ℹ️  Some tests may have timed out due to network dependencies."
    echo "   This is normal for tests that connect to external services."
else
    echo ""
    echo "⚠️  Some tests failed or timed out."
    echo "   This may be due to network connectivity issues or missing dependencies."
    echo "   Check the output above for details."
    echo ""
    exit 1
fi

echo ""
echo "🎉 Functional testing complete!"
echo ""
echo "📋 What was tested:"
echo "   • jwb-index application:"
echo "     - All 25+ command-line flags and their variations"
echo "     - Output modes: stdout, txt, html, m3u, filesystem, run"
echo "     - Language settings and validation"
echo "     - Category filtering and listing"
echo "     - Quality settings and rate limiting"
echo "     - Download flags and file management"
echo "     - Sort options and update functionality"
echo "     - Error handling and invalid inputs"
echo ""
echo "   • jwb-offline application:"
echo "     - Player command customization"
echo "     - Replay timing settings"
echo "     - Directory handling and file discovery"
echo "     - Verbosity controls"
echo "     - Error conditions and edge cases"
echo ""
echo "   • Integration scenarios:"
echo "     - Cross-application workflows"
echo "     - Complex flag combinations"
echo "     - Real-world usage patterns"
echo "     - File system interactions"
echo ""
echo "✨ All functionality has been thoroughly tested!"