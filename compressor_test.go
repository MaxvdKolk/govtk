package vtu

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestZlibHeader(t *testing.T) {
	p := NewPayload()
	c := compress(p)

	// after compression header should contain 4x int32 as bytes
	if c.head.Len() != 4*4 {
		t.Errorf("Wrong length, exp: %v, got %v", 16, c.head.Len())
	}

	// payload recognise compression
	if !c.compressed() {
		t.Errorf("Does not detect compressed header.")
	}
}

func TestZlibCompressDecompress(t *testing.T) {

	ints := [][]int{
		[]int{},
		[]int{1},
		[]int{1, 2},
		[]int{1, 2, 3}}

	p := NewPayload()
	r := NewPayload()

	for _, vals := range ints {
		p.reset()
		r.reset()

		if p.body.Len() > 0 || p.head.Len() > 0 {
			t.Errorf("Failed to reset payload.")
		}

		for _, v := range vals {
			wr := io.MultiWriter(p.body, r.body)
			err := binary.Write(wr, binary.LittleEndian, int32(v))
			if err != nil {
				t.Error("Failed setting up buffers.")
			}
		}
		p.setHeader()
		r.setHeader()

		// compress and decompress
		d := decompress(compress(p))

		if d.head.Len() == 0 {
			t.Errorf("Empty header after decompressing.")
		}

		if d.head.Len() != r.head.Len() {
			t.Errorf("Unequal header length: exp %v, got %v",
				d.head.Len(), r.head.Len())
		}

		if r.body.Len() > 0 && d.body.Len() == 0 {
			t.Errorf("Empty body after decompressing.")
		}

		if d.body.Len() != r.body.Len() {
			t.Errorf("Unequal body length: exp %v, got %v",
				d.head.Len(), r.head.Len())
		}

		if !bytes.Equal(r.body.Bytes(), d.body.Bytes()) {
			t.Errorf("Decompressed data is not the same")
		}
	}
}
