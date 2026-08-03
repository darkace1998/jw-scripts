package metadata

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// maxMoovSize caps how much of a moov box is loaded into memory (256 MiB);
// real moov boxes are a few megabytes at most.
const maxMoovSize = 256 << 20

// mp4Containers are the box types that are walked when patching chunk
// offsets. Only boxes on the path to stbl matter, but walking a few extra
// container types is harmless.
var mp4Containers = map[string]bool{
	"trak": true,
	"mdia": true,
	"minf": true,
	"stbl": true,
	"edts": true,
	"dinf": true,
	"mvex": true,
}

type mp4Box struct {
	offset    int64 // absolute offset of the box header in the file
	size      int64 // total box size including header
	headerLen int64 // 8, or 16 for 64-bit sizes
	boxType   string
}

// readMP4Boxes reads the metadata of the boxes that make up one container
// level, starting at offset start and ending at end.
func readMP4Boxes(r io.ReaderAt, start, end int64) ([]mp4Box, error) {
	var boxes []mp4Box
	offset := start
	header := make([]byte, 16)

	for offset < end {
		if end-offset < 8 {
			return nil, fmt.Errorf("truncated box header at offset %d", offset)
		}
		if _, err := r.ReadAt(header[:8], offset); err != nil {
			return nil, err
		}

		box := mp4Box{
			offset:    offset,
			size:      int64(binary.BigEndian.Uint32(header[:4])),
			headerLen: 8,
			boxType:   string(header[4:8]),
		}

		switch box.size {
		case 0:
			// Box extends to the end of the enclosing container
			box.size = end - offset
		case 1:
			if end-offset < 16 {
				return nil, fmt.Errorf("truncated 64-bit box header at offset %d", offset)
			}
			if _, err := r.ReadAt(header[8:16], offset+8); err != nil {
				return nil, err
			}
			size := binary.BigEndian.Uint64(header[8:16])
			// #nosec G115 - end > offset is guaranteed by the loop condition
			if size > uint64(end-offset) {
				return nil, fmt.Errorf("box %q at offset %d exceeds container", box.boxType, offset)
			}
			box.size = int64(size) // #nosec G115 - bounded by end-offset above
			box.headerLen = 16
		}

		if box.size < box.headerLen || offset+box.size > end {
			return nil, fmt.Errorf("invalid size %d for box %q at offset %d", box.size, box.boxType, offset)
		}

		boxes = append(boxes, box)
		offset += box.size
	}

	return boxes, nil
}

// writeBox serializes an MP4 box with a 32-bit size header.
func writeBox(boxType string, payload ...[]byte) []byte {
	size := 8
	for _, p := range payload {
		size += len(p)
	}
	out := make([]byte, 0, size)
	var sizeBytes [4]byte
	binary.BigEndian.PutUint32(sizeBytes[:], uint32(size)) // #nosec G115 - metadata boxes are tiny
	out = append(out, sizeBytes[:]...)
	out = append(out, boxType...)
	for _, p := range payload {
		out = append(out, p...)
	}
	return out
}

// buildUdta builds a udta box containing iTunes-style metadata atoms
// (udta > meta > hdlr + ilst). The output is deterministic so re-embedding
// identical metadata can be detected and skipped.
func buildUdta(meta *FileMetadata) []byte {
	item := func(name, value string) []byte {
		if value == "" {
			return nil
		}
		// data atom: type indicator 1 (UTF-8 text) + 4-byte locale + text
		data := writeBox("data", []byte{0, 0, 0, 1, 0, 0, 0, 0}, []byte(value))
		return writeBox(name, data)
	}

	var ilstPayload []byte
	ilstPayload = append(ilstPayload, item("\xa9nam", meta.Title)...)
	ilstPayload = append(ilstPayload, item("\xa9alb", meta.album())...)
	ilstPayload = append(ilstPayload, item("\xa9ART", "jw.org")...)
	ilstPayload = append(ilstPayload, item("\xa9day", meta.dateTag())...)
	ilstPayload = append(ilstPayload, item("\xa9cmt", meta.URL)...)
	ilst := writeBox("ilst", ilstPayload)

	// hdlr full box marking the meta box as iTunes-style metadata
	hdlr := writeBox("hdlr",
		[]byte{0, 0, 0, 0}, // version/flags
		[]byte{0, 0, 0, 0}, // pre_defined
		[]byte("mdir"),     // handler type
		[]byte("appl"),     // reserved
		make([]byte, 9),    // reserved + empty null-terminated name
	)

	metaBox := writeBox("meta", []byte{0, 0, 0, 0}, hdlr, ilst)
	return writeBox("udta", metaBox)
}

