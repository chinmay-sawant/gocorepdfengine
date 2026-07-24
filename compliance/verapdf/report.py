#!/usr/bin/env python3
"""Parse veraPDF XML report from stdin and print a human-readable summary.

Usage:
  verapdf -f 4 file.pdf --format xml | python3 compliance/verapdf/report.py

Exit status:
  0  PASSED
  1  FAILED
"""

import sys
import xml.etree.ElementTree as ET


def parse_report(xml_data: str) -> tuple[bool, int, int, list[str]]:
    root = ET.fromstring(xml_data)

    ns = {"vera": "http://www.verapdf.org/ValidationReport"}
    report = root.find("vera:report", ns) or root

    compliant = True
    passed_rules = 0
    failed_rules = 0
    errors: list[str] = []

    for job in report.iter("{http://www.verapdf.org/ValidationReport}job") if report.find(".//{http://www.verapdf.org/ValidationReport}job") is not None else []:
        pass

    validation_result = report.find(".//{http://www.verapdf.org/ValidationReport}validationResult")
    if validation_result is None:
        validation_result = report.find(".//validationResult")
    if validation_result is None:
        for elem in root.iter():
            if "compliant" in elem.tag or "isCompliant" in elem.tag:
                compliant = elem.text.strip().lower() == "true" if elem.text else False
                break

    for elem in root.iter():
        tag = elem.tag.split("}")[-1] if "}" in elem.tag else elem.tag
        if tag in ("compliant", "isCompliant"):
            compliant = elem.text.strip().lower() == "true" if elem.text else False
        elif tag in ("passedRules", "passedChecks"):
            passed_rules = int(elem.text.strip()) if elem.text else 0
        elif tag in ("failedRules", "failedChecks"):
            failed_rules = int(elem.text.strip()) if elem.text else 0

    for elem in root.iter():
        tag = elem.tag.split("}")[-1] if "}" in elem.tag else elem.tag
        if tag == "errorMessage" and elem.text:
            errors.append(elem.text.strip())

    return compliant, passed_rules, failed_rules, errors


def main() -> int:
    xml_data = sys.stdin.read()
    if not xml_data.strip():
        print("FAILED: no XML data on stdin", file=sys.stderr)
        return 1

    try:
        compliant, passed, failed, errors = parse_report(xml_data)
    except ET.ParseError as exc:
        print(f"FAILED: invalid XML: {exc}", file=sys.stderr)
        return 1

    if compliant:
        print(f"PASSED — {passed} rules passed, {failed} failed")
        return 0
    else:
        print(f"FAILED — {passed} passed, {failed} failed")
        for err in errors[:10]:
            print(f"  • {err}")
        if len(errors) > 10:
            print(f"  ... and {len(errors) - 10} more errors")
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
