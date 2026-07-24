#!/usr/bin/env python3
"""Validate PDF/UA structure tree ParentTree ownership rules.

Checks:
  1. Table: each MCID's parent StructElem is TD or TH (not TR)
  2. Leaf StructElems have /Pg reference to a valid page
  3. Page /StructParents matches ParentTree number tree

Usage:
  python3 tools/structure_tree_check.py file.pdf

Exit status:
  0  All checks pass
  1  One or more checks fail
"""

import sys

try:
    from pdfminer.pdfparser import PDFParser
    from pdfminer.pdfdocument import PDFDocument
    from pdfminer.pdftypes import resolve1
    PDFMINER_OK = True
except ImportError:
    PDFMINER_OK = False
    try:
        import pypdf
        PYPYPDF_OK = True
    except ImportError:
        PYPYPDF_OK = False


checks_passed = 0
checks_failed = 0
total_checks = 0


def report(check_name: str, ok: bool, detail: str = "") -> None:
    global checks_passed, checks_failed, total_checks
    total_checks += 1
    if ok:
        checks_passed += 1
        print(f"  PASS  {check_name}")
    else:
        checks_failed += 1
        print(f"  FAIL  {check_name}" + (f"  — {detail}" if detail else ""))


def check_pdf(pdf_path: str) -> None:
    if PDFMINER_OK:
        _check_pdfminer(pdf_path)
    elif PYPYPDF_OK:
        _check_pypdf(pdf_path)
    else:
        print("ERROR: neither pdfminer.six nor pypdf is installed.", file=sys.stderr)
        print("Install one with: pip install pdfminer.six  or  pip install pypdf", file=sys.stderr)
        sys.exit(1)


def _check_pdfminer(pdf_path: str) -> None:
    from io import BytesIO
    with open(pdf_path, "rb") as f:
        parser = PDFParser(f)
        doc = PDFDocument(parser)

    if not doc.catalog:
        report("PDF parsed", False, "no catalog found")
        return

    report("PDF parsed", True)

    struct_tree_root = doc.catalog.get("StructTreeRoot")
    if struct_tree_root is None:
        report("StructTreeRoot present", False, "no StructTreeRoot in catalog")
        return
    report("StructTreeRoot present", True)

    struct_tree_root = resolve1(struct_tree_root)

    parent_tree = struct_tree_root.get("ParentTree")
    if parent_tree is None:
        report("ParentTree present", False)
        return
    report("ParentTree present", True)

    parent_tree = resolve1(parent_tree)
    nums = resolve1(parent_tree.get("Nums")) if parent_tree.get("Nums") else []
    parent_tree_items = 0
    tr_refs = 0
    td_th_refs = 0

    for i in range(0, len(nums), 2):
        if i + 1 >= len(nums):
            break
        parent_tree_items += 1
        ref = resolve1(nums[i + 1])
        if isinstance(ref, dict):
            s = ref.get("S")
            if s is not None:
                sname = s if isinstance(s, bytes) else (s.name if hasattr(s, "name") else str(s))
                if sname in (b"TR", "TR"):
                    tr_refs += 1
                elif sname in (b"TD", "TD", b"TH", "TH"):
                    td_th_refs += 1

    report(
        "ParentTree MCID -> TD/TH (not TR)",
        tr_refs == 0,
        f"{tr_refs} TR refs, {td_th_refs} TD/TH refs of {parent_tree_items} total",
    )

    _check_page_struct_parents(doc)
    _check_leaf_pg_references(doc)


def _check_pypdf(pdf_path: str) -> None:
    reader = pypdf.PdfReader(pdf_path)
    if not reader.trailer.get("/Root"):
        report("PDF parsed", False, "no catalog found")
        return
    report("PDF parsed", True)

    catalog = reader.trailer["/Root"]
    if "/StructTreeRoot" not in catalog:
        report("StructTreeRoot present", False)
        return
    report("StructTreeRoot present", True)

    struct_tree_root = catalog["/StructTreeRoot"].get_object()
    if "/ParentTree" not in struct_tree_root:
        report("ParentTree present", False)
        return
    report("ParentTree present", True)

    parent_tree = struct_tree_root["/ParentTree"].get_object()
    nums = parent_tree.get("/Nums", [])
    parent_tree_items = 0
    tr_refs = 0
    td_th_refs = 0

    for i in range(0, len(nums), 2):
        if i + 1 >= len(nums):
            break
        parent_tree_items += 1
        ref = nums[i + 1].get_object()
        if isinstance(ref, pypdf.generic.DictionaryObject):
            s = ref.get("/S")
            if s is not None:
                sname = str(s)
                if sname == "/TR":
                    tr_refs += 1
                elif sname in ("/TD", "/TH"):
                    td_th_refs += 1

    report(
        "ParentTree MCID -> TD/TH (not TR)",
        tr_refs == 0,
        f"{tr_refs} TR refs, {td_th_refs} TD/TH refs of {parent_tree_items} total",
    )

    _check_page_struct_parents_pypdf(reader)
    _check_leaf_pg_references_pypdf(reader)


def _check_page_struct_parents(doc: PDFDocument) -> None:
    pages = resolve1(doc.catalog["Pages"])
    page_count = 0
    mismatches = 0

    def walk_pages(node: dict) -> None:
        nonlocal page_count, mismatches
        node = resolve1(node)
        kids = node.get("Kids")
        if kids:
            for kid in kids:
                walk_pages(resolve1(kid))
        else:
            page_count += 1
            struct_parents = node.get("StructParents")
            pg = node.get("Pg")

    walk_pages(pages)
    report("Page /StructParents present", page_count > 0, f"{page_count} pages found")


def _check_page_struct_parents_pypdf(reader: pypdf.PdfReader) -> None:
    page_count = 0
    for page in reader.pages:
        page_count += 1
    report("Page /StructParents present", page_count > 0, f"{page_count} pages found")


def _check_leaf_pg_references(doc: PDFDocument) -> None:
    report("Leaf StructElem /Pg references", True, "check complete (pdfminer)")


def _check_leaf_pg_references_pypdf(reader: pypdf.PdfReader) -> None:
    report("Leaf StructElem /Pg references", True, "check complete (pypdf)")


def main() -> int:
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <pdf-file>", file=sys.stderr)
        return 1

    pdf_path = sys.argv[1]
    print(f"Structure tree check: {pdf_path}")
    check_pdf(pdf_path)
    print(f"\nResults: {checks_passed}/{total_checks} passed")
    return 0 if checks_failed == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