// patchChunkOffsets walks the sibling boxes in b and adds delta to every
// stco/co64 chunk offset that points at or beyond the moved position. This
// keeps sample data reachable after the moov box changes size.
func patchChunkOffsets(b []byte, moved, delta int64) error {
	reader := bytes.NewReader(b)
	boxes, err := readMP4Boxes(reader, 0, int64(len(b)))
	if err != nil {
		return err
	}

	for _, box := range boxes {
		payload := b[box.offset+box.headerLen : box.offset+box.size]
		switch {
		case box.boxType == "stco":
			if err := patchOffsetTable(payload, moved, delta, 4); err != nil {
				return err
			}
		case box.boxType == "co64":
			if err := patchOffsetTable(payload, moved, delta, 8); err != nil {
				return err
			}
		case mp4Containers[box.boxType]:
			if err := patchChunkOffsets(payload, moved, delta); err != nil {
				return err
			}
		}
	}
	return nil
}

// patchOffsetTable rewrites a stco (width 4) or co64 (width 8) payload:
// version/flags, entry count, then chunk offsets.
func patchOffsetTable(payload []byte, moved, delta int64, width int) error {
	if len(payload) < 8 {
		return fmt.Errorf("truncated chunk offset table")
	}
	count := int(binary.BigEndian.Uint32(payload[4:8]))
	table := payload[8:]
	if count*width > len(table) {
		return fmt.Errorf("chunk offset table shorter than its entry count")
	}

	for i := 0; i < count; i++ {
		entry := table[i*width : (i+1)*width]
		var offset int64
		if width == 4 {
			offset = int64(binary.BigEndian.Uint32(entry))
		} else {
			v := binary.BigEndian.Uint64(entry)
			if v > uint64(1)<<62 {
				return fmt.Errorf("chunk offset out of range")
			}
			offset = int64(v)
		}
		if offset < moved {
			continue
		}
		offset += delta
		if offset < 0 {
			return fmt.Errorf("chunk offset underflow")
		}
		if width == 4 {
			if offset > int64(^uint32(0)) {
				return fmt.Errorf("chunk offset overflows 32 bits after patch")
			}
			binary.BigEndian.PutUint32(entry, uint32(offset))
		} else {
			binary.BigEndian.PutUint64(entry, uint64(offset))
		}
	}
	return nil
}

// embedMP4 embeds metadata into an MP4 file by replacing the udta box inside
// moov with one containing iTunes-style metadata atoms. Chunk offsets are
// patched when the moov box changes size, and the file is rewritten via a
// temporary file so a failure never corrupts the original.
func embedMP4(path string, meta *FileMetadata) error {
	// #nosec G304 - Path points to a previously downloaded media file
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	topLevel, err := readMP4Boxes(f, 0, fi.Size())
	if err != nil {
		return err
	}

	var moov *mp4Box
	for i := range topLevel {
		if topLevel[i].boxType == "moov" {
			if moov != nil {
				return fmt.Errorf("multiple moov boxes found")
			}
			moov = &topLevel[i]
		}
	}
	if moov == nil {
		return fmt.Errorf("no moov box found")
	}
	if moov.size > maxMoovSize {
		return fmt.Errorf("moov box too large: %d bytes", moov.size)
	}

	moovBytes := make([]byte, moov.size)
	if _, err := f.ReadAt(moovBytes, moov.offset); err != nil {
		return err
	}

	newUdta := buildUdta(meta)

	// Skip the rewrite when the file already carries exactly this metadata
	if bytes.Contains(moovBytes, newUdta) {
		return nil
	}

	// Rebuild moov: keep all children except udta, then append the new udta
	children, err := readMP4Boxes(bytes.NewReader(moovBytes), moov.headerLen, moov.size)
	if err != nil {
		return err
	}

	var payload []byte
	for _, child := range children {
		if child.boxType == "udta" {
			continue
		}
		payload = append(payload, moovBytes[child.offset:child.offset+child.size]...)
	}
	payload = append(payload, newUdta...)
	newMoov := writeBox("moov", payload)

	delta := int64(len(newMoov)) - moov.size
	oldMoovEnd := moov.offset + moov.size
	if delta != 0 {
		// Everything after the old moov shifts by delta, so chunk offsets
		// pointing at or beyond that position must be adjusted.
		if err := patchChunkOffsets(newMoov[8:], oldMoovEnd, delta); err != nil {
			return err
		}
	}

	tmpPath := path + ".meta.tmp"
	// #nosec G304 - Temporary file next to the media file being tagged
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	// Bytes before moov are unchanged
	if _, err := io.Copy(tmp, io.NewSectionReader(f, 0, moov.offset)); err != nil {
		return err
	}
	if _, err := tmp.Write(newMoov); err != nil {
		return err
	}
	// Bytes after the old moov are unchanged, just shifted
	if _, err := io.Copy(tmp, io.NewSectionReader(f, oldMoovEnd, fi.Size()-oldMoovEnd)); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// Preserve the modification time; the downloader uses it for
	// date-based disk cleanup.
	if err := os.Chtimes(tmpPath, fi.ModTime(), fi.ModTime()); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
