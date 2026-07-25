package font

// TTF offset table constants
const (
	ttfDirOffset   = 12 // Directory starts at byte 12 in TTF offset table
	ttfEntrySize   = 16 // Each table directory entry is 16 bytes
	ttfHeadMinLen  = 54 // head table minimum length
	ttfHheaMinLen  = 36 // hhea table minimum length
	ttfMaxpMinLen  = 6  // maxp table minimum (version + numGlyphs)
	ttfPostNameLen = 34 // post table name entry length
)

// TTF entry sizes
const (
	ttfWordSize  = 2 // uint16 / word size in bytes
	ttfDWordSize = 4 // uint32 / dword size in bytes
)

// TTF checksum magic
const ttfOffsetMagic = 0xB1B0AFBA

// Scaling
const (
	ttfUPEm    = 1000.0 // Units per em (PDF standard em-unit)
	ttfUPEmInt = 1000
)

// Capacity hints
const (
	ttfGlyphMapHint = 256  // Glyph count hint for map allocation
	ttfCmapBufSize  = 1024 // CMap buffer pre-allocation size
)

// Bit shift amounts
const (
	shift4  = 4
	shift8  = 8
	shift12 = 12
	shift16 = 16
	shift24 = 24
)

// Padding
const pad4Mask = 3 // mask for 4-byte alignment: (n+3) & ^3

// PDF rounding
const roundingHalf = 0.5

// CMap format constants
const (
	cmapFmt4  = 4  // Unicode BMP format 4 subtable
	cmapFmt12 = 12 // Unicode full format 12 subtable
)

// CMap format 4 constants
const (
	cmapF4BaseOffset  = 14 // format 4 subtable data offset (header size)
	cmapF4EntrySize   = 2  // uint16 element size for segment arrays
	cmapF4ReservedPad = 2  // reserved bytes between endCodes and startCodes

	cmapF4StartCodesSeg = 2 // segCount*2 bytes to startCodes (1 array + reserved)
	cmapF4IDDeltasSeg   = 4 // segCount*4 bytes to idDeltas (2 arrays + reserved)
	cmapF4IDRangeSeg    = 6 // segCount*6 bytes to idRangeOffsets (3 arrays + reserved)
)

// CMap format 0
const cmapF0MinLen = 262 // format 0: 256 glyph index entries + 6 byte header

// CMap format 12
const cmapF12MinLen = 16

// CMap header sizes
const (
	cmapHeaderLen  = 4 // version(2) + numTables(2)
	cmapEncRecSize = 8 // platformID(2) + encodingID(2) + offset(4)
)

// maxp factory table
const maxpFactoryLen = 32

// hmtx
const hmtxEntrySize = 4 // advance width(2) + lsb(2)

// HHEA offsets
const (
	hheaAscentOff  = 4 // ascent at byte 4
	hheaDescentOff = 6 // descent at byte 6
)

// head table
const headLocaFmtOff = 50 // indexToLocFormat at byte 50

// Width array sizing
const widthRangeDim = 3 // elements per W entry: first CID, last CID, width

// Liberation font registry
const stdFontCountHint = 12

// Post table
const postMinLen = 32

// OS/2 table
const os2MinLen = 86

// Name table
const (
	ttfNameHeaderLen = 6  // name table header (format + count + stringOffset)
	ttfNameEntrySize = 12 // name record entry size
)

// Font weight thresholds
const (
	weightThreshold = 500
	weightBold      = 700
)

// StemV defaults
const (
	stemVLight  = 50
	stemVNormal = 70
	stemVBold   = 85
)

// Font flag bits
const (
	flagSerifBit = 0x02 // serif bit in head.flags
	flagMonoBit  = 0x04 // mono bit in macStyle
)

// Uint16 mask
const maskU16 = 0xFFFF

// Fixed-point fraction
const fixed16Fraction = 65536.0 // 1/65536 for 16.16 fixed format

// OS/2 fsSelection bit positions
const fsRegularBit = 5

// PDF font descriptor flag bits
const flagSymbolicBit = 2
