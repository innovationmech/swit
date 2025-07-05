#!/bin/bash

# Pre-commit hook to run code quality checks
# This script runs the same checks as 'make ci' but only for staged files
set -e

echo "🔍 Running pre-commit checks..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    exit 1
fi

# Check if we're in a Go project directory
if [ ! -f "go.mod" ]; then
    echo "❌ No go.mod file found. Please run this script from the project root."
    exit 1
fi

# Get the list of staged Go files
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -z "$STAGED_GO_FILES" ]; then
    echo "✅ No Go files staged, skipping checks"
    exit 0
fi

echo "📝 Staged Go files:"
echo "$STAGED_GO_FILES"

# Run go mod tidy
echo "🧹 Running go mod tidy..."
go mod tidy

# Check if go.mod or go.sum changed
if ! git diff --exit-code go.mod go.sum > /dev/null 2>&1; then
    echo "⚠️  go.mod or go.sum was modified by 'go mod tidy'"
    echo "Please stage the changes and commit again"
    exit 1
fi

# Format the code
echo "🎨 Formatting code..."
gofmt -w $STAGED_GO_FILES

# Check if formatting changed anything
FORMATTED_FILES=$(git diff --name-only $STAGED_GO_FILES || true)
if [ -n "$FORMATTED_FILES" ]; then
    echo "⚠️  The following files were formatted:"
    echo "$FORMATTED_FILES"
    echo "Please stage the formatted files and commit again"
    exit 1
fi

# Run go vet
echo "🔍 Running go vet..."
go vet ./...

# Run tests for changed packages
echo "🧪 Running tests for affected packages..."

# Get unique directories of staged Go files
CHANGED_DIRS=$(echo "$STAGED_GO_FILES" | xargs -I {} dirname {} | sort -u)
NEEDS_INTERNAL_TEST=false
NEEDS_PKG_TEST=false

# Check if internal or pkg packages are affected
for dir in $CHANGED_DIRS; do
    if [[ "$dir" == internal/* ]]; then
        NEEDS_INTERNAL_TEST=true
    elif [[ "$dir" == pkg/* ]]; then
        NEEDS_PKG_TEST=true
    fi
done

# Run tests based on what's changed
if [ "$NEEDS_INTERNAL_TEST" = true ] && [ "$NEEDS_PKG_TEST" = true ]; then
    echo "Testing both internal and pkg packages..."
    go test -timeout=30s ./internal/... ./pkg/...
elif [ "$NEEDS_INTERNAL_TEST" = true ]; then
    echo "Testing internal packages..."
    go test -timeout=30s ./internal/...
elif [ "$NEEDS_PKG_TEST" = true ]; then
    echo "Testing pkg packages..."
    go test -timeout=30s ./pkg/...
else
    echo "No core packages affected, running quick test..."
    go test -timeout=10s ./...
fi

echo "✅ All pre-commit checks passed!"