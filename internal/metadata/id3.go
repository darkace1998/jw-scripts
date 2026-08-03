package metadata

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

const id3HeaderSize = 10

// buildID3Tag serializes an ID3v2.4 tag for the given metadata. The output
// is deterministic so re-embedding identical metadata can be detected and
// skipped.
func buildID3Tag(meta *FileMetadata) []byte {
	var frames bytes.Buffer

	writeText := func(id, value string) {
		if value == "" {
			return
		}
		// Text frame payload: encoding byte (0x03 = UTF-8) + text
		payload := append([]byte{0x03}, []byte(value)...)
		frames.WriteString(id)
		frames.Write(synchsafe(len(payload)))
		frames.Write([]byte{0, 0}) // frame flags
		frames.Write(payload)
	}

	writeText("TIT2", meta.Title)
	writeText("TALB", meta.album())
	writeText("TPE1", "jw.org")
	writeText("TDRC", meta.dateTag())

	if meta.URL != "" {
		// WOAF (official audio file webpage) is a URL frame: no encoding byte
		payload := []byte(meta.URL)
		frames.WriteString("WOAF")
		frames.Write(synchsafe(len(payload)))
		frames.Write([]byte{0, 0})
		frames.Write(payload)
	}

	tag := make([]byte, 0, id3HeaderSize+frames.Len())
	tag = append(tag, 'I', 'D', '3', 4, 0, 0) // ID3v2.4.0, no flags
	tag = append(tag, synchsafe(frames.Len())...)
	tag = append(tag, frames.Bytes()...)
	return tag
}

// synchsafe encodes n as a 4-byte synchsafe integer (7 bits per byte).
func synchsafe(n int) []byte {
	return []byte{
		byte(n >> 21 & 0x7f),
		byte(n >> 14 & 0x7f),
		byte(n >> 7 & 0x7f),
		byte(n & 0x7f),
	}
}

// existingID3TagSize returns the total size in bytes of the ID3v2 tag at the
// start of the file, or 0 when the file has no ID3v2 tag.
func existingID3TagSize(f *os.File) (int64, error) {
	header := make([]byte, id3HeaderSize)
	n, err := io.ReadFull(f, header)
	if err != nil {
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			return 0, nil
		}
		return 0, err
	}
	if n < id3HeaderSize || !bytes.Equal(header[:3], []byte("ID3")) {
		return 0, nil
	}

	size := int64(header[6]&0x7f)<<21 | int64(header[7]&0x7f)<<14 |
		int64(header[8]&0x7f)<<7 | int64(header[9]&0x7f)
	total := id3HeaderSize + size
	if header[5]&0x10 != 0 {
		// Tag has a footer
		total += id3HeaderSize
	}
	return total, nil
}

// embedMP3 writes an ID3v2.4 tag at the start of the MP3 file, replacing any
// existing ID3v2 tag. The audio data is left untouched.
func embedMP3(path string, meta *FileMetadata) error {
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

	oldTagSize, err := existingID3TagSize(f)
	if err != nil {
		return err
	}
	if oldTagSize > fi.Size() {
		return fmt.Errorf("corrupt ID3 tag: tag size %d exceeds file size %d", oldTagSize, fi.Size())
	}

	newTag := buildID3Tag(meta)

	// Skip the rewrite when the file already carries exactly this tag
	if oldTagSize == int64(len(newTag)) {
		existing := make([]byte, len(newTag))
		if _, err := f.ReadAt(existing, 0); err == nil && bytes.Equal(existing, newTag) {
			return nil
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

	if _, err := tmp.Write(newTag); err != nil {
		return err
	}
	if _, err := f.Seek(oldTagSize, io.SeekStart); err != nil {
		return err
	}
	if _, err := io.Copy(tmp, f); err != nil {
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
