#!/usr/bin/env bash
# PDF validation for gocorepdfengine:
#   veraPDF (PDF/A-4, PDF/UA-2), structure-tree consistency, optional avalpdf.
#
# Usage:
#   ./compliance/verify_pdfs.sh                      # all PDFs under compliance/fixtures/
#   ./compliance/verify_pdfs.sh --scan-all             # parse-only validity table
#   ./compliance/verify_pdfs.sh --scan-all-compliance  # validity + PDF/A-4 + PDF/UA-2 table
#   ./compliance/verify_pdfs.sh --pdf path/to/file.pdf # single file (or multiple --pdf)
#   ./compliance/verify_pdfs.sh path1.pdf path2.pdf    # positional PDF paths
#
# Environment:
#   VERIFY_PDFS_JOBS       Max parallel workers (default: nproc or 4)
#   VERAPDF_BIN            Path to veraPDF CLI (default: <repo>/verapdf/verapdf)
#   AVALPDF_BIN            Path to avalpdf CLI
#   VERIFY_STRUCTURE_TREE  Run structure_tree_check.py (default: 1)
#   VERIFY_AVALPDF         Run avalpdf on compliance PDFs (default: 1)
#   VERIFY_AVALPDF_STRICT  Fail on avalpdf issues (default: 0)
#   COMPLIANCE_FLAVOURS    Comma-separated flavours (default: 4,ua2)
#   NO_COLOR               Disable ANSI colours

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FIXTURES="${COMPLIANCE_FIXTURES:-${REPO_ROOT}/compliance/fixtures}"
VERAPDF="${VERAPDF_BIN:-${REPO_ROOT}/verapdf/verapdf}"
VERAPDF_REPORT="${REPO_ROOT}/compliance/verapdf_report.py"
STRUCTURE_TREE_CHECK="${REPO_ROOT}/compliance/structure_tree_check.py"
AVALPDF="${AVALPDF_BIN:-${REPO_ROOT}/.pdf-validators/venv/bin/avalpdf}"
VERIFY_STRUCTURE_TREE="${VERIFY_STRUCTURE_TREE:-1}"
VERIFY_AVALPDF="${VERIFY_AVALPDF:-1}"
VERIFY_AVALPDF_STRICT="${VERIFY_AVALPDF_STRICT:-0}"
COMPLIANCE_FLAVOURS="${COMPLIANCE_FLAVOURS:-4,ua2}"
PARALLEL_JOBS="${VERIFY_PDFS_JOBS:-$(
    if command -v nproc >/dev/null 2>&1; then
        nproc
    else
        echo 4
    fi
)}"

MODE="verify"
EXPLICIT_PDFS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --scan-all)
            MODE="scan-all"
            shift
            ;;
        --scan-all-compliance)
            MODE="scan-all-compliance"
            shift
            ;;
        --pdf)
            EXPLICIT_PDFS+=("$2")
            shift 2
            ;;
        -h|--help)
            sed -n '2,22p' "$0"
            exit 0
            ;;
        --)
            shift
            EXPLICIT_PDFS+=("$@")
            break
            ;;
        -*)
            echo "Unknown option: $1" >&2
            exit 2
            ;;
        *)
            EXPLICIT_PDFS+=("$1")
            shift
            ;;
    esac
done

