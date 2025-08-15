#!/bin/bash

# Build All Script for Swit Framework Documentation
# This script performs a complete build of the documentation website

set -e

echo "🚀 Starting complete documentation build..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}==>${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if we're in the right directory
if [ ! -f "package.json" ]; then
    print_error "package.json not found. Please run this script from docs/pages directory"
    exit 1
fi

# Check if docs/generated directory exists
if [ ! -d "../../generated" ]; then
    print_warning "docs/generated directory not found. API documentation may be incomplete."
    print_warning "Run 'make swagger' from the root directory to generate API specs first."
fi

# Step 1: Install dependencies
print_status "Installing dependencies..."
if npm ci; then
    print_success "Dependencies installed"
else
    print_error "Failed to install dependencies"
    exit 1
fi

# Step 2: Generate API documentation
print_status "Generating API documentation..."
if node scripts/api-docs-generator.js; then
    print_success "API documentation generated"
else
    print_error "Failed to generate API documentation"
    exit 1
fi

# Step 3: Build VitePress site
print_status "Building VitePress site..."
if npm run build; then
    print_success "VitePress site built successfully"
else
    print_error "Failed to build VitePress site"
    exit 1
fi

# Step 4: Verify build output
print_status "Verifying build output..."
if [ -d "dist" ]; then
    DIST_SIZE=$(du -sh dist | cut -f1)
    print_success "Build output verified - Size: $DIST_SIZE"
    
    # Check for critical files
    critical_files=("index.html" "zh/index.html" "en/index.html")
    for file in "${critical_files[@]}"; do
        if [ -f "dist/$file" ]; then
            print_success "Found $file"
        else
            print_warning "Missing $file"
        fi
    done
else
    print_error "Build output directory not found"
    exit 1
fi

# Step 5: Check for common issues
print_status "Running post-build checks..."

# Check for broken internal links (basic check)
if grep -r "href=\"/[^\"]*\"" dist/ | grep -v "href=\"/zh/\|href=\"/en/\|href=\"/images/\|href=\"/api/\|href=\"/guide/\|href=\"/examples/\|href=\"/community/\"" > /tmp/potential_broken_links.txt; then
    if [ -s /tmp/potential_broken_links.txt ]; then
        print_warning "Potential broken internal links found:"
        head -10 /tmp/potential_broken_links.txt
        echo "Full list saved to /tmp/potential_broken_links.txt"
    fi
fi

# Check for missing images
if grep -r "src=\"/images/" dist/ | while read -r line; do
    image_path=$(echo "$line" | sed -n 's/.*src="\([^"]*\)".*/\1/p')
    if [ -n "$image_path" ] && [ ! -f "dist$image_path" ]; then
        print_warning "Missing image: $image_path"
    fi
done; then
    print_success "Image check completed"
fi

# Performance check
print_status "Checking build performance..."
HTML_FILES=$(find dist -name "*.html" | wc -l)
JS_FILES=$(find dist -name "*.js" | wc -l)
CSS_FILES=$(find dist -name "*.css" | wc -l)

print_success "Build statistics:"
echo "  HTML files: $HTML_FILES"
echo "  JavaScript files: $JS_FILES"
echo "  CSS files: $CSS_FILES"
echo "  Total size: $DIST_SIZE"

# Final success message
echo ""
print_success "Documentation build completed successfully!"
print_status "You can now:"
echo "  • Preview locally: npm run preview"
echo "  • Deploy to GitHub Pages: git push origin gh-pages"
echo "  • Serve statically from ./dist directory"
echo ""

# Optional: Create deployment info
cat > dist/build-info.json << EOF
{
  "buildTime": "$(date -Iseconds)",
  "buildNumber": "$(date +%Y%m%d-%H%M%S)",
  "gitCommit": "$(git rev-parse HEAD 2>/dev/null || echo 'unknown')",
  "gitBranch": "$(git branch --show-current 2>/dev/null || echo 'unknown')",
  "nodeVersion": "$(node --version)",
  "npmVersion": "$(npm --version)"
}
EOF

print_success "Build info saved to dist/build-info.json"

echo "🎉 All done!"