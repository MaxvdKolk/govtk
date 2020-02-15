package vtu

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
)

// apply compression to bytes
type Compressor interface {
	Compress(*Payload)
}

type NoCompressor struct{}

func (nc *NoCompressor) Compress(p *Payload) {
	p.compressed = false
	p.blocksize += int32(len(p.data))
}

type Zlib struct {
	level int
}

func (z *Zlib) Compress(p *Payload) {

	p.blocks += 1
	p.blocksize += int32(len(p.data))
	p.lastblocksize = int32(len(p.data))

	var cb bytes.Buffer
	writer, err := zlib.NewWriterLevel(&cb, zlib.DefaultCompression)
	if err != nil {
		panic("problem zlib compression")
	}
	writer.Write(p.data)
	writer.Close()

	p.compressed = true
	p.compressedblocks = append(p.compressedblocks, int32(len(cb.Bytes())))

	p.data = cb.Bytes()
}

// representation of data structure -> on construction receives an encoder
// for base64 this is the base64 encoder, while for binary this is just empty
//
// for appended data we need to append to this single payload continously?
type Payload struct {
	compressed       bool
	blocks           int32
	blocksize        int32
	lastblocksize    int32
	compressedblocks []int32
	data             []byte
	//    Compressor Compressor
	//    Encoder
}

// return the header: if compressed, return something else
func (p *Payload) Header() []byte {
	var header bytes.Buffer

	if p.compressed {

		// just a single block for now
		err := binary.Write(&header, binary.LittleEndian, int32(p.blocks))
		if err != nil {
			panic("error")
		}

		err = binary.Write(&header, binary.LittleEndian, int32(p.blocksize))
		if err != nil {
			panic("error")
		}

		err = binary.Write(&header, binary.LittleEndian, int32(p.lastblocksize))
		if err != nil {
			panic("error")
		}

		// todo modify
		err = binary.Write(&header, binary.LittleEndian, int32(p.compressedblocks[0]))
		if err != nil {
			panic("error")
		}

		return header.Bytes()

	} else {
		err := binary.Write(&header, binary.LittleEndian, int32(p.blocksize))
		if err != nil {
			panic("error")
		}
		return header.Bytes()
	}

}
