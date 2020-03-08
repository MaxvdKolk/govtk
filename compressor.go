package vtu

import (
	"compress/zlib"
	"encoding/binary"
	"io"
	"log"
)

// The compressor interface requires the ability to compress and
// decompress a payload.
type compressor interface {
	compress(p *Payload) *Payload
	decompress(p *Payload) *Payload
}

// Satifies the compressor iterface, without applying any (de)compression.
type noCompression struct{}

// Compress returns the payload without any compression
func (nc noCompression) compress(p *Payload) *Payload {
	if p.head.Len() == 0 {
		// insert header if not set
		p.setHeader()
	}
	return p
}

// Decompress returns the payload without any decompression
func (nc noCompression) decompress(p *Payload) *Payload {
	return p
}

func (nc noCompression) String() string {
	return `Compressor: no compression`
}

// Satisfies the compressor interface using compress/zlib for (de)compression.
type zlibCompression struct{}

// Compress returns a compressed copy of the provided payload and updates
// the payload's header.
func (z zlibCompression) compress(p *Payload) *Payload {
	c := newPayload()

	// zlib writer
	writer, err := zlib.NewWriterLevel(c.body, zlib.DefaultCompression)
	if err != nil {
		log.Fatal(err)
	}

	// compress data and capture uncompressed payload's size
	n, err := io.Copy(writer, p.body)
	if err != nil {
		log.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		log.Fatal(err)
	}

	// write the header
	header := []int32{1, int32(n), int32(n), int32(c.body.Len())}
	for _, val := range header {
		err := binary.Write(c.head, binary.LittleEndian, val)
		if err != nil {
			log.Fatal(err)
		}
	}

	return c
}

// Decompress returns a decompressed copy of the provided payload and updates
// the payload's header.
func (z zlibCompression) decompress(p *Payload) *Payload {
	d := newPayload()

	reader, err := zlib.NewReader(p.body)
	if err != nil {
		log.Fatal(err)
	}

	// decompress data
	_, err = io.Copy(d.body, reader)
	if err != nil {
		log.Fatal(err)
	}

	if err := reader.Close(); err != nil {
		log.Fatal(err)
	}

	d.setHeader()
	return d
}

func (zc zlibCompression) String() string {
	return `Compressor: compress/zlib`
}
