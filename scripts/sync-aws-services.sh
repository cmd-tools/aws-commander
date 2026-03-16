#!/usr/bin/env bash
#
# AWS Commander Service Sync Wrapper
#
# This script wraps the Python sync script and provides a convenient CLI.
#
# Usage:
#   ./sync-aws-services.sh [options]
#
# Options:
#   --dry-run      Show what would change without making changes
#   --report       Generate a detailed report of differences (default if no other options)
#   --apply        Apply changes to configurations
#   --backup       Create backup before applying changes (used with --apply)
#   --service SVC  Only process specific service
#   --verbose      Enable verbose output
#   --help         Show this help message
#
# Environment Variables:
#   AWS_CLI_PATH          Path to AWS CLI source (default: ~/code/aws-cli)
#   AWS_COMMANDER_PATH    Path to aws-commander (default: parent of this script)
#
# Examples:
#
#   # See what would change (dry-run)
#   ./sync-aws-services.sh --dry-run
#
#   # Generate report
#   ./sync-aws-services.sh --report
#
#   # Apply all changes with backup
#   ./sync-aws-services.sh --apply --backup
#
#   # Process a single service
#   ./sync-aws-services.sh --service dynamodb --apply
#
#   # Use custom paths
#   AWS_CLI_PATH=/path/to/aws-cli AWS_COMMANDER_PATH=/path/to/aws-commander ./sync-aws-services.sh --apply
#

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_NAME="$(basename "${BASH_SOURCE[0]}")"

# Default paths
DEFAULT_AWS_CLI_PATH="${HOME}/code/aws-cli"
DEFAULT_AWS_COMMANDER_PATH="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors (if terminal supports it)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

show_help() {
    head -44 "${BASH_SOURCE[0]}" | tail -38 | sed 's/^#//'
}

# Parse arguments
DRY_RUN=false
REPORT=false
APPLY=false
BACKUP=false
SERVICE=""
VERBOSE=false
PYTHON_ARGS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            PYTHON_ARGS+=(--dry-run)
            shift
            ;;
        --report)
            REPORT=true
            PYTHON_ARGS+=(--report)
            shift
            ;;
        --apply)
            APPLY=true
            PYTHON_ARGS+=(--apply)
            shift
            ;;
        --backup)
            BACKUP=true
            PYTHON_ARGS+=(--backup)
            shift
            ;;
        --service)
            SERVICE="$2"
            PYTHON_ARGS+=(--service "$2")
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            PYTHON_ARGS+=(--verbose)
            shift
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# Set defaults for paths
AWS_CLI_PATH="${AWS_CLI_PATH:-$DEFAULT_AWS_CLI_PATH}"
AWS_COMMANDER_PATH="${AWS_COMMANDER_PATH:-$DEFAULT_AWS_COMMANDER_PATH}"

# Validate paths
if [[ ! -d "${AWS_CLI_PATH}" ]]; then
    echo -e "${RED}ERROR: AWS CLI path not found: ${AWS_CLI_PATH}${NC}"
    echo "Set AWS_CLI_PATH environment variable or use --aws-cli-path"
    exit 1
fi

if [[ ! -d "${AWS_COMMANDER_PATH}" ]]; then
    echo -e "${RED}ERROR: aws-commander path not found: ${AWS_COMMANDER_PATH}${NC}"
    echo "Set AWS_COMMANDER_PATH environment variable or use --aws-commander-path"
    exit 1
fi

if [[ ! -d "${AWS_COMMANDER_PATH}/configurations" ]]; then
    echo -e "${RED}ERROR: configurations directory not found in ${AWS_COMMANDER_PATH}${NC}"
    exit 1
fi

# Check for Python
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}ERROR: python3 not found${NC}"
    exit 1
fi

# Check for botocore
if ! python3 -c "import botocore" 2>/dev/null; then
    echo -e "${YELLOW}WARNING: botocore not installed${NC}"
    echo "Installing botocore..."
    if command -v pip3 &> /dev/null; then
        pip3 install botocore --quiet
    elif command -v pip &> /dev/null; then
        pip install botocore --quiet
    else
        echo -e "${RED}ERROR: pip not found. Install botocore manually: pip install botocore${NC}"
        exit 1
    fi
fi

# Default action is --report if nothing specified
if [[ "${DRY_RUN}" == "false" && "${REPORT}" == "false" && "${APPLY}" == "false" ]]; then
    REPORT=true
    PYTHON_ARGS+=(--report)
fi

# Warning for --apply without --backup
if [[ "${APPLY}" == "true" && "${BACKUP}" == "false" ]]; then
    echo -e "${YELLOW}WARNING: Applying changes without --backup. Use --backup to create a backup.${NC}"
    echo "Press Ctrl+C to cancel, or Enter to continue..."
    read -r
fi

# Run the sync script
echo -e "${BLUE}AWS CLI Path:${NC} ${AWS_CLI_PATH}"
echo -e "${BLUE}aws-commander Path:${NC} ${AWS_COMMANDER_PATH}"
echo ""

python3 "${SCRIPT_DIR}/sync_aws_services.py" \
    --aws-cli-path "${AWS_CLI_PATH}" \
    --aws-commander-path "${AWS_COMMANDER_PATH}" \
    "${PYTHON_ARGS[@]}"

EXIT_CODE=$?

if [[ ${EXIT_CODE} -eq 0 ]]; then
    if [[ "${APPLY}" == "true" ]]; then
        echo ""
        echo -e "${GREEN}Changes applied successfully!${NC}"
        
        # Suggest verification
        echo ""
        echo "Run these commands to verify:"
        echo "  cd ${AWS_COMMANDER_PATH}"
        echo "  go build ./..."
        echo "  go vet ./..."
    fi
else
    echo -e "${RED}Sync failed with exit code ${EXIT_CODE}${NC}"
fi

exit ${EXIT_CODE}
