package metadata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testMeta() *FileMetadata {
	return &FileMetadata{
		Title:        "Test Title",
		CategoryName: "Test Category",
		URL:          "https://example.com/file",
		Published:    "2023-11-14T22:13:20Z",
	}
}

func writeTestFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	// #nosec G304 - path is constrained to t.TempDir() in these tests
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func TestEmbedUnsupportedFormat(t *testing.T) {
	path := writeTestFile(t, "file.pdf", []byte("pdf"))
	if err := Embed(path, testMeta()); !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf("expected ErrUnsupportedFormat, got %v", err)
	}
}

// --- MP3 / ID3 ---

func TestEmbedMP3AddsTagAndPreservesAudio(t *testing.T) {
	audio := []byte("\xff\xfbFAKE-AUDIO-DATA")
	path := writeTestFile(t, "song.mp3", audio)
	modTime := time.Unix(1700000000, 0)
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}

	if err := Embed(path, testMeta()); err != nil {
		t.Fatalf("Embed() returned error: %v", err)
	}

	content := readTestFile(t, path)
	if !bytes.HasPrefix(content, []byte("ID3")) {
		t.Fatal("expected file to start with an ID3 tag")
	}
	if !bytes.Contains(content, []byte("Test Title")) {
		t.Error("expected embedded title in tag")
	}
	if !bytes.Contains(content, []byte("Test Category")) {
		t.Error("expected embedded album (category) in tag")
	}
	if !bytes.Contains(content, []byte("2023-11-14")) {
		t.Error("expected embedded date in tag")
	}
	if !bytes.Contains(content, []byte("https://example.com/file")) {
		t.Error("expected embedded URL in tag")
	}
	if !bytes.HasSuffix(content, audio) {
		t.Error("expected audio data to be preserved after the tag")
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.ModTime().Equal(modTime) {
		t.Errorf("expected modification time to be preserved, got %v", fi.ModTime())
	}
}

func TestEmbedMP3IsIdempotent(t *testing.T) {
	path := writeTestFile(t, "song.mp3", []byte("\xff\xfbAUDIO"))

	if err := Embed(path, testMeta()); err != nil {
		t.Fatalf("first Embed() returned error: %v", err)
	}
	first := readTestFile(t, path)

	if err := Embed(path, testMeta()); err != nil {
		t.Fatalf("second Embed() returned error: %v", err)
	}
	second := readTestFile(t, path)

	if !bytes.Equal(first, second) {
		t.Error("expected repeated embedding of identical metadata to leave the file unchanged")
	}
}

func TestEmbedMP3ReplacesExistingTag(t *testing.T) {
	audio := []byte("\xff\xfbAUDIO")
	path := writeTestFile(t, "song.mp3", audio)

	if err := Embed(path, testMeta()); err != nil {
		t.Fatalf("first Embed() returned error: %v", err)
	}

	updated := testMeta()
	updated.Title = "Updated Title"
	if err := Embed(path, updated); err != nil {
		t.Fatalf("second Embed() returned error: %v", err)
	}

	content := readTestFile(t, path)
	if bytes.Contains(content, []byte("Test Title")) {
		t.Error("expected old title to be replaced")
	}
	if !bytes.Contains(content, []byte("Updated Title")) {
		t.Error("expected new title in tag")
	}
	if bytes.Count(content, []byte("ID3")) != 1 {
		t.Error("expected exactly one ID3 tag")
	}
	if !bytes.HasSuffix(content, audio) {
		t.Error("expected audio data to be preserved")
	}
}

// --- MP4 ---

// buildTestMP4 constructs a minimal MP4 file: ftyp + moov (with a stco
// pointing into mdat) + mdat, in the given order. moovFirst controls whether
// moov comes before mdat (faststart layout) or after it.
func buildTestMP4(moovFirst bool, mdatPayload []byte) []byte {
	ftyp := writeBox("ftyp", []byte("isom\x00\x00\x02\x00isomiso2"))
	mdat := writeBox("mdat", mdatPayload)

	makeMoov := func(chunkOffsets []uint32) []byte {
		stcoPayload := make([]byte, 8+4*len(chunkOffsets))
		binary.BigEndian.PutUint32(stcoPayload[4:8], uint32(len(chunkOffsets))) // #nosec G115 - test data
		for i, off := range chunkOffsets {
			binary.BigEndian.PutUint32(stcoPayload[8+4*i:], off)
		}
		stco := writeBox("stco", stcoPayload)
		return writeBox("moov", writeBox("trak", writeBox("mdia", writeBox("minf", writeBox("stbl", stco)))))
	}

	// Compute where the mdat payload starts so stco can point at it
	// (probe with the same number of entries as the final moov)
	probeMoov := makeMoov([]uint32{0, 0})
	var mdatHeaderOff int
	if moovFirst {
		mdatHeaderOff = len(ftyp) + len(probeMoov)
	} else {
		mdatHeaderOff = len(ftyp)
	}
	dataOff := uint32(mdatHeaderOff + 8) // #nosec G115 - test data
	moov := makeMoov([]uint32{dataOff, dataOff + 4})

	var file []byte
	file = append(file, ftyp...)
	if moovFirst {
		file = append(file, moov...)
		file = append(file, mdat...)
	} else {
		file = append(file, mdat...)
		file = append(file, moov...)
	}
	return file
}

