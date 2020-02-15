package vtu

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type DataArray interface {
	Append(*DArray)
	Ints(name string, n int, data []int)
	Floats(name string, n int, data []float64)
}

// convertor to specific format
type Arrayer interface {
	Ints(name string, n int, data []int) *DArray
	Floats(name string, n int, data []float64) *DArray
}

// produce a specific array matching the format
// this probably needs to be attached to the header itself
func createArray(format string, fieldData bool, compressor Compressor) DataArray {

	var c Compressor
	if compressor != nil {
		c = compressor
	} else {
		c = &NoCompressor{}
	}

	switch format {
	case ascii:
		return &Array{Arrayer: Asciier{},
			fieldData: fieldData,
		}
	case FormatBinary:
		return &Array{Arrayer: Base64er{Compressor: c},
			fieldData: fieldData,
		}
	default:
		panic("not sure what data array to add")
	}
}

// todo change these to provide the full header? Or just attach this routine
// to the header itself, that would prevent passing all this stuff around
func NewArray(format string, compressor Compressor) DataArray {
	return createArray(format, false, compressor)
}

func NewFieldArray(format string, compressor Compressor) DataArray {
	return createArray(format, true, compressor)
}

// as we use the DataArray interface now, the Data []*DArray could be anything
// that could be something with string data, but also something with raw data?
type Array struct {
	XMLName   xml.Name
	Data      []*DArray
	Arrayer   Arrayer `xml:"-"`
	fieldData bool    `xml:"-"` // store as global field data
}

func (a *Array) Append(da *DArray) {

	if a.fieldData {
		// requires "NumberOfTuples" rather then "Components"
		da.NumberOfTuples = da.NumberOfComponents
		da.NumberOfComponents = 0
	}

	a.Data = append(a.Data, da)
}

func (a *Array) Ints(name string, n int, data []int) {
	a.Append(a.Arrayer.Ints(name, n, data))
}

func (a *Array) Floats(name string, n int, data []float64) {
	a.Append(a.Arrayer.Floats(name, n, data))
}

type Asciier struct{}

func (a Asciier) Ints(name string, n int, data []int) *DArray {
	d := intToString(data, " ")
	return NewDArray("UInt32", name, ascii, n, d)
}

func (a Asciier) Floats(name string, n int, data []float64) *DArray {
	d := floatToString(data, " ")
	return NewDArray("Float64", name, ascii, n, d)
}

type Base64er struct {
	Compressor Compressor
	appending  bool
}

func (b Base64er) Ints(name string, n int, data []int) *DArray {
	var buf bytes.Buffer

	// size header as int32
	binary.Write(&buf, binary.LittleEndian, int32(len(data)*4))

	// convert data to bytes
	for _, v := range data {
		err := binary.Write(&buf, binary.LittleEndian, int32(v))
		if err != nil {
			panic("error")
		}
	}

	// compress
	//b.Compressor.Compress(&buf, &buf)

	// encode
	d := base64.StdEncoding.EncodeToString(buf.Bytes())
	return NewDArray("UInt32", name, FormatBinary, n, d)
}

func (b Base64er) Floats(name string, n int, data []float64) *DArray {

	var bdat bytes.Buffer

	for _, v := range data {
		err := binary.Write(&bdat, binary.LittleEndian, v)
		if err != nil {
			panic("error payload binary")
		}
	}

	payload := Payload{data: bdat.Bytes()}
	b.Compressor.Compress(&payload)
	d := b.Encode(&payload)
	return NewDArray("Float64", name, FormatBinary, n, d)
}

// Encode encodes the payload to base64.
func (b *Base64er) Encode(p *Payload) string {
	var d string

	// when compressed, header and data need to be encoded seperately
	if p.compressed {
		d += base64.StdEncoding.EncodeToString(p.Header())
		d += base64.StdEncoding.EncodeToString(p.data)
	} else {
		d += base64.StdEncoding.EncodeToString(append(p.Header(), p.data...))
	}
	return d
}

type Binaryer struct {
	//Compressor Compressor
	appending bool
}

func floatToString(data []float64, sep string) string {
	if len(data) == 0 {
		return ""
	}

	s := make([]string, len(data))
	for i, d := range data {
		s[i] = fmt.Sprintf("%f", d)
	}

	return strings.Join(s, sep)
}

func intToString(data []int, sep string) string {
	if len(data) == 0 {
		return ""
	}

	s := make([]string, len(data))
	for i, d := range data {
		s[i] = strconv.Itoa(d)
	}

	return strings.Join(s, sep)
}

// not sure if i like this... maybe store just as ints?
func stringToInts(s string) []int {
	str := strings.Split(s, " ")
	ints := make([]int, len(str), len(str))
	for i := 0; i < len(str); i++ {
		f, err := strconv.ParseInt(str[i], 10, 32)
		if err != nil {
			panic(fmt.Sprintf("%v", err))
		}
		ints[i] = int(f)
	}
	return ints
}
