package trix

import (
	"io"

	"github.com/Snider/Enchantrix/pkg/trix"
)

// ToRawTrix writes a raw Trix container: the JSON header, then the payload
// copied straight through from reader to writer.
//
// This is the hot-path counterpart to ToTrix. ToTrix packages a DataNode —
// it tars the node and optionally encrypts the tarball, which means the whole
// archive is resident and the stored bytes are not the caller's bytes. A
// consumer holding one large file (a State log, a model blob) wants neither:
// nothing is tarred, encrypted or compressed here, and the payload is never
// held in memory, so a multi-gigabyte container costs a copy buffer.
//
// Because the binary tail is stored verbatim it can later be mapped rather
// than read — see FromRawTrixHeader for the offset.
//
//	src, _ := os.Open("session.mvlog")
//	dst, _ := os.Create("session.kv")
//	n, err := trix.ToRawTrix(map[string]interface{}{"kind": "state-kv"}, "KVST", src, dst)
//
// It returns the total number of bytes written to w.
func ToRawTrix(header map[string]interface{}, magic string, payload io.Reader, w io.Writer) (int64, error) {
	return trix.EncodeStream(header, magic, payload, w)
}

// FromRawTrixHeader reads a raw Trix container's header and reports where its
// payload starts, without reading a single payload byte.
//
// The counterpart to FromTrix for containers written by ToRawTrix: rather
// than materialising a DataNode, it hands back the offset a caller needs to
// mmap, pread or section-read the binary tail directly.
//
//	info, err := trix.FromRawTrixHeader(f, "KVST")
//	section := io.NewSectionReader(f, info.PayloadOffset, info.PayloadBytes)
//
// info.PayloadBytes is best-effort — it is populated when the reader can
// report its own size (an *os.File or a *bytes.Reader), and is 0 otherwise.
func FromRawTrixHeader(r io.ReaderAt, magic string) (trix.HeaderInfo, error) {
	return trix.ReadHeaderInfo(r, magic)
}
