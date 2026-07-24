# Phase 5 — PDF/UA-2 Tagging

**Status:** Not started  
**Depends on:** Phase 4 recommended (XMP already present); phase 1–3 required  
**Base plan refs:** §6.2/6.4/6.5, §8, §11 Phase E, §12

---

## Goal

Emit **tagged** PDF 2.0 that claims and passes **PDF/UA-2** (ISO 14289-2:2024) under veraPDF flavour `ua2`, with structure-tree rules strict enough for PAC/Adobe-class ParentTree ownership.

Default end state: **PDF/A-4 + PDF/UA-2** together.

---

## Package layout

- [ ] `engine/structure/` — StructureManager, MCID, ParentTree, StructElem emit
- [ ] Extend `engine/content/` — BDC / EMC helpers
- [ ] Extend `engine/layout/` — table/heading structure hooks
- [ ] Extend `engine/page/` — `/StructParents`, `/Tabs`
- [ ] Extend `engine/meta/` — `pdfuaid` + extension schema
- [ ] `tools/structure_tree_check.py` (or Go port) — ParentTree leaf ownership

---

## Checklist — mode rules

- [ ] `ModePDFUA2` / `Tagged` enables structure manager
- [ ] `ModePDFA4` implies tagged (match gopdfsuit: `tagged := Tagged || PDFA`)
- [ ] Structure methods no-op when disabled (untagged path stays fast)

## Checklist — catalog (UA-2)

- [ ] `/Lang (en-US)` (or configurable)
- [ ] `/MarkInfo << /Marked true >>`
- [ ] `/StructTreeRoot <id> 0 R`
- [ ] `/ViewerPreferences << /DisplayDocTitle true >>`
- [ ] `/Metadata` present (from phase 4 / always)

## Checklist — XMP UA claim

- [ ] `pdfuaid:part` = **2**
- [ ] `pdfuaid:rev` = **2024**
- [ ] `pdfaExtension` schema registration for `pdfuaid` `part` + `rev`
- [ ] Keep A-4 `pdfaid` fields when dual-mode

## Checklist — Namespace (PDF 2.0)

- [ ] Object: `/Type /Namespace`
- [ ] `/NS (http://iso.org/pdf2/ssn)`
- [ ] StructTreeRoot `/Namespaces [ <ns> 0 R ]`
- [ ] Document StructElem `/NS <ns> 0 R`

## Checklist — StructTreeRoot

- [ ] `/Type /StructTreeRoot`
- [ ] `/K` → Document element
- [ ] `/ParentTree` → number tree
- [ ] `/Namespaces` array

## Checklist — structure types (`/S`)

Implement as needed for fixtures; minimum bold items:

- [ ] **`/Document`** (required top-level)
- [ ] **`/H1`** (title)
- [ ] **`/P`** (paragraph)
- [ ] **`/Table`**, **`/TR`**, **`/TH`**, **`/TD`**
- [ ] `/Figure` + `/Alt` (image fixture)
- [ ] `/Link` + `/OBJR` (link fixture)
- [ ] Optional later: `/Part`, `/Sect`, `/Div`, `/H2`–`/H6`, lists, `/Caption`, `/Form`, `/Reference`

### Table hierarchy (critical)

```
Document → Table → TR → TD|TH
```

- [ ] Each **TD/TH owns its MCID** (not the TR)
- [ ] ParentTree[page][mcid] points to that TD/TH StructElem
- [ ] Leaf StructElem has `/Pg` to the correct page
- [ ] Multi-page tables: TR `/Pg` consistent with child TD pages (structure_tree_check)

## Checklist — StructElem keys

- [ ] `/Type /StructElem`
- [ ] `/S /…`
- [ ] `/P` parent (Document’s parent = StructTreeRoot)
- [ ] `/K` kids (MCIDs, child refs, OBJR)
- [ ] `/Pg` where required
- [ ] `/T` title when needed
- [ ] `/Alt` for figures

### Link OBJR

- [ ] `/K [ << /Type /OBJR /Obj <annot> 0 R /Pg <page> 0 R >> ]`

## Checklist — marked content (streams)

- [ ] Emit `/<S> << /MCID n >> BDC` … `EMC`
- [ ] Optional `/Alt (…)` in BDC properties
- [ ] Per-page MCID counter from 0
- [ ] Pagination chrome as `/Artifact << /Attached [/Top] /Type /Pagination >> BDC` … `EMC` when appropriate

## Checklist — ParentTree

- [ ] Number tree: `<< /Nums [ pageKey [ elemRefs… ] … ] >>`
- [ ] Array index = MCID
- [ ] Annotation StructParent keys map to Link StructElem
- [ ] Page `/StructParents` only when page has MCIDs

## Checklist — page UA extras

- [ ] `/StructParents n` when tagged content exists
- [ ] `/Tabs /S` when page has annotations (ISO 14289-2 8.9.3.3)

## Checklist — object emit order

- [ ] Reserve StructTreeRoot ID before catalog
- [ ] Emit: Namespace → StructTreeRoot → ParentTree → all StructElem objects
- [ ] Assign StructElem object IDs iteratively (parent-before-children)

---

## Fixture matrix (UA-2 + dual)

| Fixture | Description | `-f 4` | `-f ua2` | structure_tree |
|---------|-------------|--------|----------|----------------|
| `minimal-text` | H1 or P + Document | PASS | PASS | PASS |
| `table-simple` | TH+TD ownership | PASS | PASS | PASS |
| `table-multipage` | `/Pg` + ParentTree stress | PASS | PASS | PASS |
| `heading-title` | H1 + body | PASS | PASS | PASS |
| `figure-alt` | Figure + Alt | PASS | PASS | PASS |
| `link-annot` | Link OBJR + Tabs /S | PASS | PASS | PASS |

- [ ] All rows green before calling phase 5 done

---

## veraPDF + structure test checklist

- [ ] `TestCompliance_PDFUA2_Minimal`
- [ ] `TestCompliance_PDFUA2_Table`
- [ ] `TestCompliance_Matrix_AllFixtures` (4 + ua2)
- [ ] `TestCompliance_StructureTree_ParentTreeOwnership`
- [ ] `run_verapdf.sh` supports `-f ua2`
- [ ] structure_tree_check fails if ParentTree points at TR for TD MCID

### Failure triage (UA-2)

| Symptom | Check |
|---------|--------|
| Missing structure | StructTreeRoot / MarkInfo / Lang |
| ParentTree errors | MCID owned by TR; Nums order |
| PAC fail, veraPDF pass | Same ParentTree / multi-page `/Pg` |
| Link issues | missing OBJR or `/Tabs /S` |

---

## Acceptance criteria

- [ ] Dual-mode fixtures PASS veraPDF **`-f 4`** and **`-f ua2`**
- [ ] structure_tree_check PASS on table fixtures
- [ ] Catalog contains MarkInfo, StructTreeRoot, Lang, ViewerPreferences
- [ ] XMP contains both pdfaid (4/2020) and pdfuaid (2/2024)

---

## Done when

Compliant mode = PDF 2.0 + PDF/A-4 + PDF/UA-2 with automated gates green on the fixture matrix above.
