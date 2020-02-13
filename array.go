package vtu

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
)

type DataArray interface {
	Append(*DArray)
	Ints(name string, n int, data []int)
	Floats(name string, n int, data []float64)
}

// convertor to specific format
type Arrayer interface {
	Ints(name, format string, n int, data []int) *DArray
	Floats(name, format string, n int, data []float64) *DArray
}

// compressor for compression levels
type Compressor interface {
	Compress()
}

// produce a specific array matching the format
// this probably needs to be attached to the header itself
func NewArray(format string) DataArray {
	switch format {
	case ascii:
		return &Array{Arrayer: Asciier{}}
	case FormatBinary:
		return &Array{Arrayer: Base64er{}}
	default:
		panic("not sure what data array to add")
	}
}

// as we use the DataArray interface now, the Data []*DArray could be anything
// that could be something with string data, but also something with raw data?
type Array struct {
	XMLName xml.Name
	Data    []*DArray
	Arrayer Arrayer `xml:"-"` // do not convert to xml
}

func (a *Array) Append(da *DArray) {
	a.Data = append(a.Data, da)
}

func (a *Array) Ints(name string, n int, data []int) {
	a.Append(a.Arrayer.Ints(name, ascii, n, data))
}

func (a *Array) Floats(name string, n int, data []float64) {
	a.Append(a.Arrayer.Floats(name, ascii, n, data))
}

type Asciier struct{}

func (a Asciier) Ints(name, format string, n int, data []int) *DArray {
	d := intToString(data, " ")
	return NewDArray("UInt32", name, format, n, d)
}

func (a Asciier) Floats(name, format string, n int, data []float64) *DArray {
	d := floatToString(data, " ")
	return NewDArray("Float64", name, format, n, d)
}

type Base64er struct{}

func (b Base64er) Ints(name, format string, n int, data []int) *DArray {
	var buf bytes.Buffer

	// size header as int32
	binary.Write(&buf, binary.LittleEndian, int32(len(data)*4))

	// convert data to bytes
	for _, v := range data {
		err := binary.Write(&buf, binary.LittleEndian, v)
		if err != nil {
			panic("error")
		}
	}

	// compress
	// a.Compressor.Compress(&buf)

	// encode
	d := base64.StdEncoding.EncodeToString(buf.Bytes())
	return NewDArray("UInt32", name, format, n, d)
}

func (b Base64er) Floats(name, format string, n int, data []float64) *DArray {
	var buf bytes.Buffer

	// size header as int32
	binary.Write(&buf, binary.LittleEndian, int32(len(data)*8))

	// convert data to bytes
	for _, v := range data {
		err := binary.Write(&buf, binary.LittleEndian, v)
		if err != nil {
			panic("error")
		}
	}

	// compress
	// a.Compressor.Compress(&buf)

	// encode
	d := base64.StdEncoding.EncodeToString(buf.Bytes())
	return NewDArray("UInt32", name, format, n, d)
}
