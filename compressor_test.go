package vtk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"testing"
)

// assert no compression does not actually compress
func TestNoCompressedHeader(t *testing.T) {
	compressors := []compressor{noCompression{}}
	for _, compressor := range compressors {
		t.Run(fmt.Sprintf("%v", compressor), func(t *testing.T) {
			p := newPayload()

			c, err := compressor.compress(p)
			if err != nil {
				t.Errorf("Compress error %v", err)
			}

			p.setHeader()

			if c.head.Len() != 4 {
				t.Errorf("Wrong header length, exp %v, got %v",
					4, c.head.Len())
			}

			if c.isCompressed() {
				t.Errorf("Should not be compressed")
			}
		})
	}
}

// todo add test to verify if paraview is emtpy or not without a header
// results in problems

// verify compressors modify header to 4*int32
func TestCompressedHeader(t *testing.T) {

	compressors := []compressor{zlibCompression{}}

	for _, compressor := range compressors {
		t.Run(fmt.Sprintf("%v", compressor), func(t *testing.T) {

			p := newPayload()
			c, err := compressor.compress(p)
			if err != nil {
				t.Errorf("Compress error %v", err)
			}

			// after compression header should contain 4x int32 as bytes
			if c.head.Len() != 4*4 {
				t.Errorf("Wrong length, exp: %v, got %v", 16, c.head.Len())
			}

			// payload recognise compression
			if !c.isCompressed() {
				t.Errorf("Does not detect compressed header.")
			}
		})
	}

}

// TestCompressors asserts content of payload remains equal for the
// available structs satisfying the compressor interface.
func TestCompressors(t *testing.T) {
	ints := [][]int{
		[]int{},
		[]int{1},
		[]int{1, 2},
		[]int{1, 2, 3}}

	compressors := []compressor{noCompression{}, zlibCompression{}}

	for _, compressor := range compressors {
		t.Run(fmt.Sprintf("%v", compressor), func(t *testing.T) {
			testCompressDecompress(ints, compressor, t)
		})
	}
}

// TestCompressDecompress verifies payload after compressing, decompressing.
func testCompressDecompress(cases [][]int, c compressor, t *testing.T) {

	p := newPayload()
	r := newPayload()

	for _, vals := range cases {
		p.reset()
		r.reset()

		if p.body.Len() > 0 || p.head.Len() > 0 {
			t.Errorf("Failed to reset payload.")
		}

		// setup payload and reference payload
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
		p, err := c.compress(p)
		if err != nil {
			t.Errorf("Compress error %v", err)

		}
		d, err := c.decompress(p)
		if err != nil {
			t.Errorf("Decompress error %v", err)
		}

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
			t.Errorf("Unequal body content")
		}
	}
}
