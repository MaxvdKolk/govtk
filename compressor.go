package vtu

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
)

// interface to compress a payload
// todo the compressor does not require the full *Payload...
// it should just compress the bytes. The payload is able to create
// its own header
type Compressor interface {
	Compress(*Payload)
}

type NoCompressor struct{}

func (nc *NoCompressor) Compress(p *Payload) {
	p.compressed = false
	//p.blockSize += int32(len(p.data))
}

type Zlib struct {
	level int
}

func (z *Zlib) Compress(p *Payload) {
	// when compressed the header of the payload requires
	// blocks, blocksize, lastblocksize, compressedblocksizes,  (add links)
	// however, we compress each block individually, this seems to simplify
	// as the header always is followed by a single block?
	//
	// header | data | header | data
	//
	// rather than header | data | data | data ?

	// todo refactor?
	var cb bytes.Buffer
	writer, err := zlib.NewWriterLevel(&cb, zlib.DefaultCompression)
	if err != nil {
		panic("problem zlib compression")
	}
	writer.Write(p.data.Bytes())
	writer.Close()

	p.compressed = true
	p.compressedBlockSize = int32(len(cb.Bytes()))

	// replace original bytes by compressed bytes
	p.data = cb
}

// for appended data we need to append to this single payload continously?
type Payload struct {
	compressed          bool
	blocks              int32 // TODO currently unsed, blocks always 1
	blockSize           int32
	lastBlockSize       int32 // TODO currently unused, blockSize always last
	compressedBlockSize int32
	data                bytes.Buffer
}

func (p *Payload) headerData() []int32 {
	if p.compressed {
		return []int32{1, p.blockSize, p.blockSize, p.compressedBlockSize}
	} else {
		return []int32{p.blockSize}
	}
}

// return the header: if compressed, return something else
func (p *Payload) Header() []byte {

	var header bytes.Buffer

	for _, v := range p.headerData() {
		err := binary.Write(&header, binary.LittleEndian, v)
		if err != nil {
			panic("error")
		}
	}

	return header.Bytes()
}
