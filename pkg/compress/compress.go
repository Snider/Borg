package compress

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/ulikunitz/xz"
)

// Compress compresses data using the specified format.
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

// Decompress decompresses data, detecting the format automatically.
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
