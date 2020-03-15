package vtk

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestNewPayload(t *testing.T) {
	p := newPayload()
	if p.head.Len() > 0 {
		t.Errorf("New header is not empty, len: %v", p.head.Len())
	}

	if p.body.Len() > 0 {
		t.Errorf("New body is not empty, len: %v", p.body.Len())
	}
}

func TestNewPayloadFromData(t *testing.T) {
	datas := []interface{}{
		int8(0), int16(0), int32(0), int64(0),
		int8(100), int16(100), int32(100), int64(100),
		[]int8{0, 1, 2, 3, 4}, []int16{0, 1, 2, 3, 4},
		[]int32{0, 1, 2, 3, 4}, []int64{0, 1, 2, 3, 4},
		float32(0), float64(0), float32(1.0), float64(1.0),
		[]float32{0.0, 0.1, 0.2, 0.3, 0.4},
		[]float64{0.0, 0.1, 0.2, 0.3, 0.4},
	}

	tmp := new(bytes.Buffer)
	for _, data := range datas {

		p, _ := newPayloadFromData(data)
		if p.head.Len() != 4 {
			t.Errorf("Wrong header length: exp: %v, got: %v",
				4, p.head.Len())
		}

		tmp.Reset()
		err := binary.Write(tmp, binary.LittleEndian, data)
		if err != nil {
			t.Errorf("Cannot setup test buffer: %v", err)
		}

		if p.body.Len() != tmp.Len() {
			t.Errorf("Wrong body length: exp: %v, got: %v",
				tmp.Len(), p.body.Len())
		}

		if !bytes.Equal(p.body.Bytes(), tmp.Bytes()) {
			t.Errorf("Wrong body content: exp %#v, got %#v",
				tmp.Bytes(), p.body.Bytes())
		}

		if p.isCompressed() {
			t.Errorf("Payload should not be compressed.")
		}
	}
}

func TestPayloadFromInvalidData(t *testing.T) {
	_, err := newPayloadFromData(string("-"))
	if err == nil {
		t.Errorf("Payload should return not nil for faulty input")
	}
}

func TestSetHeader(t *testing.T) {
	p := newPayload()

	p.setHeader()
	if p.head.Len() != 4 {
		t.Errorf("Int32 header not right length: %v", p.head.Len())
	}

}
