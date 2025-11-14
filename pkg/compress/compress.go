package compress

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/ulikunitz/xz"
)

// Compress compresses a byte slice using the specified format.
// Supported formats are "gz" and "xz". If an unsupported format is provided,
// the original data is returned unmodified.
//
// Example:
//
//	compressedData, err := compress.Compress([]byte("hello world"), "gz")
//	if err != nil {
//		// handle error
//	}
//	// compressedData now holds the gzipped version of "hello world"
func Compress(data []byte, format string) ([]byte, error) {
	var buf bytes.Buffer
	var writer io.WriteCloser
	var err error

	switch format {
	case "gz":
		writer = gzip.NewWriter(&buf)
	case "xz":
		writer, err = xz.NewWriter(&buf)
		if err != nil {
			return nil, err
		}
	default:
		return data, nil
	}

	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Decompress decompresses a byte slice, automatically detecting the compression
// format (gz or xz) by inspecting the header magic bytes. If the data is not
// compressed in a recognized format, it is returned unmodified.
//
// Example:
//
//	decompressedData, err := compress.Decompress(compressedData)
//	if err != nil {
//		// handle error
//	}
//	// decompressedData now holds the original uncompressed data
func Decompress(data []byte) ([]byte, error) {
	// Check for gzip header
	if len(data) > 2 && data[0] == 0x1f && data[1] == 0x8b {
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	}

	// Check for xz header
	if len(data) > 6 && data[0] == 0xfd && data[1] == '7' && data[2] == 'z' && data[3] == 'X' && data[4] == 'Z' && data[5] == 0x00 {
		reader, err := xz.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return io.ReadAll(reader)
	}

	return data, nil
}
