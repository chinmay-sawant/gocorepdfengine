#!/usr/bin/env bash
# Run veraPDF validation on a PDF and emit XML report to stdout.
# Skips gracefully when veraPDF is not installed (exit 0) so CI doesn't hard-fail.
#
# Usage:
#   ./compliance/verapdf/run_verapdf.sh -f 4 file.pdf
#   ./compliance/verapdf/run_verapdf.sh -f ua2 file.pdf
#
# Environment:
#   VERAPDF_BIN   Path to veraPDF CLI (default: verapdf)

set -euo pipefail

VERAPDF="${VERAPDF_BIN:-verapdf}"

if ! command -v "${VERAPDF}" &>/dev/null; then
    echo "SKIP: veraPDF not found at '${VERAPDF}'. Install veraPDF or set VERAPDF_BIN." >&2
    exit 0
fi

flavour=""
file=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        -f)
            flavour="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $(basename "$0") -f <4|ua2> <file>"
            exit 0
            ;;
        -*)
            echo "Unknown option: $1" >&2
            exit 2
            ;;
        *)
            file="$1"
            shift
            ;;
    esac
done

if [[ -z "${flavour}" ]]; then
    echo "ERROR: -f flavour is required (4 or ua2)" >&2
    exit 2
fi

if [[ -z "${file}" ]]; then
    echo "ERROR: no PDF file specified" >&2
    exit 2
fi

if [[ ! -f "${file}" ]]; then
    echo "ERROR: file not found: ${file}" >&2
    exit 1
fi

output=$("${VERAPDF}" -f "${flavour}" --format xml "${file}" 2>/dev/null)

echo "${output}"

if echo "${output}" | grep -q 'failedJobs="[^0]' 2>/dev/null; then
    exit 1
fi

if echo "${output}" | grep -qi 'isCompliant="false"' 2>/dev/null; then
    exit 1
fi
