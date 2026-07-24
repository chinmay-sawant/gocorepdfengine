# Phase 4 — PDF/A-4 Compliance

**Status:** Not started  
**Depends on:** Phase 3 complete (embedded fonts); phase 2 recommended for image/color cases  
**Base plan refs:** §3, §6.2/6.4/6.6, §7, §9, §11 Phase D, §12

---

## Goal

Emit documents that **claim and pass PDF/A-4** (ISO 19005-4 / PDF 2.0 archival) under veraPDF flavour `4`.

---

## Package layout

- [ ] `engine/meta/` — XMP packet builder
- [ ] `engine/color/` — sRGB + Gray ICC profiles (pre-compressed)
- [ ] `engine/pdfa/` — OutputIntent, A-4 rules, trailer Info omission
- [ ] `compliance/verapdf/` — real checks for flavour `4`

---

## Checklist — mode rules

- [ ] `ModePDFA4` enables this phase’s object set
- [ ] `ModeEmbedFonts` forced true under A-4
- [ ] No bare unembedded standard fonts in A-4 mode
- [ ] Optional: `ModePDFA4` implies tagged later (phase 5); for pure A-4 first fixture, keep untagged only if validator allows — **prefer enabling tagging after phase 5**; first A-4 gate can be A-4-only fixtures if needed

> Note: gopdfsuit pairs A-4 with UA-2 in production. For this phase, prioritize **veraPDF `-f 4` PASS**. Full dual claim is phase 5 exit.

## Checklist — XMP metadata stream

Object tags:
- [ ] `/Type /Metadata`
- [ ] `/Subtype /XML`
- [ ] `/Length …`

Packet:
- [ ] `xpacket` begin with BOM + id `W5M0MpCehiHzreSzNTczkc9d`
- [ ] `xpacket end="w"` + padding
- [ ] `pdfaid:part` = **4**
- [ ] `pdfaid:rev` = **2020**
- [ ] `xmp:CreateDate`, `xmp:ModifyDate`, `xmp:MetadataDate`
- [ ] `xmp:CreatorTool`
- [ ] `dc:format` = `application/pdf`
- [ ] Optional: `dc:title`, `dc:creator`, `dc:description`, `dc:subject`
- [ ] `pdf:Producer`
- [ ] `xmpMM:DocumentID`, `xmpMM:InstanceID` (`uuid:…`)

Catalog:
- [ ] `/Metadata <id> 0 R`

## Checklist — OutputIntent + ICC

### ICC sRGB stream
- [ ] `/N 3`
- [ ] `/Alternate /DeviceRGB`
- [ ] `/Filter /FlateDecode`
- [ ] Valid ICC v2.1 profile bytes

### ICC Gray stream
- [ ] `/N 1`
- [ ] `/Alternate /DeviceGray`
- [ ] `/Filter /FlateDecode`

### OutputIntent
- [ ] `/Type /OutputIntent`
- [ ] `/S /GTS_PDFA1`
- [ ] `/OutputConditionIdentifier` (e.g. sRGB IEC61966-2.1)
- [ ] `/RegistryName` (e.g. http://www.color.org)
- [ ] `/Info`
- [ ] `/DestOutputProfile` → sRGB ICC

### Catalog
- [ ] `/OutputIntents [ <oi> 0 R ]`

### Page resources
- [ ] `/ColorSpace << /DefaultRGB [/ICCBased <srgb> 0 R] /DefaultGray [/ICCBased <gray> 0 R] >>`

## Checklist — images under A-4

- [ ] Image `/ColorSpace` rewritten to `[/ICCBased <srgb> 0 R]` (not bare `/DeviceRGB`)
- [ ] JPEG/PNG filters unchanged (`/DCTDecode` / `/FlateDecode`)

## Checklist — trailer / Info

- [ ] **Omit** trailer `/Info` under A-4
- [ ] Do **not** emit Info dictionary object under A-4
- [ ] Keep trailer `/ID`
- [ ] Keep trailer `/Root`, `/Size`

## Checklist — fonts under A-4

- [ ] All text uses Liberation (or other fully embedded) fonts via phase 3 chain
- [ ] No incomplete Type1 standard-only font objects in A-4 mode

## Checklist — object ID reservation (before catalog write)

- [ ] Reserve Metadata object ID
- [ ] Reserve ICC sRGB ID
- [ ] Reserve OutputIntent ID
- [ ] Reserve ICC Gray ID
- [ ] Catalog refs use reserved IDs (no post-hoc placeholder rewrite)

## Checklist — non-goals (defer)

- [ ] PDF/A-4e / 4f
- [ ] Encryption under A-4
- [ ] External streams / JS

---

## Fixture matrix (A-4)

| Fixture | Description | veraPDF `-f 4` |
|---------|-------------|----------------|
| `minimal-text` | One page, Liberation, XMP+OI | PASS |
| `table-simple` | 3×3 table, embedded fonts | PASS |
| `figure-image` | Image with ICCBased | PASS |

- [ ] Generate each fixture from engine
- [ ] Store under `compliance/verapdf/fixtures/` or temp in tests

---

## veraPDF test checklist

- [ ] Install path documented; `VERAPDF_BIN` override
- [ ] `run_verapdf.sh` runs `-f 4`
- [ ] Go test: `TestCompliance_PDFA4_Minimal` (skip if no veraPDF)
- [ ] Go test: `TestCompliance_PDFA4_Table` (optional same phase)
- [ ] CI job placeholder for compliance (may be manual first)

### Failure triage (A-4)

| Symptom | Check |
|---------|--------|
| Metadata failures | `pdfaid:part/rev`, xpacket framing, Info present |
| Font failures | unembedded standard fonts, width mismatch |
| Color failures | DeviceRGB images, missing OutputIntent/ICC |

---

## Acceptance criteria

- [ ] `minimal-text` **PASS** veraPDF flavour `4`
- [ ] No trailer `/Info` in A-4 mode
- [ ] Catalog has `/Metadata` and `/OutputIntents`
- [ ] Page resources include DefaultRGB/DefaultGray ICCBased

---

## Done when

`ModePDFA4` documents pass veraPDF PDF/A-4 on the phase-4 fixture set.
