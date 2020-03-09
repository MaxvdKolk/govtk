package vtu

import (
	"bytes"
	"encoding/binary"
	"testing"
)

type pair struct {
	val []float64
	str []byte
}

// Verify binarise creates the expected bytes of some input data.
// todo: test more inputs?
func TestBinarise(t *testing.T) {
	var pairs = []pair{
		pair{
			val: []float64{1},
			str: []byte("1.000000"),
		},
		pair{ // print -, dont print +
			val: []float64{-1, +1},
			str: []byte("-1.000000 1.000000"),
		},
		pair{ // space before -
			val: []float64{1, -1},
			str: []byte("1.000000 -1.000000"),
		},
		pair{
			val: []float64{1, 2, 3},
			str: []byte("1.000000 2.000000 3.000000"),
		},
		pair{
			val: make([]float64, 3),
			str: []byte("0.000000 0.000000 0.000000"),
		},
	}

	// ascii requires manual desired output
	t.Logf("Ascii encoder binarise")
	enc := Asciier{}
	for _, pair := range pairs {
		p := enc.binarise(pair.val)
		if !bytes.Equal(p.body.Bytes(), pair.str) {
			t.Errorf("Wrong binarised content for %v", pair.val)
		}
	}

	// base64 and binary just validate wrt a binary.writer
	encoders := []encoder{Base64er{}, Binaryer{}}
	for _, enc := range encoders {
		r := newPayload()
		for _, pair := range pairs {
			p := enc.binarise(pair.val)
			r.reset()
			err := binary.Write(r.body, binary.LittleEndian, pair.val)
			if err != nil {
				t.Error("Failed setting up buffers.")
			}

			if !bytes.Equal(p.body.Bytes(), r.body.Bytes()) {
				t.Errorf("Wrong binarised content for enc: %T, data: %v",
					enc, pair.val)
			}
		}
	}
}