// readStcoOffsets extracts the chunk offsets from the (single) stco box.
func readStcoOffsets(t *testing.T, content []byte) []uint32 {
	t.Helper()
	idx := bytes.Index(content, []byte("stco"))
	if idx < 0 {
		t.Fatal("no stco box found")
	}
	payload := content[idx+4:]
	count := binary.BigEndian.Uint32(payload[4:8])
	offsets := make([]uint32, count)
	for i := range offsets {
		offsets[i] = binary.BigEndian.Uint32(payload[8+4*i:])
	}
	return offsets
}

func TestEmbedMP4FaststartLayoutShiftsChunkOffsets(t *testing.T) {
	mdatPayload := []byte("MEDIA-DATA")
	original := buildTestMP4(true, mdatPayload)
	path := writeTestFile(t, "video.mp4", original)

	if err := Embed(path, testMeta()); err != nil {
		t.Fatalf("Embed() returned error: %v", err)
	}

	content := readTestFile(t, path)

	for _, want := range [][]byte{
		[]byte("\xa9nam"), []byte("Test Title"),
		[]byte("\xa9alb"), []byte("Test Category"),
		[]byte("\xa9day"), []byte("2023-11-14"),
		[]byte("ilst"), []byte("udta"),
	} {
		if !bytes.Contains(content, want) {
			t.Errorf("expected embedded metadata %q in file", want)
		}
	}

	// The chunk offsets must still point at the mdat payload after moov grew
	offsets := readStcoOffsets(t, content)
	if len(offsets) != 2 {
		t.Fatalf("expected 2 chunk offsets, got %d", len(offsets))
	}
	if got := content[offsets[0] : offsets[0]+4]; !bytes.Equal(got, mdatPayload[:4]) {
		t.Errorf("first chunk offset points at %q, want %q", got, mdatPayload[:4])
	}
	if got := content[offsets[1] : offsets[1]+4]; !bytes.Equal(got, mdatPayload[4:8]) {
		t.Errorf("second chunk offset points at %q, want %q", got, mdatPayload[4:8])
	}

	grew := len(content) - len(original)
	if grew <= 0 {
		t.Error("expected file to grow after embedding")
	}
	origOffsets := readStcoOffsets(t, original)
	grew32 := uint32(grew) // #nosec G115 - small positive test value
	if offsets[0] != origOffsets[0]+grew32 {
		t.Errorf("expected chunk offset shifted by %d, got %d -> %d", grew, origOffsets[0], offsets[0])
	}
}

func TestEmbedMP4MoovAtEndKeepsChunkOffsets(t *testing.T) {
	mdatPayload := []byte("MEDIA-DATA")
	original := buildTestMP4(false, mdatPayload)
	path := writeTestFile(t, "video.mp4", original)

	if err := Embed(path, testMeta()); err != nil {
		t.Fatalf("Embed() returned error: %v", err)
	}

	content := readTestFile(t, path)

	origOffsets := readStcoOffsets(t, original)
	offsets := readStcoOffsets(t, content)
	for i := range offsets {
		if offsets[i] != origOffsets[i] {
			t.Errorf("chunk offset %d changed from %d to %d; mdat did not move", i, origOffsets[i], offsets[i])
		}
	}
	if got := content[offsets[0] : offsets[0]+4]; !bytes.Equal(got, mdatPayload[:4]) {
		t.Errorf("first chunk offset points at %q, want %q", got, mdatPayload[:4])
	}
}

func TestEmbedMP4IsIdempotent(t *testing.T) {
	path := writeTestFile(t, "video.mp4", buildTestMP4(true, []byte("MEDIA-DATA")))

	if err := Embed(path, testMeta()); err != nil {
		t.Fatalf("first Embed() returned error: %v", err)
	}
	first := readTestFile(t, path)

	if err := Embed(path, testMeta()); err != nil {
		t.Fatalf("second Embed() returned error: %v", err)
	}
	second := readTestFile(t, path)

	if !bytes.Equal(first, second) {
		t.Error("expected repeated embedding of identical metadata to leave the file unchanged")
	}
}

func TestEmbedMP4ReplacesExistingMetadata(t *testing.T) {
	path := writeTestFile(t, "video.mp4", buildTestMP4(true, []byte("MEDIA-DATA")))

	if err := Embed(path, testMeta()); err != nil {
		t.Fatalf("first Embed() returned error: %v", err)
	}

	updated := testMeta()
	updated.Title = "Updated Title"
	if err := Embed(path, updated); err != nil {
		t.Fatalf("second Embed() returned error: %v", err)
	}

	content := readTestFile(t, path)
	if bytes.Contains(content, []byte("Test Title")) {
		t.Error("expected old title to be replaced")
	}
	if !bytes.Contains(content, []byte("Updated Title")) {
		t.Error("expected new title in metadata")
	}
	if bytes.Count(content, []byte("udta")) != 1 {
		t.Error("expected exactly one udta box")
	}

	// Chunk offsets must still resolve to the media data
	offsets := readStcoOffsets(t, content)
	if got := content[offsets[0] : offsets[0]+4]; !bytes.Equal(got, []byte("MEDI")) {
		t.Errorf("first chunk offset points at %q after re-embed, want %q", got, "MEDI")
	}
}

func TestEmbedMP4RejectsCorruptFile(t *testing.T) {
	path := writeTestFile(t, "video.mp4", []byte("this is not an mp4 file"))
	if err := Embed(path, testMeta()); err == nil {
		t.Error("expected error for corrupt MP4 file")
	}
}