# Soft-skip when there is nothing to validate yet (engine not producing fixtures).
early_pdfs=()
if ((${#EXPLICIT_PDFS[@]} > 0)); then
    early_pdfs=("${EXPLICIT_PDFS[@]}")
elif [[ -d "${FIXTURES}" ]]; then
    while IFS= read -r -d '' pdf; do
        early_pdfs+=("${pdf}")
    done < <(find "${FIXTURES}" -type f -iname '*.pdf' -print0 2>/dev/null | sort -z)
fi
if ((${#early_pdfs[@]} == 0)); then
    echo "SKIP: no PDF files to validate under ${FIXTURES}"
    echo "Add fixtures or pass paths; install veraPDF with: make install-verapdf"
    exit 0
fi

if [[ ! -x "${VERAPDF}" ]]; then
    echo "veraPDF CLI not found at ${VERAPDF}" >&2
    echo "Install with: make install-verapdf" >&2
    exit 1
fi

if [[ ! -f "${VERAPDF_REPORT}" ]]; then
    echo "veraPDF report helper not found at ${VERAPDF_REPORT}" >&2
    exit 1
fi

if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
    COLOR_RESET=$'\033[0m'
    COLOR_BOLD=$'\033[1m'
    COLOR_GREEN=$'\033[32m'
    COLOR_RED=$'\033[31m'
    COLOR_YELLOW=$'\033[33m'
    COLOR_CYAN=$'\033[36m'
else
    COLOR_RESET=""
    COLOR_BOLD=""
    COLOR_GREEN=""
    COLOR_RED=""
    COLOR_YELLOW=""
    COLOR_CYAN=""
fi

print_pass() {
    printf '%bPASS%b %s\n' "${COLOR_GREEN}" "${COLOR_RESET}" "$*"
}

print_fail() {
    printf '%bFAIL%b %s\n' "${COLOR_RED}${COLOR_BOLD}" "${COLOR_RESET}" "$*"
}

print_info() {
    printf '%bINFO%b %s\n' "${COLOR_CYAN}" "${COLOR_RESET}" "$*"
}

print_skip() {
    printf '%bSKIP%b %s\n' "${COLOR_YELLOW}" "${COLOR_RESET}" "$*"
}

human_size() {
    local bytes="$1"
    if (( bytes >= 1048576 )); then
        printf "%.2f MB" "$(awk "BEGIN {print ${bytes}/1048576}")"
    elif (( bytes >= 1024 )); then
        printf "%.1f KB" "$(awk "BEGIN {print ${bytes}/1024}")"
    else
        printf "%d B" "${bytes}"
    fi
}

# Parse the PDF with veraPDF (validation off). Success means openable structure.
check_valid_pdf() {
    local pdf="$1"

    if [[ ! -s "${pdf}" ]]; then
        echo "invalid|empty or missing file"
        return
    fi

    local output=""
    local exit_code=0
    output="$("${VERAPDF}" --off --extract lowLevelInfo --format text --loglevel 0 "${pdf}" 2>&1)" || exit_code=$?

    if (( exit_code == 0 )); then
        echo "valid|veraPDF parsed successfully"
        return
    fi

    local msg="${output//$'\n'/; }"
    msg="${msg//$'\r'/}"
    if [[ -z "${msg}" ]]; then
        msg="veraPDF parse failed (exit ${exit_code})"
    fi
    echo "invalid|${msg}"
}

check_structure_tree() {
    local pdf="$1"

    if [[ "${VERIFY_STRUCTURE_TREE}" != "1" ]]; then
        echo "skip|structure-tree checks disabled"
        return 0
    fi
    if [[ ! -f "${STRUCTURE_TREE_CHECK}" ]]; then
        echo "skip|structure_tree_check.py not found"
        return 0
    fi

    local output exit_code=0
    output="$(python3 "${STRUCTURE_TREE_CHECK}" "${pdf}" 2>&1)" || exit_code=$?
    if ((exit_code == 0)); then
        echo "ok|$(printf '%s\n' "${output}" | tail -1)"
    else
        echo "fail|${output}"
    fi
}

check_avalpdf() {
    local pdf="$1"

    if [[ "${VERIFY_AVALPDF}" != "1" ]]; then
        echo "skip|avalpdf checks disabled"
        return 0
    fi
    if [[ ! -x "${AVALPDF}" ]]; then
        echo "skip|avalpdf not installed (make install-pdf-validators)"
        return 0
    fi

    local tmpdir report_json exit_code=0
    tmpdir="$(mktemp -d)"
    "${AVALPDF}" "${pdf}" --report -o "${tmpdir}" --quiet 2>/dev/null || exit_code=$?
    report_json="$(find "${tmpdir}" -maxdepth 1 -name '*validation_report.json' -print -quit 2>/dev/null || true)"

    if ((exit_code != 0)); then
        rm -rf "${tmpdir}"
        echo "fail|avalpdf exited ${exit_code}"
        return
    fi
    if [[ -z "${report_json}" ]]; then
        rm -rf "${tmpdir}"
        echo "fail|avalpdf produced no validation report"
        return
    fi

    local counts
    counts="$(python3 - "${report_json}" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as handle:
    payload = json.load(handle)
results = payload.get("validation_results", payload)
issues = len(results.get("issues") or [])
warnings = len(results.get("warnings") or [])
print(f"{issues} {warnings}")
PY
)"
    rm -rf "${tmpdir}"
    local issues="${counts%% *}"
    local warnings="${counts##* }"

    if [[ "${VERIFY_AVALPDF_STRICT}" == "1" && "${issues}" -gt 0 ]]; then
        echo "fail|avalpdf ${issues} issue(s), ${warnings} warning(s)"
    elif [[ "${issues}" -gt 0 || "${warnings}" -gt 0 ]]; then
        echo "warn|avalpdf ${issues} issue(s), ${warnings} warning(s) (non-blocking)"
    else
        echo "ok|avalpdf clean"
    fi
}

check_verapdf_compliance() {
    local pdf="$1"
    local json_dir="$2"
    shift 2
    local flavours=("$@")

    local ok=true
    local details=()
    local json_files=()

    mkdir -p "${json_dir}"

    for flavour in "${flavours[@]}"; do
        local json_out="${json_dir}/$(basename "${pdf}")_${flavour}.json"
        local output
        local exit_code=0
        output="$(
            python3 "${VERAPDF_REPORT}" check \
                --verapdf "${VERAPDF}" \
                --pdf "${pdf}" \
                --flavour "${flavour}" \
                --json-out "${json_out}" \
                --sampledata "${FIXTURES}/" \
                2>&1
        )" || exit_code=$?

        json_files+=("${json_out}")
        if [[ -n "${output}" ]]; then
            printf '%s\n' "${output}" >&2
        fi

        if ((exit_code == 0)); then
            details+=("PASS ${flavour}")
        else
            ok=false
            details+=("FAIL ${flavour}")
        fi
    done

    if [[ "${ok}" == true ]]; then
        echo "compliant|$(IFS='; '; echo "${details[*]}")|$(IFS=','; echo "${json_files[*]}")"
    else
        echo "not compliant|$(IFS='; '; echo "${details[*]}")|$(IFS=','; echo "${json_files[*]}")"
    fi
}

wait_for_slot() {
    local -n _running_ref="$1"
    local max_jobs="$2"
    while ((_running_ref >= max_jobs)); do
        if ! wait -n 2>/dev/null; then
            wait || true
        fi
        ((_running_ref--)) || true
    done
}

collect_pdfs() {
    local -n _out="$1"
    _out=()
    if ((${#EXPLICIT_PDFS[@]} > 0)); then
        _out=("${EXPLICIT_PDFS[@]}")
        return
    fi
    if [[ ! -d "${FIXTURES}" ]]; then
        echo "Fixtures directory not found: ${FIXTURES}" >&2
        echo "Place PDFs under compliance/fixtures/ or pass --pdf paths." >&2
        exit 1
    fi
    while IFS= read -r -d '' pdf; do
        _out+=("${pdf}")
    done < <(find "${FIXTURES}" -type f -iname '*.pdf' -print0 | sort -z)
}

# Display path relative to fixtures when possible.
rel_path() {
    local pdf="$1"
    local prefix="${FIXTURES%/}/"
    if [[ "${pdf}" == "${prefix}"* ]]; then
        echo "${pdf#${prefix}}"
    else
        echo "${pdf}"
    fi
}

verify_one_pdf() {
    local pdf="$1"
    local json_dir="$2"
    local failures=0
    local rel
    rel=$(rel_path "${pdf}")

    echo ""
    printf '%b==> %s%b\n' "${COLOR_BOLD}" "${rel}" "${COLOR_RESET}"

    if [[ ! -f "${pdf}" ]]; then
        print_skip "${rel}: file not found"
        return 0
    fi

    local validity_result validity_status validity_details
    validity_result=$(check_valid_pdf "${pdf}")
    validity_status="${validity_result%%|*}"
    validity_details="${validity_result#*|}"
    if [[ "${validity_status}" == "valid" ]]; then
        print_pass "valid pdf: ${validity_details}"
    else
        print_fail "valid pdf: ${validity_details}"
        ((failures++)) || true
    fi

    local gen_size
    gen_size=$(stat -c '%s' "${pdf}")
    print_info "size: ${gen_size} bytes ($(human_size "${gen_size}"))"

    IFS=',' read -ra flavours <<< "${COMPLIANCE_FLAVOURS}"
    local compliance_result status details json_list
    compliance_result=$(check_verapdf_compliance "${pdf}" "${json_dir}" "${flavours[@]}")
    status="${compliance_result%%|*}"
    details="${compliance_result#*|}"
    json_list="${details##*|}"
    details="${details%%|*}"
    if [[ "${status}" == "compliant" ]]; then
        print_pass "compliance: ${details}"
    else
        print_fail "compliance: ${details}"
        ((failures++)) || true
    fi
    if [[ -n "${json_list}" ]]; then
        printf 'COMPLIANCE_JSON:%s\n' "${json_list}"
    fi

    local struct_result struct_status struct_details
    struct_result=$(check_structure_tree "${pdf}")
    struct_status="${struct_result%%|*}"
    struct_details="${struct_result#*|}"
    case "${struct_status}" in
        ok)
            print_pass "structure-tree: ${struct_details}"
            ;;
        fail)
            print_fail "structure-tree: ${struct_details}"
            ((failures++)) || true
            ;;
        skip)
            print_skip "structure-tree: ${struct_details}"
            ;;
    esac

    local aval_result aval_status aval_details
    aval_result=$(check_avalpdf "${pdf}")
    aval_status="${aval_result%%|*}"
    aval_details="${aval_result#*|}"
    case "${aval_status}" in
        ok)
            print_pass "avalpdf: ${aval_details}"
            ;;
        warn)
            print_info "avalpdf: ${aval_details}"
            ;;
        fail)
            print_fail "avalpdf: ${aval_details}"
            ((failures++)) || true
            ;;
        skip)
            print_skip "avalpdf: ${aval_details}"
            ;;
    esac

    return "${failures}"
}

