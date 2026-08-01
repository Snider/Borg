package trix

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Snider/Borg/pkg/datanode"
)

// rawPatternByte varies with absolute position, so a chunk arriving out of
// order, duplicated or dropped changes the bytes instead of sliding past.
func rawPatternByte(offset int64) byte { return byte(offset*31 + 7) }

// rawPatternReader emits size bytes without ever holding more than one chunk.
type rawPatternReader struct {
	size int64
	off  int64
}

func (p *rawPatternReader) Read(b []byte) (int, error) {
	if p.off >= p.size {
		return 0, io.EOF
	}
	n := int64(len(b))
	if remaining := p.size - p.off; n > remaining {
		n = remaining
	}
	for i := int64(0); i < n; i++ {
		b[i] = rawPatternByte(p.off + i)
	}
	p.off += n
	return int(n), nil
}

// rawCountingWriter deliberately does not implement io.ReaderFrom, so io.Copy
// takes its generic path — the one the allocation claim is about.
type rawCountingWriter struct{ n int64 }

func (w *rawCountingWriter) Write(b []byte) (int, error) {
	w.n += int64(len(b))
	return len(b), nil
}

func TestToRawTrix(t *testing.T) {
	header := map[string]interface{}{"kind": "state-kv", "id": "session-1-fold-1"}
	payload := []byte("the binary tail, stored verbatim")

	var out bytes.Buffer
	n, err := ToRawTrix(header, "KVST", bytes.NewReader(payload), &out)
	if err != nil {
		t.Fatalf("ToRawTrix() error = %v", err)
	}
	if n != int64(out.Len()) {
		t.Errorf("ToRawTrix() returned %d, container is %d bytes", n, out.Len())
	}
	if got := string(out.Bytes()[:4]); got != "KVST" {
		t.Errorf("Expected magic 'KVST', got %q", got)
	}

	info, err := FromRawTrixHeader(bytes.NewReader(out.Bytes()), "KVST")
	if err != nil {
		t.Fatalf("FromRawTrixHeader() error = %v", err)
	}
	if info.Header["kind"] != "state-kv" || info.Header["id"] != "session-1-fold-1" {
		t.Errorf("header round trip lost fields: %v", info.Header)
	}
	if info.PayloadBytes != int64(len(payload)) {
		t.Errorf("PayloadBytes = %d, want %d", info.PayloadBytes, len(payload))
	}
	if got := out.Bytes()[info.PayloadOffset:]; !bytes.Equal(got, payload) {
		t.Errorf("payload at reported offset = %q, want %q", got, payload)
	}
}

// TestToRawTrix_NoTarNoEncryption is the reason this helper exists: unlike
// ToTrix it must not wrap the payload in a tar, encrypt it or compress it.
// The container tail has to be the caller's bytes so it can be mapped later.
func TestToRawTrix_NoTarNoEncryption(t *testing.T) {
	// Bytes that look like something a "helpful" transform would touch:
	// gzip magic, a JSON fragment, NULs, valid base64.
	payload := []byte{0x1F, 0x8B, 0x08, 0x00, '{', '"', 'a', '"', ':', '1', '}', 0x00, 'Z', 'm', '9', 'v'}

	var out bytes.Buffer
	if _, err := ToRawTrix(map[string]interface{}{"kind": "state-kv"}, "KVST", bytes.NewReader(payload), &out); err != nil {
		t.Fatalf("ToRawTrix() error = %v", err)
	}
	info, err := FromRawTrixHeader(bytes.NewReader(out.Bytes()), "KVST")
	if err != nil {
		t.Fatalf("FromRawTrixHeader() error = %v", err)
	}

	tail := out.Bytes()[info.PayloadOffset:]
	if !bytes.Equal(tail, payload) {
		t.Fatalf("payload was transformed: got %x, want %x", tail, payload)
	}

	// The header carries only what the caller supplied.
	for _, key := range []string{"checksum", "checksum_algo", "compression", "sigils", "encrypted", "encryption_algorithm"} {
		if _, present := info.Header[key]; present {
			t.Errorf("ToRawTrix added %q to the header", key)
		}
	}

	// And the tail is emphatically not a tarball, unlike ToTrix's.
	if _, err := datanode.FromTar(tail); err == nil {
		t.Error("raw payload parsed as a tar — ToRawTrix must not tar the payload")
	}

	// Same content through ToTrix is a tar, and therefore bigger and different.
	dn := datanode.New()
	dn.AddData("payload.bin", payload)
	tarred, err := ToTrix(dn, "")
	if err != nil {
		t.Fatalf("ToTrix() error = %v", err)
	}
	if len(tarred) <= out.Len() {
		t.Errorf("expected the tarred container (%d) to exceed the raw one (%d)", len(tarred), out.Len())
	}
}

