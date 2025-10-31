package trix

import (
	"archive/tar"
	"os"
)

type Cube struct {
	writer *tar.Writer
	file   *os.File
}

func NewCube(path string) (*Cube, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &Cube{
		writer: tar.NewWriter(file),
		file:   file,
	}, nil
}

func (c *Cube) AddFile(path string, content []byte) error {
	hdr := &tar.Header{
		Name: path,
		Mode: 0600,
		Size: int64(len(content)),
	}
	if err := c.writer.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := c.writer.Write(content); err != nil {
		return err
	}
	return nil
}

func (c *Cube) Close() error {
	if err := c.writer.Close(); err != nil {
		return err
	}
	return c.file.Close()
}

func Extract(path string) (*tar.Reader, *os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return tar.NewReader(file), file, nil
}

func AppendToCube(path string) (*Cube, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Cube{
		writer: tar.NewWriter(file),
		file:   file,
	}, nil
}
