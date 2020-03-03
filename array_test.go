package vtu

import (
	"encoding/base64"
	"encoding/binary"
	"testing"
)

type pair struct {
	val []float64
	exp string
}

func TestFloats(t *testing.T) {
	pairs := []pair{
		pair{
			val: []float64{1},
			exp: string("1.000000"),
		},
		pair{ // print -, dont print +
			val: []float64{-1, +1},
			exp: string("-1.000000 1.000000"),
		},
		pair{ // space before -
			val: []float64{1, -1},
			exp: string("1.000000 -1.000000"),
		},
		pair{
			val: []float64{1, 2, 3},
			exp: string("1.000000 2.000000 3.000000"),
		},
		pair{
			val: make([]float64, 3),
			exp: string("0.000000 0.000000 0.000000"),
		},
	}

	testAsciiFloats(pairs, t)
	testBase64Floats(pairs, t)
}

func testAsciiFloats(pairs []pair, t *testing.T) {
	enc := Asciier{}
	for i := range pairs {
		pair := pairs[i]
		pl := enc.Floats(pair.val)

		if string(pl.data) != pair.exp {
			t.Logf("got: '%v', want: '%v'", string(pl.data), pair.exp)
			t.Fail()
		}

		if enc.String(pl) != pair.exp {
			t.Logf("payload conversion: '%v', want: '%v'", string(pl.data), pair.exp)
			t.Fail()
		}
	}
}

func testBase64Floats(pairs []pair, t *testing.T) {
	enc := Base64er{}
	cmp := NoCompressor{}

	for i := range pairs {
		pair := pairs[i]
		pl := enc.Floats(pair.val)

		cmp.Compress(pl)

		//t.Logf(string(pl.data))
		t.Logf("block size %v", pl.headerData()[0])

		encString := enc.String(pl)

		exp, err := base64.StdEncoding.DecodeString(encString)

		if err != nil {
			t.Logf("got error %v", err)
		}

		t.Logf("Encoded string: '%v'", encString)
		t.Logf("Expected size: '%v'", binary.LittleEndian.Uint32(exp[:4]))
		t.Logf("Decoded data: '%v'", string(exp[5:]))

		if string(exp) != pair.exp {
			t.Logf("got: '%v', want: '%v'", string(exp), pair.exp)
			t.Fail()
		}
	}
}