// TestFromRawTrixHeader_File proves the offset addresses a real file and that
// reading the header leaves the caller's file position alone.
func TestFromRawTrixHeader_File(t *testing.T) {
	payload := []byte("state log bytes")
	path := filepath.Join(t.TempDir(), "session.kv")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	written, err := ToRawTrix(map[string]interface{}{"kind": "state-kv"}, "KVST", bytes.NewReader(payload), f)
	if err != nil {
		t.Fatalf("ToRawTrix() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	stat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if stat.Size() != written {
		t.Errorf("wrote %d bytes, file is %d", written, stat.Size())
	}

	in, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = in.Close() }()

	info, err := FromRawTrixHeader(in, "KVST")
	if err != nil {
		t.Fatalf("FromRawTrixHeader() error = %v", err)
	}
	if info.PayloadBytes != int64(len(payload)) {
		t.Errorf("PayloadBytes = %d, want %d", info.PayloadBytes, len(payload))
	}

	tail := make([]byte, info.PayloadBytes)
	if _, err := in.ReadAt(tail, info.PayloadOffset); err != nil {
		t.Fatalf("ReadAt() error = %v", err)
	}
	if !bytes.Equal(tail, payload) {
		t.Errorf("payload at offset %d = %q, want %q", info.PayloadOffset, tail, payload)
	}

	// The handle's own offset is untouched, so an interleaved read still
	// starts at the beginning of the container.
	first := make([]byte, 4)
	if _, err := in.Read(first); err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if string(first) != "KVST" {
		t.Errorf("FromRawTrixHeader moved the file offset: next read gave %q", first)
	}
}

func TestFromRawTrixHeader_Errors(t *testing.T) {
	var container bytes.Buffer
	if _, err := ToRawTrix(map[string]interface{}{"a": 1}, "KVST", bytes.NewReader([]byte("p")), &container); err != nil {
		t.Fatalf("ToRawTrix() error = %v", err)
	}

	if _, err := FromRawTrixHeader(bytes.NewReader(container.Bytes()), "TRIX"); err == nil {
		t.Error("expected a magic mismatch to fail")
	}
	if _, err := FromRawTrixHeader(bytes.NewReader([]byte("nope")), "KVST"); err == nil {
		t.Error("expected a truncated container to fail")
	}
	if _, err := ToRawTrix(nil, "TOOLONG", nil, &bytes.Buffer{}); err == nil {
		t.Error("expected a bad magic length to fail")
	}
}

// TestToRawTrix_BoundedAllocations is the receipt: a payload past the 64 MiB
// mark must cost the same as a small one. If ToRawTrix ever grew a buffering
// step, the 65 -> 130 MiB delta would move with it.
func TestToRawTrix_BoundedAllocations(t *testing.T) {
	if testing.Short() {
		t.Skip("large payload test skipped in -short mode")
	}

	const (
		size64    = 65 << 20
		size128   = 130 << 20
		tolerance = 1 << 20 // one io.Copy buffer plus GC noise
	)

	measure := func(size int64) uint64 {
		var (
			w      rawCountingWriter
			before runtime.MemStats
			after  runtime.MemStats
			err    error
		)
		runtime.GC()
		runtime.ReadMemStats(&before)
		_, err = ToRawTrix(map[string]interface{}{"kind": "state-kv"}, "KVST", &rawPatternReader{size: size}, &w)
		runtime.ReadMemStats(&after)
		if err != nil {
			t.Fatalf("ToRawTrix() error = %v", err)
		}
		if w.n-int64(size) <= 0 {
			t.Fatalf("wrote %d bytes for a %d byte payload", w.n, size)
		}
		return after.TotalAlloc - before.TotalAlloc
	}

	big := measure(size64)
	bigger := measure(size128)
	t.Logf("ToRawTrix allocated bytes: 65MiB=%d 130MiB=%d", big, bigger)

	if big > tolerance {
		t.Errorf("encoding 65 MiB allocated %d bytes — the payload is being buffered", big)
	}
	delta := bigger - big
	if bigger < big {
		delta = big - bigger
	}
	if delta > tolerance {
		t.Errorf("allocation tracks payload size: 65 MiB=%d, 130 MiB=%d", big, bigger)
	}
}

// TestToRawTrix_RoundTripLargePayload proves a >64 MiB payload survives the
// container byte for byte, with neither side holding it.
func TestToRawTrix_RoundTripLargePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("large payload test skipped in -short mode")
	}

	const size = 65 << 20
	path := filepath.Join(t.TempDir(), "large.kv")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	written, err := ToRawTrix(map[string]interface{}{"kind": "state-kv"}, "KVST", &rawPatternReader{size: size}, f)
	if err != nil {
		t.Fatalf("ToRawTrix() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	in, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = in.Close() }()

	info, err := FromRawTrixHeader(in, "KVST")
	if err != nil {
		t.Fatalf("FromRawTrixHeader() error = %v", err)
	}
	if info.PayloadBytes != size {
		t.Fatalf("PayloadBytes = %d, want %d", info.PayloadBytes, int64(size))
	}
	if info.PayloadOffset != written-size {
		t.Fatalf("PayloadOffset = %d, want %d", info.PayloadOffset, written-size)
	}

	// Compare in a fixed-size window so the check itself stays bounded.
	section := io.NewSectionReader(in, info.PayloadOffset, info.PayloadBytes)
	buf := make([]byte, 64<<10)
	var off int64
	for {
		n, err := section.Read(buf)
		for i := 0; i < n; i++ {
			if buf[i] != rawPatternByte(off+int64(i)) {
				t.Fatalf("payload differs from input at offset %d", off+int64(i))
			}
		}
		off += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
	}
	if off != size {
		t.Errorf("read %d payload bytes, want %d", off, int64(size))
	}
}
