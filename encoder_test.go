package govtk

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

type pair struct {
	val       []float64
	str       []byte
	hex       string
	b64, b64c []byte
}

// the test values are only for float64 now...
// probably better to extend to more elaborate examples
var pairs = []pair{
	pair{
		val:  []float64{1},
		str:  []byte("1.000000"),
		hex:  "000000000000f03f",
		b64:  []byte("CAAAAAAAAAAAAPA/"),
		b64c: []byte("AQAAAAgAAAAIAAAAEQAAAA==eJxiAIMP9oAAAAD//wInATA="),
	},
	pair{ // print -, dont print +
		val:  []float64{-1, +1},
		str:  []byte("-1.000000 1.000000"),
		hex:  "000000000000f0bf000000000000f03f",
		b64:  []byte("EAAAAAAAAAAAAPC/AAAAAAAA8D8="),
		b64c: []byte("AQAAABAAAAAQAAAAEwAAAA==eJxiAIMP+6G0PSAAAP//EkYC3w=="),
	},
	pair{ // space before -
		val:  []float64{1, -1},
		str:  []byte("1.000000 -1.000000"),
		hex:  "000000000000f03f000000000000f0bf",
		b64:  []byte("EAAAAAAAAAAAAPA/AAAAAAAA8L8="),
		b64c: []byte("AQAAABAAAAAQAAAAEwAAAA==eJxiAIMP9lB6PyAAAP//DkYC3w=="),
	},
	pair{
		val:  []float64{1, 2, 3},
		str:  []byte("1.000000 2.000000 3.000000"),
		hex:  "000000000000f03f00000000000000400000000000000840",
		b64:  []byte("GAAAAAAAAAAAAPA/AAAAAAAAAEAAAAAAAAAIQA=="),
		b64c: []byte("AQAAABgAAAAYAAAAGAAAAA==eJxiAIMP9hCawQFCcTgAAgAA//8XtwG4"),
	},
	pair{
		val:  make([]float64, 3),
		str:  []byte("0.000000 0.000000 0.000000"),
		hex:  "000000000000000000000000000000000000000000000000",
		b64:  []byte("GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="),
		b64c: []byte("AQAAABgAAAAYAAAADwAAAA==eJxiwAEAAQAA//8AGAAB"),
	},
}

// Ensure the content of byte slices are expected for each encoding format.
func TestBinarise(t *testing.T) {

	// ascii format
	enc := asciier{}
	buf := new(bytes.Buffer)
	for _, pair := range pairs {
		buf.Reset()
		n := int32(len(pair.str))
		err := binary.Write(buf, binary.LittleEndian, n)
		if err != nil {
			t.Fatalf("Cannot setup reference values %v.", err)
		}

		p := enc.binarise(pair.val)

		// header content
		got := p.head.Bytes()
		exp := buf.Bytes()
		if !bytes.Equal(got, exp) {
			t.Errorf("Wrong header: got: %x exp: %x", got, exp)
		}

		// body content
		got = p.body.Bytes()
		exp = pair.str
		if !bytes.Equal(got, exp) {
			t.Errorf("Wrong body: got: %x exp: %x", got, exp)
		}
	}

	// base64, binary format
	encoders := []encoder{base64er{}, binaryer{}}
	for _, enc := range encoders {
		for _, pair := range pairs {
			p := enc.binarise(pair.val)

			buf.Reset()
			n := int32(len(pair.val) * 8)
			err := binary.Write(buf, binary.LittleEndian, n)
			if err != nil {
				t.Fatalf("Cannot setup reference values %v.", err)
			}

			// header content
			got := p.head.Bytes()
			exp := buf.Bytes()
			if !bytes.Equal(got, exp) {
				t.Errorf("Wrong header: got %x exp %x", got, exp)
			}

			// body content
			decoded, err := hex.DecodeString(pair.hex)
			if err != nil {
				t.Fatalf("Cannot decode hex: %v", pair.hex)
			}
			exp = p.body.Bytes()
			if !bytes.Equal(decoded, exp) {
				t.Logf("hello %08x", p.body.Bytes())
				t.Errorf("Wrong header: got %x exp %x", decoded, exp)
			}
		}
	}
}

// Ensure ascii encoding only contains the payloads body
func TestEncodeAscii(t *testing.T) {
	enc := asciier{}
	for _, pair := range pairs {
		p := enc.binarise(pair.val)
		b, err := enc.encode(p)
		if err != nil {
			t.Errorf("Encoder error %v", err)
		}
		if !bytes.Equal(pair.str, b) {
			t.Errorf("%T provides wrongly encoded payload", enc)
		}
	}
}

// Ensure the binary encoding just equals the body and header
// This seems a bit trivial?
func TestEncodeBinary(t *testing.T) {
	enc := binaryer{}
	compressors := []compressor{noCompression{}, zlibCompression{}}

	for _, c := range compressors {
		for _, p := range pairs {
			pl, err := c.compress(enc.binarise(p.val))
			if err != nil {
				t.Errorf("Compress error %v", err)
			}
			got, err := enc.encode(pl)
			if err != nil {
				t.Errorf("Encoder error %v", err)
			}
			exp := append(pl.head.Bytes(), pl.body.Bytes()...)

			if !bytes.Equal(got, exp) {
				t.Errorf("Wrongly encoded payload: got: %x, exp: %x", got, exp)
			}
		}
	}
}

// Ensure we get expected values.
// Current tests are quite limited, i.e. only float64 slices.
func TestEncodeBase64(t *testing.T) {
	enc := base64er{}
	var c compressor

	c = noCompression{}
	for _, p := range pairs {
		pl, err := c.compress(enc.binarise(p.val))
		if err != nil {
			t.Errorf("Compress error %v", err)
		}

		got, err := enc.encode(pl)
		if err != nil {
			t.Errorf("Encoder error %v", err)
		}

		if !bytes.Equal(got, p.b64) {
			t.Errorf("Wrongly uncompressed base64 encoding: got: %v exp: %v",
				string(got), string(p.b64))
		}
	}

	c = zlibCompression{level: DefaultCompression}
	for _, p := range pairs {
		pl, err := c.compress(enc.binarise(p.val))
		if err != nil {
			t.Errorf("Compress error %v", err)
		}

		got, err := enc.encode(pl)
		if err != nil {
			t.Errorf("Encoder error %v", err)
		}

		if !bytes.Equal(got, p.b64c) {
			t.Errorf("Wrongly uncompressed base64 encoding: got: %v exp: %v",
				string(got), string(p.b64))
		}
	}
}
