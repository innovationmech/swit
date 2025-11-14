#!/usr/bin/env bash
# Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
#
# Security scanning script for the swit project.
# This script runs multiple security scanners and generates comprehensive reports.

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUTPUT_DIR="${PROJECT_ROOT}/_output/security"

# Default configuration
TOOLS="${TOOLS:-gosec,govulncheck}"
FORMAT="${FORMAT:-json,html}"
TARGET="${TARGET:-./...}"
FAIL_ON_HIGH="${FAIL_ON_HIGH:-true}"
FAIL_ON_MEDIUM="${FAIL_ON_MEDIUM:-false}"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --tools)
            TOOLS="$2"
            shift 2
            ;;
        --format)
            FORMAT="$2"
            shift 2
            ;;
        --target)
            TARGET="$2"
            shift 2
            ;;
        --fail-on-high)
            FAIL_ON_HIGH="$2"
            shift 2
            ;;
        --fail-on-medium)
            FAIL_ON_MEDIUM="$2"
            shift 2
            ;;
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --help)
            cat <<EOF
Usage: $0 [OPTIONS]

Security scanning script for the swit project.

Options:
    --tools <tools>           Comma-separated list of tools to run (default: gosec,govulncheck)
                             Available: gosec, govulncheck, trivy
    --format <formats>        Comma-separated list of output formats (default: json,html)
                             Available: json, html, sarif, text
    --target <target>         Target directory or package pattern (default: ./...)
    --fail-on-high <bool>     Fail on high severity findings (default: true)
    --fail-on-medium <bool>   Fail on medium severity findings (default: false)
    --output-dir <dir>        Output directory for reports (default: _output/security)
    --help                    Show this help message

Environment Variables:
    TOOLS                     Override default tools
    FORMAT                    Override default output formats
    TARGET                    Override default target
    FAIL_ON_HIGH              Override fail on high setting
    FAIL_ON_MEDIUM            Override fail on medium setting

Examples:
    # Run all scanners with default settings
    $0

    # Run only gosec
    $0 --tools gosec

    # Generate SARIF report
    $0 --format sarif

    # Scan specific package
    $0 --target ./pkg/security/...

    # Fail on medium severity
    $0 --fail-on-medium true

EOF
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Unknown option: $1${NC}" >&2
            echo "Use --help for usage information" >&2
            exit 1
            ;;
    esac
done

# Print banner
print_banner() {
    echo -e "${BLUE}"
    echo "================================================="
    echo "           SECURITY SCANNING TOOL"
    echo "================================================="
    echo -e "${NC}"
}

# Print section header
print_section() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Print success message
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Print warning message
print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Print error message
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check tool availability
check_tool() {
    local tool=$1
    if command_exists "$tool"; then
        print_success "$tool is available"
        return 0
    else
        print_warning "$tool is not installed"
        return 1
    fi
}

# Install gosec
install_gosec() {
    print_section "Installing gosec"
    if command_exists go; then
        go install github.com/securego/gosec/v2/cmd/gosec@latest
        print_success "gosec installed successfully"
    else
        print_error "Go is not installed. Cannot install gosec."
        return 1
    fi
}

# Install govulncheck
install_govulncheck() {
    print_section "Installing govulncheck"
    if command_exists go; then
        go install golang.org/x/vuln/cmd/govulncheck@latest
        print_success "govulncheck installed successfully"
    else
        print_error "Go is not installed. Cannot install govulncheck."
        return 1
    fi
}

# Install trivy
install_trivy() {
    print_section "Installing trivy"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        if command_exists brew; then
            brew install aquasecurity/trivy/trivy
            print_success "trivy installed successfully"
        else
            print_error "Homebrew is not installed. Cannot install trivy."
            return 1
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin
        print_success "trivy installed successfully"
    else
        print_error "Unsupported OS for automatic trivy installation."
        return 1
    fi
}

# Run gosec scanner
run_gosec() {
    print_section "Running gosec"
    local output_file="${OUTPUT_DIR}/gosec-report.json"
    mkdir -p "${OUTPUT_DIR}"
    
    if gosec -fmt=json -out="${output_file}" -no-fail "${TARGET}" 2>&1; then
        print_success "gosec scan completed"
        return 0
    else
        local exit_code=$?
        if [[ $exit_code -eq 1 ]]; then
            print_warning "gosec found security issues"
            return 0
        else
            print_error "gosec scan failed with exit code $exit_code"
            return 1
        fi
    fi
}

