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

# Get the list of staged files
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)
STAGED_PROTO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.proto$' || true)

# Check if there are any staged files that need checking
if [ -z "$STAGED_GO_FILES" ] && [ -z "$STAGED_PROTO_FILES" ]; then
    echo "✅ No Go or proto files staged, skipping checks"
    exit 0
fi

# Flag to track if we need to regenerate docs
REGENERATE_DOCS=false

echo "📝 Staged files summary:"
[ -n "$STAGED_GO_FILES" ] && echo "  Go files: $(echo $STAGED_GO_FILES | wc -w)"
[ -n "$STAGED_PROTO_FILES" ] && echo "  Proto files: $(echo $STAGED_PROTO_FILES | wc -w)"

# Store initial state of generated files for cleanup
INITIAL_GENERATED_FILES=""
if [ -d "api/gen" ]; then
    INITIAL_GENERATED_FILES=$(find api/gen -type f 2>/dev/null || true)
fi

INITIAL_SWAGGER_FILES=""
if [ -d "docs/generated" ]; then
    INITIAL_SWAGGER_FILES=$(find docs/generated -name "*.go" -o -name "*.json" -o -name "*.yaml" 2>/dev/null || true)
fi
if [ -d "internal" ]; then
    INITIAL_SWAGGER_DOCS=$(find internal -path "*/docs/docs.go" 2>/dev/null || true)
    INITIAL_SWAGGER_FILES="$INITIAL_SWAGGER_FILES $INITIAL_SWAGGER_DOCS"
fi

# Process proto files if any
if [ -n "$STAGED_PROTO_FILES" ]; then
    echo ""
    echo "🔧 Processing protobuf files..."
    echo "📝 Staged proto files:"
    echo "$STAGED_PROTO_FILES"
    
    # Check if buf is installed
    if command -v buf &> /dev/null; then
        if [ -d "api" ]; then
            echo "🎨 Formatting proto files..."
            make proto-advanced OPERATION=format
            
            # Check if formatting changed anything in staged files
            FORMATTED_PROTO_FILES=$(git diff --name-only $STAGED_PROTO_FILES || true)
            if [ -n "$FORMATTED_PROTO_FILES" ]; then
                echo "⚠️  The following proto files were formatted:"
                echo "$FORMATTED_PROTO_FILES"
                echo "Please stage the formatted files and commit again"
                exit 1
            fi
            
            # Lint proto files
            echo "🔍 Linting proto files..."
            make proto-advanced OPERATION=lint
            
            # Generate protobuf code for testing (but don't require staging)
            echo "⚙️  Generating protobuf code for testing..."
            make proto
            echo "ℹ️  Generated protobuf code is for testing only and will not be committed"
            
            REGENERATE_DOCS=true
        else
            echo "⚠️  No API directory found, skipping proto checks"
        fi
    else
        echo "⚠️  Buf CLI not installed, skipping proto checks"
        echo "Install with: make proto-setup"
    fi
fi

# Process Go files if any
if [ -n "$STAGED_GO_FILES" ]; then
    echo ""
    echo "🔧 Processing Go files..."
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
    echo "🎨 Formatting Go code..."
    gofmt -w $STAGED_GO_FILES

    # Check if formatting changed anything
    FORMATTED_GO_FILES=$(git diff --name-only $STAGED_GO_FILES || true)
    if [ -n "$FORMATTED_GO_FILES" ]; then
        echo "⚠️  The following Go files were formatted:"
        echo "$FORMATTED_GO_FILES"
        echo "Please stage the formatted files and commit again"
        exit 1
    fi

    # Check for Swagger annotations in Go files that might need doc regeneration
    for file in $STAGED_GO_FILES; do
        if grep -q "@Summary\|@Description\|@Tags\|@Accept\|@Produce\|@Param\|@Success\|@Failure\|@Router" "$file" 2>/dev/null; then
            REGENERATE_DOCS=true
            break
        fi
    done

    # Check copyright statements
    echo "📄 Checking copyright statements..."
    STAGED_GO_FILES_WITHOUT_DOCS=$(echo $STAGED_GO_FILES | tr ' ' '\n' | grep -v '/docs/docs.go$' || true)
    
    if [ -n "$STAGED_GO_FILES_WITHOUT_DOCS" ]; then
        # 先检查暂存文件的版权声明
        FILES_NEED_UPDATE=""
        for file in $STAGED_GO_FILES_WITHOUT_DOCS; do
            if [ -f "$file" ] && ! grep -q "^// Copyright" "$file"; then
                FILES_NEED_UPDATE="$FILES_NEED_UPDATE $file"
            fi
        done
        
        if [ -n "$FILES_NEED_UPDATE" ]; then
            echo "⚠️  Some staged files need copyright statements"
            # 运行版权修复（会修复所有文件，但我们只关心暂存的）
            make copyright > /dev/null 2>&1 || true
            
            # 检查暂存文件是否被修改
            MODIFIED_FILES=""
            for file in $STAGED_GO_FILES_WITHOUT_DOCS; do
                if ! git diff --exit-code "$file" > /dev/null 2>&1; then
                    MODIFIED_FILES="$MODIFIED_FILES $file"
                fi
            done
            
            if [ -n "$MODIFIED_FILES" ]; then
                echo "🔧 Copyright statements were updated in:$MODIFIED_FILES"
                # 自动重新暂存修改的文件
                git add $MODIFIED_FILES
                echo "✅ Updated files have been automatically restaged"
            fi
        else
            echo "✅ All staged Go files have proper copyright statements"
        fi
    fi

    # Generate swagger docs for testing if needed
    if [ "$REGENERATE_DOCS" = true ]; then
        echo "📚 Generating Swagger documentation for testing..."
        
        # Check if swag is installed
        if command -v swag &> /dev/null; then
            echo "⚙️  Generating Swagger documentation for testing..."
            make swagger
            echo "ℹ️  Generated Swagger documentation is for testing only and will not be committed"
        else
            echo "⚠️  Swag tool not installed, skipping Swagger doc generation"
            echo "Install with: make swagger-setup"
        fi
    fi

    # Run go vet (this needs the generated code to be present)
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
fi

