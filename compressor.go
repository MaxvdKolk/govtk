package vtu

import (
	"compress/zlib"
	"encoding/binary"
	"io"
	"log"
)

// interface to compress a payload
// todo the compressor does not require the full *Payload...
// it should just compress the bytes. The payload is able to create
// its own header
// todo: add interface
/*

type compressor interface {
	compress(p *Payload) *Payload
	decompress(p *Payload) *Payload
}

*/

type Compressor func(p *Payload) *Payload

// noCompress returns the payload without any compression
func noCompress(p *Payload) *Payload {
	return p
}

// Compress returns a compressed copy of the provided payload and updates
// the payload's header.
//
// For compressed payloads the header is given by four int32 values
// - number of blocks present (currently always == 1)
// - number of bytes of current block
// - number of bytes previous block (currently equal to current block)
// - number of bytes compressed block
func compress(p *Payload) *Payload {
	c := NewPayload()

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
func decompress(p *Payload) *Payload {
	d := NewPayload()

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
