# veraPDF Compliance Harness

## Installation

veraPDF requires **Java 11+**. Download from [verapdf.org](https://verapdf.org/).

```bash
# Download the latest installer from verapdf.org, then:
java -jar verapdf-installer-*-SNAPSHOT.zip
# Or use the project Makefile:
make install-verapdf
```

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `VERAPDF_BIN` | `verapdf` | Path to veraPDF CLI |

Override the binary path without modifying scripts:

```bash
export VERAPDF_BIN=/opt/verapdf/verapdf
```

## Usage

```bash
# Validate a PDF against PDF/A-4
./compliance/verapdf/run_verapdf.sh -f 4 compliance/fixtures/minimal-text.pdf

# Validate against PDF/UA-2
./compliance/verapdf/run_verapdf.sh -f ua2 compliance/fixtures/minimal-text.pdf

# Pipe through the Python report parser
verapdf -f 4 file.pdf --format xml | python3 compliance/verapdf/report.py
```

The `-f` flag accepts `4` (PDF/A-4) or `ua2` (PDF/UA-2).

## ISO Standards

- **PDF/A-4** — ISO 19005-4:2020 (long-term archiving)
- **PDF/UA-2** — ISO 14289-2:2024 (universal accessibility)

## Dependencies

- `compliance/verapdf/report.py` requires **Python 3**
- `tools/structure_tree_check.py` requires **Python 3** with `pdfminer.six` or `pypdf`
