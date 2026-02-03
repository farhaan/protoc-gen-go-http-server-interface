#!/usr/bin/env bash
#
# Regenerate all protobuf code after template or codegen changes.
#
# Usage:
#   ./scripts/regenerate.sh              # Full regeneration (build + generate + test)
#   ./scripts/regenerate.sh --skip-build # Skip build, just regenerate
#   ./scripts/regenerate.sh --skip-test  # Skip running tests
#   ./scripts/regenerate.sh --check      # Check if regeneration is needed (CI mode)
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# Parse arguments
SKIP_BUILD=false
SKIP_TEST=false
CHECK_MODE=false
for arg in "$@"; do
    case $arg in
        --skip-build) SKIP_BUILD=true ;;
        --skip-test) SKIP_TEST=true ;;
        --check) CHECK_MODE=true ;;
        --help|-h)
            echo "Usage: $0 [--skip-build] [--skip-test] [--check]"
            echo ""
            echo "Options:"
            echo "  --skip-build  Skip building the plugin binary"
            echo "  --skip-test   Skip running tests after regeneration"
            echo "  --check       Check if regeneration produces different output (for CI)"
            exit 0
            ;;
    esac
done

cd "$ROOT_DIR"

# Step 1: Build and install the plugin
if [[ "$SKIP_BUILD" == false ]]; then
    log_info "Building and installing protoc-gen-go-http-server-interface..."
    go install .
    log_info "Plugin installed successfully"
else
    log_info "Skipping build (--skip-build)"
fi

# Step 2: Check for buf
if ! command -v buf &> /dev/null; then
    log_error "buf is not installed. Please install it: https://buf.build/docs/installation"
    exit 1
fi

# Step 3: Regenerate examples
regenerate_example() {
    local dir="$1"
    local name="$2"

    if [[ ! -d "$dir" ]]; then
        log_warn "Directory $dir does not exist, skipping"
        return
    fi

    if [[ ! -f "$dir/buf.gen.yaml" ]]; then
        log_warn "No buf.gen.yaml in $dir, skipping"
        return
    fi

    log_info "Regenerating $name..."
    (cd "$dir" && buf generate)
    log_info "$name regenerated"
}

log_info "=== Regenerating Examples ==="
while IFS= read -r dir; do
    name="${dir#$ROOT_DIR/}"
    regenerate_example "$dir" "$name"
done < <(find "$ROOT_DIR/examples" -name "buf.gen.yaml" -exec dirname {} \;)

# Step 4: Run tests
if [[ "$SKIP_TEST" == false ]]; then
    log_info "=== Running tests ==="

    # Run main module tests
    log_info "Testing main module (httpinterface)..."
    if go test ./... -count=1; then
        log_info "Main module tests passed"
    else
        log_error "Main module tests failed"
        exit 1
    fi

    while IFS= read -r dir; do
        name="${dir#$ROOT_DIR/}"
        log_info "Testing $name..."
        if (cd "$dir" && go test ./... -count=1); then
            log_info "$name tests passed"
        else
            log_error "$name tests failed"
            exit 1
        fi
    done < <(find "$ROOT_DIR/examples" -name "go.mod" -exec dirname {} \;)

    # Run all tests in tests/ module
    if [[ -d "$ROOT_DIR/tests" ]]; then
        log_info "Testing tests/ module..."
        if (cd "$ROOT_DIR/tests" && go test ./... -count=1); then
            log_info "tests/ module tests passed"
        else
            log_error "tests/ module tests failed"
            exit 1
        fi
    fi

    log_info "All tests passed"
else
    log_info "Skipping tests (--skip-test)"
fi

# Step 5: Check mode - verify no uncommitted changes to generated files
if [[ "$CHECK_MODE" == true ]]; then
    log_info "=== Checking for uncommitted changes ==="

    CHANGED_FILES=$(git diff --name-only -- '*.pb.go' 2>/dev/null || true)
    if [[ -n "$CHANGED_FILES" ]]; then
        log_error "Generated files have changed after regeneration:"
        echo "$CHANGED_FILES"
        log_error "Please run './scripts/regenerate.sh' and commit the changes"
        exit 1
    fi

    log_info "No uncommitted changes to generated files"
fi

log_info "=== Regeneration complete ==="
