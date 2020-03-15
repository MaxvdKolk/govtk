package govtk

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"log"
)

// The encoder interface provides functionality to convert int or float data
// towards a payload. Additionally, the encoder encodes the payload's data.
type encoder interface {
	binarise(data interface{}) *Payload
	encode(*Payload) ([]byte, error)
	//decode([]byte) *Payload // todo
	format() string
}

// Asciier encodes the payload using the ascii format.
type Asciier struct{}

// Binarise creates a payload where the body is filled with the bytes of the
// string representation of the provided data. A space (" ") is inserted
// after each element of the data, except after the last.
func (a Asciier) binarise(data interface{}) *Payload {
	p := newPayload()

	// temp func to write []byte to buffer
	writeVal := func(buf io.Writer, data []byte) {
		err := binary.Write(buf, binary.LittleEndian, data)
		if err != nil {
			log.Fatal(err)
		}
	}

	// temp func to write a separator (" ") to buffer
	writeSep := func(buf io.Writer) {
		err := binary.Write(buf, binary.LittleEndian, []byte(" "))
		if err != nil {
			log.Fatal(err)
		}
	}

	// each type needs a string conversion before writing to buffer
	switch v := data.(type) {
	case []int:
		for i, x := range v {
			writeVal(p.body, []byte(fmt.Sprintf("%d", x)))
			if i < len(v)-1 {
				writeSep(p.body)
			}
		}
	case []float64:
		for i, x := range v {
			writeVal(p.body, []byte(fmt.Sprintf("%f", x)))
			if i < len(v)-1 {
				writeSep(p.body)
			}
		}
	default:
		log.Fatalf("No binarise case for %T in Asciier", v)
	}

	// set header
	if err := p.setHeader(); err != nil {
		log.Fatalf("could not set header %v", err)
	}
	return p
}

// Encode encodes the payload to []byte.
// For ascii format only the body of the payload is required.
func (a Asciier) encode(p *Payload) ([]byte, error) {
	return p.body.Bytes(), nil
}

func (a Asciier) format() string { return FormatAscii }

// Base64er encodes the payload using standard base64 encoding.
type Base64er struct{}

func (b Base64er) binarise(data interface{}) *Payload {
	p, err := newPayloadFromData(data)
	if err != nil {
		log.Fatalf("Cannot convert data to payload: %v", err)
	}
	return p
}

func (b Base64er) encode(p *Payload) ([]byte, error) {
	enc := base64.StdEncoding
	data := new(bytes.Buffer)
	encoder := base64.NewEncoder(enc, data)

	// write header
	if _, err := encoder.Write(p.head.Bytes()); err != nil {
		return nil, err
	}

	// compress header and body separately
	if p.isCompressed() {
		if err := encoder.Close(); err != nil {
			return nil, err
		}
		encoder = base64.NewEncoder(enc, data)
	}

	// write body
	if _, err := encoder.Write(p.body.Bytes()); err != nil {
		return nil, err
	}

	// close body
	if err := encoder.Close(); err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

func (b Base64er) format() string { return FormatBinary }

// Binaryer encodes the payload as raw binary data.
type Binaryer struct{}

func (b Binaryer) binarise(data interface{}) *Payload {
	p, err := newPayloadFromData(data)
	if err != nil {
		log.Fatalf("Cannot convert data to payload: %v", err)
	}
	return p
}

func (b Binaryer) encode(p *Payload) ([]byte, error) {
	return append(p.head.Bytes(), p.body.Bytes()...), nil
}

func (b Binaryer) format() string { return FormatRaw }
