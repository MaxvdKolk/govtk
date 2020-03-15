package govtk

import (
	"compress/zlib"
	"encoding/binary"
	"io"
)

// These constants are copied from the zlib package, which are in turn copied
// from the flat package, so that code that imports "vtu" does not also have
// to import "compress/zlib".
const (
	NoCompression      = zlib.NoCompression
	BestSpeed          = zlib.BestSpeed
	BestCompression    = zlib.BestCompression
	DefaultCompression = zlib.DefaultCompression
	HuffmanOnly        = zlib.HuffmanOnly
)

// The compressor interface requires the ability to compress and
// decompress a payload.
type compressor interface {
	compress(p *Payload) (*Payload, error)
	decompress(p *Payload) (*Payload, error)
}

// Satifies the compressor iterface, without applying any (de)compression.
type noCompression struct{}

// Compress returns the payload without any compression
func (nc noCompression) compress(p *Payload) (*Payload, error) {
	if p.head.Len() == 0 {
		// insert header if not set
		p.setHeader()
	}
	return p, nil
}

// Decompress returns the payload without any decompression
func (nc noCompression) decompress(p *Payload) (*Payload, error) {
	return p, nil
}

func (nc noCompression) String() string {
	return `Compressor: no compression`
}

// Satisfies the compressor interface using compress/zlib for (de)compression.
type zlibCompression struct {
	level int
}

// Compress returns a compressed copy of the provided payload and updates
// the payload's header.
func (z zlibCompression) compress(p *Payload) (*Payload, error) {
	c := newPayload()

	// zlib writer
	writer, err := zlib.NewWriterLevel(c.body, z.level)
	if err != nil {
		return nil, err
	}

	// compress data and capture uncompressed payload's size
	n, err := io.Copy(writer, p.body)
	if err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	// write the header
	header := []int32{1, int32(n), int32(n), int32(c.body.Len())}
	for _, val := range header {
		err := binary.Write(c.head, binary.LittleEndian, val)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Decompress returns a decompressed copy of the provided payload and updates
// the payload's header.
func (z zlibCompression) decompress(p *Payload) (*Payload, error) {
	d := newPayload()

	reader, err := zlib.NewReader(p.body)
	if err != nil {
		return nil, err
	}

	// decompress data
	_, err = io.Copy(d.body, reader)
	if err != nil {
		return nil, err
	}

	if err := reader.Close(); err != nil {
		return nil, err
	}

	d.setHeader()
	return d, nil
}

func (z zlibCompression) String() string {
	return `Compressor: compress/zlib`
}
