#!/usr/bin/env bash
# Thin wrapper: run veraPDF PDF/A-4 and/or PDF/UA-2 on one or more PDFs.
#
# Usage:
#   ./compliance/run_verapdf.sh path/to/file.pdf
#   ./compliance/run_verapdf.sh --flavour 4 path/to/file.pdf
#   ./compliance/run_verapdf.sh --flavour ua2 fixtures/*.pdf
#   ./compliance/run_verapdf.sh --both compliance/fixtures/*.pdf
#
# Environment:
#   VERAPDF_BIN   Path to veraPDF CLI (default: <repo>/verapdf/verapdf)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERAPDF="${VERAPDF_BIN:-${REPO_ROOT}/verapdf/verapdf}"
REPORT="${REPO_ROOT}/compliance/verapdf_report.py"
FIXTURES="${REPO_ROOT}/compliance/fixtures"

FLAVOURS=()
PDFS=()

usage() {
    cat <<EOF
Usage: $(basename "$0") [options] [pdf...]

Options:
  --flavour F   Validation flavour (4, ua2, ...). Repeatable.
  --both        Shortcut for --flavour 4 --flavour ua2 (default if none given)
  -h, --help    Show this help

If no PDF paths are given, scans ${FIXTURES}/**/*.pdf

Examples:
  $(basename "$0") --both out.pdf
  $(basename "$0") --flavour 4 compliance/fixtures/minimal-text.pdf
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --flavour)
            FLAVOURS+=("$2")
            shift 2
            ;;
        --both)
            FLAVOURS+=("4" "ua2")
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        --)
            shift
            PDFS+=("$@")
            break
            ;;
        -*)
            echo "Unknown option: $1" >&2
            usage >&2
            exit 2
            ;;
        *)
            PDFS+=("$1")
            shift
            ;;
    esac
done

if ((${#FLAVOURS[@]} == 0)); then
    FLAVOURS=("4" "ua2")
fi

if [[ ! -x "${VERAPDF}" ]]; then
    echo "veraPDF CLI not found at ${VERAPDF}" >&2
    echo "Install with: make install-verapdf" >&2
    exit 1
fi

if ((${#PDFS[@]} == 0)); then
    if [[ ! -d "${FIXTURES}" ]]; then
        echo "No PDFs given and fixtures dir missing: ${FIXTURES}" >&2
        exit 1
    fi
    while IFS= read -r -d '' pdf; do
        PDFS+=("${pdf}")
    done < <(find "${FIXTURES}" -type f -iname '*.pdf' -print0 | sort -z)
fi

if ((${#PDFS[@]} == 0)); then
    echo "No PDF files to validate." >&2
    echo "Place fixtures under compliance/fixtures/ or pass paths on the CLI." >&2
    exit 1
fi

failures=0
for pdf in "${PDFS[@]}"; do
    if [[ ! -f "${pdf}" ]]; then
        echo "FAIL file not found: ${pdf}" >&2
        ((failures++)) || true
        continue
    fi
    echo "==> ${pdf}"
    for flavour in "${FLAVOURS[@]}"; do
        if ! python3 "${REPORT}" check \
            --verapdf "${VERAPDF}" \
            --pdf "${pdf}" \
            --flavour "${flavour}" \
            --sampledata "${FIXTURES}/"; then
            ((failures++)) || true
        fi
    done
done

echo ""
if ((failures > 0)); then
    echo "FAILED: ${failures} check(s)"
    exit 1
fi
echo "PASSED: all checks"
