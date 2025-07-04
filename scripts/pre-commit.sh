#!/bin/bash

# Pre-commit hook to run code quality checks
set -e

echo "🔍 Running pre-commit checks..."

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
AFFECTED_PACKAGES=$(echo "$STAGED_GO_FILES" | xargs -I {} dirname {} | sort -u | xargs -I {} go list -f '{{.ImportPath}}' ./{})

if [ -n "$AFFECTED_PACKAGES" ]; then
    echo "Testing packages: $AFFECTED_PACKAGES"
    echo "$AFFECTED_PACKAGES" | xargs go test -timeout=30s
else
    echo "No packages to test"
fi

echo "✅ All pre-commit checks passed!"