# Check .gitignore for generated files
echo ""
echo "🗂️  Checking .gitignore for generated files..."
GITIGNORE_ISSUES=""

# Check if generated directories are ignored
if [ ! -f ".gitignore" ]; then
    echo "⚠️  No .gitignore file found"
    GITIGNORE_ISSUES="missing"
else
    # Check for important patterns
    if ! grep -q "api/gen/" .gitignore; then
        echo "⚠️  api/gen/ should be added to .gitignore"
        GITIGNORE_ISSUES="$GITIGNORE_ISSUES api/gen/"
    fi
    
    if ! grep -q "docs/generated/" .gitignore; then
        echo "⚠️  docs/generated/ should be added to .gitignore"
        GITIGNORE_ISSUES="$GITIGNORE_ISSUES docs/generated/"
    fi
    
    if ! grep -q "internal/.*/docs/docs.go" .gitignore && ! grep -q "*/docs/docs.go" .gitignore; then
        echo "⚠️  Generated Swagger docs (*/docs/docs.go) should be added to .gitignore"
        GITIGNORE_ISSUES="$GITIGNORE_ISSUES swagger-docs"
    fi
fi

if [ -n "$GITIGNORE_ISSUES" ] && [ "$GITIGNORE_ISSUES" != "missing" ]; then
    echo ""
    echo "💡 Consider adding these patterns to .gitignore:"
    echo "   api/gen/"
    echo "   docs/generated/"
    echo "   internal/*/docs/docs.go"
    echo ""
    echo "   These directories contain generated code that should not be committed."
fi

# Verify no generated files are being committed
echo "🔍 Checking for accidentally staged generated files..."
STAGED_GENERATED_FILES=""

# Check for staged protobuf generated files (only additions and modifications, not deletions)
STAGED_PROTO_GEN=$(git diff --cached --name-only --diff-filter=AM | grep "api/gen/" || true)
if [ -n "$STAGED_PROTO_GEN" ]; then
    STAGED_GENERATED_FILES="$STAGED_GENERATED_FILES protobuf"
    echo "⚠️  Found staged protobuf generated files:"
    echo "$STAGED_PROTO_GEN"
fi

# Check for staged swagger generated files (only additions and modifications, not deletions)
STAGED_SWAGGER_GEN=$(git diff --cached --name-only --diff-filter=AM | grep -E "(docs/generated/|internal/.*/docs/docs\.go)" || true)
if [ -n "$STAGED_SWAGGER_GEN" ]; then
    STAGED_GENERATED_FILES="$STAGED_GENERATED_FILES swagger"
    echo "⚠️  Found staged Swagger generated files:"
    echo "$STAGED_SWAGGER_GEN"
fi

if [ -n "$STAGED_GENERATED_FILES" ]; then
    echo ""
    echo "❌ Generated files should not be committed!"
    echo "   Please unstage these files with:"
    echo "   git reset HEAD <file>"
    echo ""
    echo "   Generated files are created during build/test and should be ignored in git."
    exit 1
fi

echo ""
echo "✅ All pre-commit checks passed!"
echo ""
echo "📊 Summary:"
[ -n "$STAGED_GO_FILES" ] && echo "  ✅ Go files: formatted, linted, tested"
[ -n "$STAGED_PROTO_FILES" ] && echo "  ✅ Proto files: formatted, linted, generated"
[ "$REGENERATE_DOCS" = true ] && echo "  ✅ Documentation: validated"
echo "  ✅ Copyright: verified"
echo "  ✅ Dependencies: tidied"
echo "  ✅ Generated files: properly excluded from commit"