// Package binary provides ELF, Mach-O, and PE section parsers for Bun SEA binaries.
package binary

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

var (
	// BunTrailer is the trailer appended to valid .bun sections.
	BunTrailer = []byte("\n---- Bun! ----\n")

	// bunBytecodeMarker identifies Bun bytecode bundles.
	bunBytecodeMarker = []byte("// @bun @bytecode")
)

// FindBunSection dispatches to ELF / Mach-O / PE parser by magic bytes.
// Returns the file offset and size of the .bun section.
func FindBunSection(data []byte) (offset int, size int, err error) {
	if len(data) < 4 {
		return 0, 0, fmt.Errorf("data too short to identify binary format")
	}

	head := data[:4]

	// ELF
	if bytes.Equal(head[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		return findBunSectionELF(data)
	}

	// Mach-O (native or fat)
	magic := binary.BigEndian.Uint32(head)
	switch magic {
	case 0xCFFAEDFE, 0xFEEDFACF, // MH_MAGIC_64, MH_CIGAM_64
		0xCEFAEDFE, 0xFEEDFACE, // MH_MAGIC, MH_CIGAM
		0xCAFEBABE, 0xBEBAFECA, // FAT_MAGIC, FAT_CIGAM
		0xCAFEBABF, 0xBFBAFECA: // FAT_MAGIC_64, FAT_CIGAM_64
		return findBunSectionMachO(data)
	}

	// PE
	if head[0] == 'M' && head[1] == 'Z' {
		return findBunSectionPE(data)
	}

	return 0, 0, fmt.Errorf("unknown binary format magic=%x", head)
}

// BunSectionHasValidTrailer checks whether the .bun section ends with the
// Bun trailer, allowing for optional PE file-alignment NUL padding.
func BunSectionHasValidTrailer(data []byte, bunLo, bunHi int) bool {
	if bunHi > len(data) {
		bunHi = len(data)
	}
	section := data[bunLo:bunHi]
	trimmed := bytes.TrimRight(section, "\x00")
	return bytes.HasSuffix(trimmed, BunTrailer)
}

// trailerValidAt bounds-checks a candidate section size before validating
// its trailer.
func trailerValidAt(data []byte, rawOff, size int) bool {
	if size <= 0 || rawOff < 0 || rawOff+size > len(data) {
		return false
	}
	return BunSectionHasValidTrailer(data, rawOff, rawOff+size)
}

// BunSectionIsBytecode returns true when .bun section uses compiled Bun
// bytecode — indicating in-place patching is required.
func BunSectionIsBytecode(data []byte) bool {
	end := 1024
	if end > len(data) {
		end = len(data)
	}
	return bytes.Contains(data[:end], bunBytecodeMarker)
}

// FindActiveBundleBounds locates the patchable region within the .bun section.
//
// Bun SEA ≥ 2.1.119 embeds the main entry bundle PLUS a virtual-filesystem
// (VFS) copy. Patching the VFS copy corrupts Bun's module-loader.
//
// Two layouts:
//
//	Layout A (pre-2.1.150): marker close to section start (gap ≤ 4096).
//	  Try u32 size field at marker-4, or find second marker.
//
//	Layout B (2.1.150+ Windows PE): marker far into section (gap > 4096).
//	  Active bundle = [bunLo, marker) — raw JS source before bytecode.
func FindActiveBundleBounds(data []byte, bunLo, bunHi int) (effLo, effHi int) {
	markerLen := len(bunBytecodeMarker)

	// Step 1: find first marker
	absMarker := bytes.Index(data[bunLo:bunHi], bunBytecodeMarker)
	if absMarker < 0 {
		return bunLo, bunHi // no bytecode — patch whole section
	}
	absMarker += bunLo // convert to absolute offset

	gap := absMarker - bunLo

	// Step 2: Layout B — marker far into section (> 4 KB)
	// Everything before the first bytecode marker is raw JS source.
	if gap > 4096 {
		return bunLo, absMarker
	}

	// Step 3: Layout A — marker close to section start, try size field
	sizeFieldOff := absMarker - 4
	if sizeFieldOff >= bunLo && sizeFieldOff+4 <= len(data) {
		blobSize := int(binary.LittleEndian.Uint32(data[sizeFieldOff : sizeFieldOff+4]))
		activeEnd := absMarker + blobSize
		if blobSize >= 1024 && blobSize <= bunHi-absMarker {
			return absMarker, activeEnd
		}
	}

	// Step 4: size field absent/implausible — find second marker (VFS boundary)
	searchStart := absMarker + markerLen
	if searchStart < bunHi {
		idx := bytes.Index(data[searchStart:bunHi], bunBytecodeMarker)
		if idx >= 0 {
			absMarker2 := searchStart + idx
			return absMarker, absMarker2
		}
	}

	// Step 5: last resort — full section
	return bunLo, bunHi
}

// ── ELF parser ──────────────────────────────────────────────────────────────

func findBunSectionELF(data []byte) (int, int, error) {
	if len(data) < 0x40 {
		return 0, 0, fmt.Errorf("ELF header too short")
	}

	// ELF64 header fields
	eShoff := binary.LittleEndian.Uint64(data[0x28:0x30])     // e_shoff
	eShentsize := binary.LittleEndian.Uint16(data[0x3A:0x3C]) // e_shentsize
	eShnum := binary.LittleEndian.Uint16(data[0x3C:0x3E])     // e_shnum
	eShstrndx := binary.LittleEndian.Uint16(data[0x3E:0x40])  // e_shstrndx

	// String table section header
	strtabShdr := int(eShoff) + int(eShstrndx)*int(eShentsize)
	if strtabShdr+0x28 > len(data) {
		return 0, 0, fmt.Errorf("ELF string table section header out of bounds")
	}
	strtabOff := binary.LittleEndian.Uint64(data[strtabShdr+0x18 : strtabShdr+0x20])
	strtabSize := binary.LittleEndian.Uint64(data[strtabShdr+0x20 : strtabShdr+0x28])

	strtab := data[strtabOff : strtabOff+strtabSize]

	for i := 0; i < int(eShnum); i++ {
		sh := int(eShoff) + i*int(eShentsize)
		if sh+0x28 > len(data) {
			break
		}
		shName := binary.LittleEndian.Uint32(data[sh : sh+4])
		if int(shName) >= len(strtab) {
			continue
		}

		// Read null-terminated name from string table
		nameEnd := bytes.IndexByte(strtab[shName:], 0)
		if nameEnd < 0 {
			continue
		}
		name := strtab[shName : shName+uint32(nameEnd)]
		if string(name) == ".bun" {
			offset := int(binary.LittleEndian.Uint64(data[sh+0x18 : sh+0x20]))
			size := int(binary.LittleEndian.Uint64(data[sh+0x20 : sh+0x28]))
			return offset, size, nil
		}
	}

	return 0, 0, fmt.Errorf(".bun section not found (ELF)")
}

// ── Mach-O parser ───────────────────────────────────────────────────────────

func findBunSectionMachO(data []byte) (int, int, error) {
	if len(data) < 4 {
		return 0, 0, fmt.Errorf("Mach-O data too short")
	}

	magic := binary.BigEndian.Uint32(data[:4])

	// Fat binary — pick first arch entry and recurse.
	switch magic {
	case 0xCAFEBABE, 0xBEBAFECA, 0xCAFEBABF, 0xBFBAFECA:
		return findBunSectionFat(data, magic)
	}

	// Verify Mach-O magic
	switch magic {
	case 0xFEEDFACF, 0xCFFAEDFE, 0xFEEDFACE, 0xCEFAEDFE:
		// valid
	default:
		return 0, 0, fmt.Errorf("not a Mach-O binary")
	}

	is64 := magic == 0xFEEDFACF || magic == 0xCFFAEDFE
	bigEndian := magic == 0xFEEDFACF || magic == 0xFEEDFACE
	var bo binary.ByteOrder = binary.LittleEndian
	if bigEndian {
		bo = binary.BigEndian
	}

	ncmds := bo.Uint32(data[16:20])
	hdrSize := 28
	if is64 {
		hdrSize = 32
	}

	cur := hdrSize
	const lcSegment64 = 0x19
	const lcSegment = 0x01

	for i := uint32(0); i < ncmds; i++ {
		if cur+8 > len(data) {
			break
		}
		cmd := bo.Uint32(data[cur : cur+4])
		cmdsize := bo.Uint32(data[cur+4 : cur+8])

		if cmd == lcSegment64 {
			if cur+72 > len(data) {
				break
			}
			nsects := bo.Uint32(data[cur+64 : cur+68])
			sectBase := cur + 72

			for j := uint32(0); j < nsects; j++ {
				s := sectBase + int(j)*80
				if s+80 > len(data) {
					break
				}
				sectname := readCString(data[s : s+16])
				sectSegname := readCString(data[s+16 : s+32])

				if sectname == "__bun" || sectname == ".bun" || sectSegname == "__BUN" {
					size := int(bo.Uint64(data[s+40 : s+48]))
					offset := int(bo.Uint32(data[s+48 : s+52]))
					return offset, size, nil
				}
			}
		} else if cmd == lcSegment {
			if cur+56 > len(data) {
				break
			}
			nsects := bo.Uint32(data[cur+48 : cur+52])
			sectBase := cur + 56

			for j := uint32(0); j < nsects; j++ {
				s := sectBase + int(j)*68
				if s+68 > len(data) {
					break
				}
				sectname := readCString(data[s : s+16])

				if sectname == "__bun" || sectname == ".bun" {
					size := int(bo.Uint32(data[s+36 : s+40]))
					offset := int(bo.Uint32(data[s+40 : s+44]))
					return offset, size, nil
				}
			}
		}

		cur += int(cmdsize)
	}

	return 0, 0, fmt.Errorf(".bun/__bun section not found (Mach-O)")
}

func findBunSectionFat(data []byte, magic uint32) (int, int, error) {
	isBig := magic == 0xCAFEBABE || magic == 0xCAFEBABF
	var bo binary.ByteOrder = binary.LittleEndian
	if isBig {
		bo = binary.BigEndian
	}

	if len(data) < 8 {
		return 0, 0, fmt.Errorf("fat binary too short")
	}
	nfat := bo.Uint32(data[4:8])

	// Entry size: 20 bytes for 32-bit fat_arch, 32 for 64-bit.
	entrySize := 20
	if magic == 0xCAFEBABF || magic == 0xBFBAFECA {
		entrySize = 32
	}

	for i := uint32(0); i < nfat; i++ {
		base := 8 + int(i)*entrySize
		if base+12 > len(data) {
			break
		}
		archOff := int(bo.Uint32(data[base+8 : base+12]))
		if archOff >= len(data) {
			continue
		}
		sub := data[archOff:]
		subOff, subSize, err := findBunSectionMachO(sub)
		if err == nil {
			return archOff + subOff, subSize, nil
		}
	}

	return 0, 0, fmt.Errorf(".bun section not found in fat binary")
}

// ── PE parser ───────────────────────────────────────────────────────────────

func findBunSectionPE(data []byte) (int, int, error) {
	if len(data) < 0x40 || data[0] != 'M' || data[1] != 'Z' {
		return 0, 0, fmt.Errorf("not a PE binary")
	}

	peOff := int(binary.LittleEndian.Uint32(data[0x3C:0x40]))
	if peOff+4 > len(data) {
		return 0, 0, fmt.Errorf("PE offset out of bounds")
	}
	if !bytes.Equal(data[peOff:peOff+4], []byte("PE\x00\x00")) {
		return 0, 0, fmt.Errorf("PE signature missing")
	}

	coff := peOff + 4
	if coff+20 > len(data) {
		return 0, 0, fmt.Errorf("COFF header too short")
	}
	nsect := int(binary.LittleEndian.Uint16(data[coff+2 : coff+4]))
	optSize := int(binary.LittleEndian.Uint16(data[coff+16 : coff+18]))
	sectTable := coff + 20 + optSize

	// Each IMAGE_SECTION_HEADER = 40 bytes.
	for i := 0; i < nsect; i++ {
		s := sectTable + i*40
		if s+40 > len(data) {
			break
		}
		name := readCString(data[s : s+8])
		if name == ".bun" {
			vsize := int(binary.LittleEndian.Uint32(data[s+8 : s+12]))
			rsize := int(binary.LittleEndian.Uint32(data[s+16 : s+20]))
			rawOff := int(binary.LittleEndian.Uint32(data[s+20 : s+24]))

			// PE SizeOfRawData includes file-alignment padding. VirtualSize is
			// usually the live .bun payload length. Recent Bun SEA builds
			// (CC 2.1.228+) place the trailer at the end of the raw data with
			// rsize > vsize, so probe both bounds and prefer whichever carries
			// a valid trailer.
			if trailerValidAt(data, rawOff, rsize) {
				return rawOff, rsize, nil
			}
			if trailerValidAt(data, rawOff, vsize) {
				return rawOff, vsize, nil
			}
			sz := vsize
			if sz == 0 {
				sz = rsize
			}
			return rawOff, sz, nil
		}
	}

	return 0, 0, fmt.Errorf(".bun section not found (PE)")
}

// ── helpers ──────────────────────────────────────────────────────────────────

// readCString reads a null-terminated string from a fixed-size byte slice.
func readCString(b []byte) string {
	idx := bytes.IndexByte(b, 0)
	if idx < 0 {
		return string(b)
	}
	return string(b[:idx])
}