# Run govulncheck scanner
run_govulncheck() {
    print_section "Running govulncheck"
    local output_file="${OUTPUT_DIR}/govulncheck-report.json"
    mkdir -p "${OUTPUT_DIR}"
    
    if govulncheck -json "${TARGET}" > "${output_file}" 2>&1; then
        print_success "govulncheck scan completed"
        return 0
    else
        local exit_code=$?
        if [[ $exit_code -eq 3 ]]; then
            print_warning "govulncheck found vulnerabilities"
            return 0
        else
            print_error "govulncheck scan failed with exit code $exit_code"
            return 1
        fi
    fi
}

# Run trivy scanner
run_trivy() {
    print_section "Running trivy"
    local output_file="${OUTPUT_DIR}/trivy-report.json"
    mkdir -p "${OUTPUT_DIR}"
    
    if trivy fs --format json --output "${output_file}" --scanners vuln,misconfig,secret "${PROJECT_ROOT}" 2>&1; then
        print_success "trivy scan completed"
        return 0
    else
        print_error "trivy scan failed"
        return 1
    fi
}

# Main execution
main() {
    print_banner
    
    # Change to project root
    cd "${PROJECT_ROOT}"
    
    print_section "Configuration"
    echo "Tools:          ${TOOLS}"
    echo "Formats:        ${FORMAT}"
    echo "Target:         ${TARGET}"
    echo "Output Dir:     ${OUTPUT_DIR}"
    echo "Fail on High:   ${FAIL_ON_HIGH}"
    echo "Fail on Medium: ${FAIL_ON_MEDIUM}"
    
    # Create output directory
    mkdir -p "${OUTPUT_DIR}"
    
    # Check and install tools
    print_section "Checking Tool Availability"
    
    local tools_array=(${TOOLS//,/ })
    local available_tools=()
    local failed=false
    
    for tool in "${tools_array[@]}"; do
        if check_tool "$tool"; then
            available_tools+=("$tool")
        else
            print_warning "Attempting to install $tool..."
            case "$tool" in
                gosec)
                    if install_gosec; then
                        available_tools+=("$tool")
                    fi
                    ;;
                govulncheck)
                    if install_govulncheck; then
                        available_tools+=("$tool")
                    fi
                    ;;
                trivy)
                    if install_trivy; then
                        available_tools+=("$tool")
                    fi
                    ;;
                *)
                    print_error "Unknown tool: $tool"
                    ;;
            esac
        fi
    done
    
    if [[ ${#available_tools[@]} -eq 0 ]]; then
        print_error "No scanning tools available"
        exit 1
    fi
    
    # Run scanners
    print_section "Running Security Scans"
    
    for tool in "${available_tools[@]}"; do
        case "$tool" in
            gosec)
                run_gosec || failed=true
                ;;
            govulncheck)
                run_govulncheck || failed=true
                ;;
            trivy)
                run_trivy || failed=true
                ;;
        esac
    done
    
    # Summary
    print_section "Scan Summary"
    
    local total_findings=0
    local critical_count=0
    local high_count=0
    local medium_count=0
    
    # Parse reports and count findings (simplified - in production use proper JSON parsing)
    if [[ -f "${OUTPUT_DIR}/gosec-report.json" ]]; then
        local gosec_count=$(grep -o '"Issues":\[' "${OUTPUT_DIR}/gosec-report.json" | wc -l || echo 0)
        total_findings=$((total_findings + gosec_count))
    fi
    
    echo "Total findings: ${total_findings}"
    echo "Reports saved to: ${OUTPUT_DIR}"
    
    # Determine exit status
    if [[ "$failed" = true ]]; then
        print_error "Some scans failed"
        exit 1
    fi
    
    if [[ "$FAIL_ON_HIGH" = "true" && $high_count -gt 0 ]]; then
        print_error "High severity findings detected (fail-on-high is enabled)"
        exit 1
    fi
    
    if [[ "$FAIL_ON_MEDIUM" = "true" && $medium_count -gt 0 ]]; then
        print_error "Medium severity findings detected (fail-on-medium is enabled)"
        exit 1
    fi
    
    print_success "Security scan completed successfully"
}

# Run main function
main "$@"

