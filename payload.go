package govtk

import (
	"bytes"
	"encoding/binary"
)

// Payload contains the data for a single dataarray in the vtk format.
// The data is represented as a byte slice for the header and the body.
//
// For uncompressed payloads the header is a single int32.
// For compressed payloads the header is a set of by four int32:
// 	 number of blocks (currently always == 1)
// 	 bytes current body
// 	 bytes previous body (when num blocks == 1: equal to current block)
// 	 bytes compressed block
type Payload struct {
	head *bytes.Buffer
	body *bytes.Buffer
}

// NewPayload returns a pointer to a payload with empty buffers.
func newPayload() *Payload {
	return &Payload{head: new(bytes.Buffer), body: new(bytes.Buffer)}
}

// NewPayloadFromData returns a pointer to payload constructed for the
// data interface{}. The header is set after filling, no matter if the
// write operation failed. It is up to the caller to verify err == nil.
func newPayloadFromData(data interface{}) (*Payload, error) {
	p := newPayload()

	switch v := data.(type) {
	case []int:
		for _, x := range v {
			err := binary.Write(p.body, binary.LittleEndian, int32(x))
			if err != nil {
				return nil, err
			}
		}
	default:
		err := binary.Write(p.body, binary.LittleEndian, data)
		if err != nil {
			return nil, err
		}
	}

	p.setHeader()
	return p, nil
}

// setHeader sets the header buffer with the data's length in bytes.
func (p *Payload) setHeader() error {
	p.head.Reset()
	return binary.Write(p.head, binary.LittleEndian, int32(p.body.Len()))
}

// compressed returns true if the payload has been compressed.
func (p *Payload) isCompressed() bool {
	if p.head.Len() == 4 {
		// a single int32 header implies no compression
		return false
	}
	return true
}

// Reset resets both byte slices of the payload.
func (p *Payload) reset() {
	p.head.Reset()
	p.body.Reset()
}
