package vtu

import (
	"bytes"
	"encoding/binary"
)

// A payload contains the data for a single dataarray in the vtk format.
// The data is represented as a byte slice for the header and the body.
type Payload struct {
	head *bytes.Buffer
	body *bytes.Buffer
}

func NewPayload() *Payload {
	return &Payload{head: new(bytes.Buffer), body: new(bytes.Buffer)}
}

// setHeader sets the header buffer with the data's length in bytes.
func (p *Payload) setHeader() error {
	p.head.Reset()
	err := binary.Write(p.head, binary.LittleEndian, int32(p.body.Len()))
	return err
}

// compressed returns true if the payload has been compressed.
func (p *Payload) compressed() bool {
	if p.head.Len() == 4 {
		// a single int32 header implies no compression
		return false
	}
	return true
}