run_verify() {
    local -a pdfs=()
    collect_pdfs pdfs

    if ((${#pdfs[@]} == 0)); then
        print_skip "No PDF files found under ${FIXTURES}"
        print_info "Add fixtures (see plans phase 4/5) or pass explicit paths."
        exit 0
    fi

    echo "PDF compliance validation (${#pdfs[@]} PDF(s), flavours=${COMPLIANCE_FLAVOURS}, ${PARALLEL_JOBS} workers)..."

    local tmpdir
    tmpdir=$(mktemp -d)
    trap 'rm -rf "${tmpdir}"' RETURN

    local idx=0
    local running=0
    local -a compliance_json_files=()

    for pdf in "${pdfs[@]}"; do
        wait_for_slot running "${PARALLEL_JOBS}"
        (
            set +e
            verify_one_pdf "${pdf}" "${tmpdir}/compliance" > "${tmpdir}/${idx}.log" 2>&1
            echo $? > "${tmpdir}/${idx}.rc"
        ) &
        ((running++)) || true
        ((idx++)) || true
    done

    while ((running > 0)); do
        wait -n 2>/dev/null || wait || true
        ((running--)) || true
    done

    local failures=0
    local i
    for ((i = 0; i < idx; i++)); do
        while IFS= read -r line; do
            if [[ "${line}" == COMPLIANCE_JSON:* ]]; then
                local json_list="${line#COMPLIANCE_JSON:}"
                local json_path
                IFS=',' read -ra json_paths <<< "${json_list}"
                for json_path in "${json_paths[@]}"; do
                    if [[ -f "${json_path}" ]]; then
                        compliance_json_files+=("${json_path}")
                    fi
                done
                continue
            fi
            printf '%s\n' "${line}"
        done < "${tmpdir}/${i}.log"
        failures=$((failures + $(cat "${tmpdir}/${i}.rc")))
    done

    if ((${#compliance_json_files[@]} > 0)); then
        python3 "${VERAPDF_REPORT}" table --sampledata "${FIXTURES}/" "${compliance_json_files[@]}"
    fi

    echo ""
    if ((failures > 0)); then
        print_fail "PDF compliance validation failed (${failures} issue(s))"
        exit 1
    fi
    print_pass "PDF compliance validation passed"
}

scan_one_pdf() {
    local pdf="$1"
    local rel size_bytes size_human result status details status_label
    rel=$(rel_path "${pdf}")
    size_bytes=$(stat -c '%s' "${pdf}")
    size_human=$(human_size "${size_bytes}")

    result=$(check_valid_pdf "${pdf}")
    status="${result%%|*}"
    details="${result#*|}"

    if [[ "${status}" == "valid" ]]; then
        status_label="valid"
    else
        status_label="invalid"
    fi

    details="${details//|/\\|}"
    printf "| %s | %s (%s) | %s | %s |\n" "${rel}" "${size_human}" "${size_bytes}" "${status_label}" "${details}"
    if [[ "${status}" == "valid" ]]; then
        return 0
    fi
    return 1
}

scan_one_pdf_compliance() {
    local pdf="$1"
    local json_dir="$2"
    local -a json_files=()
    IFS=',' read -ra flavours <<< "${COMPLIANCE_FLAVOURS}"
    local flavour

    mkdir -p "${json_dir}"

    for flavour in "${flavours[@]}"; do
        local json_out="${json_dir}/$(basename "${pdf}")_${flavour}.json"
        python3 "${VERAPDF_REPORT}" check \
            --verapdf "${VERAPDF}" \
            --pdf "${pdf}" \
            --flavour "${flavour}" \
            --json-out "${json_out}" \
            --sampledata "${FIXTURES}/" \
            >/dev/null 2>&1 || true
        json_files+=("${json_out}")
    done

    printf 'COMPLIANCE_JSON:%s\n' "$(IFS=','; echo "${json_files[*]}")"
}

scan_all() {
    local with_compliance=false
    if [[ "${1:-}" == "--compliance" ]]; then
        with_compliance=true
    fi

    local -a pdfs=()
    collect_pdfs pdfs

    if ((${#pdfs[@]} == 0)); then
        print_skip "No PDF files found under ${FIXTURES}"
        exit 0
    fi

    echo "Scanning PDFs under ${FIXTURES} with veraPDF parse check (${PARALLEL_JOBS} workers)..."
    if [[ "${with_compliance}" == true ]]; then
        echo "Also checking flavours=${COMPLIANCE_FLAVOURS} for each PDF."
    fi
    echo ""
    printf "| PDF | Size | Valid | Details |\n"
    printf "|-----|------|-------|----------|\n"

    local tmpdir
    tmpdir=$(mktemp -d)
    trap 'rm -rf "${tmpdir}"' RETURN

    local total=${#pdfs[@]}
    local idx=0
    local running=0
    local -a compliance_json_files=()

    for pdf in "${pdfs[@]}"; do
        wait_for_slot running "${PARALLEL_JOBS}"
        (
            set +e
            scan_one_pdf "${pdf}" > "${tmpdir}/${idx}.row"
            echo $? > "${tmpdir}/${idx}.status"
            if [[ "${with_compliance}" == true ]]; then
                scan_one_pdf_compliance "${pdf}" "${tmpdir}/compliance" > "${tmpdir}/${idx}.compliance"
            fi
        ) &
        ((running++)) || true
        ((idx++)) || true
    done

    while ((running > 0)); do
        wait -n 2>/dev/null || wait || true
        ((running--)) || true
    done

    local valid=0 invalid=0
    local i
    for ((i = 0; i < total; i++)); do
        cat "${tmpdir}/${i}.row"
        if [[ "$(cat "${tmpdir}/${i}.status")" == "0" ]]; then
            ((valid++)) || true
        else
            ((invalid++)) || true
        fi
        if [[ "${with_compliance}" == true && -f "${tmpdir}/${i}.compliance" ]]; then
            while IFS= read -r line; do
                if [[ "${line}" == COMPLIANCE_JSON:* ]]; then
                    local json_list="${line#COMPLIANCE_JSON:}"
                    local json_path
                    IFS=',' read -ra json_paths <<< "${json_list}"
                    for json_path in "${json_paths[@]}"; do
                        if [[ -f "${json_path}" ]]; then
                            compliance_json_files+=("${json_path}")
                        fi
                    done
                fi
            done < "${tmpdir}/${i}.compliance"
        fi
    done

    echo ""
    if [[ "${with_compliance}" == true && ${#compliance_json_files[@]} -gt 0 ]]; then
        python3 "${VERAPDF_REPORT}" table --sampledata "${FIXTURES}/" "${compliance_json_files[@]}"
        echo ""
    fi

    echo "Scan complete: ${valid} valid, ${invalid} invalid, ${total} total"
    if ((invalid > 0)); then
        exit 1
    fi
}

case "${MODE}" in
    verify)
        run_verify
        ;;
    scan-all)
        scan_all
        ;;
    scan-all-compliance)
        scan_all --compliance
        ;;
    *)
        echo "Unknown mode: ${MODE}" >&2
        exit 2
        ;;
esac
