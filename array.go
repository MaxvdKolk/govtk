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

// bit weird it has both functions to string and []byte while one is allowed?
type Encoder interface {
	Ints(data []int) *Payload
	Floats(data []float64) *Payload
	Format() string
	String(*Payload) string
	Raw(*Payload) []byte
}

type Appender interface {
	Append(Type, name string, n int, p *Payload, enc Encoder) *DArray
}

// as we use the DataArray interface now, the Data []*DArray could be anything
// that could be something with string data, but also something with raw data?
type Array struct {
	XMLName    xml.Name
	Data       []*DArray
	fieldData  bool       `xml:"-"` // store as global field data on true
	Appender   Appender   `xml:"-"`
	Encoder    Encoder    `xml:"-"`
	Compressor Compressor `xml:"-"`
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
	payload := a.Encoder.Ints(data)
	a.Compressor.Compress(payload)
	a.Append(a.Appender.Append("UInt32", name, n, payload, a.Encoder))
}

func (a *Array) Floats(name string, n int, data []float64) {
	payload := a.Encoder.Floats(data)
	a.Compressor.Compress(payload)
	a.Append(a.Appender.Append("Float64", name, n, payload, a.Encoder))
}

// todo could be modified by directly writing to a bytes.Buffer vs Floats/Ints?

type Asciier struct{}

func (a Asciier) Ints(data []int) *Payload {
	var buf bytes.Buffer
	for _, v := range data {

		// the string representation
		err := binary.Write(&buf, binary.LittleEndian, strconv.Itoa(v))
		if err != nil {
			panic("error")
		}

		// the separator
		err = binary.Write(&buf, binary.LittleEndian, " ")
		if err != nil {
			panic("error")
		}
	}

	return &Payload{data: buf.Bytes()}
}

func (a Asciier) Floats(data []float64) *Payload {

	var buf bytes.Buffer
	for _, v := range data {

		// need to cast the strings to byte first
		// todo check this statement
		tmp := []byte(strconv.FormatFloat(v, 'f', 6, 32))
		err := binary.Write(&buf, binary.LittleEndian, tmp)
		if err != nil {
			panic("error")
		}

		// the separator
		err = binary.Write(&buf, binary.LittleEndian, []byte(" "))
		if err != nil {
			panic("error")
		}
	}

	return &Payload{data: buf.Bytes()}
}

func (a Asciier) Format() string {
	return ascii
}

func (a Asciier) String(p *Payload) string {
	return string(p.data)
}

func (a Asciier) Raw(p *Payload) []byte {
	return []byte{}
}

type Base64er struct{}

func (b Base64er) Ints(data []int) *Payload {
	var buf bytes.Buffer
	for _, v := range data {
		err := binary.Write(&buf, binary.LittleEndian, int32(v))
		if err != nil {
			panic("error payload binary")
		}
	}
	return &Payload{data: buf.Bytes()}
}

func (b Base64er) Floats(data []float64) *Payload {
	var buf bytes.Buffer
	for _, v := range data {
		err := binary.Write(&buf, binary.LittleEndian, v)
		if err != nil {
			panic("error payload binary")
		}
	}

	return &Payload{data: buf.Bytes()}
}

// Encode encodes the payload to base64.
func (b *Base64er) Encode(p *Payload) string {

	var d string

	// compressed: combine header and data after encoding
	if p.compressed {
		d += base64.StdEncoding.EncodeToString(p.Header())
		d += base64.StdEncoding.EncodeToString(p.data)
		return d
	}

	// uncompressed: combine header and data before encoding
	d += base64.StdEncoding.EncodeToString(append(p.Header(), p.data...))
	return d
}

func (b Base64er) String(p *Payload) string {
	return b.Encode(p)
}

func (b Base64er) Raw(p *Payload) []byte { return []byte{} }

func (b Base64er) Format() string { return FormatBinary }

type Binaryer struct{}

func (b Binaryer) Ints(data []int) *Payload {
	var buf bytes.Buffer
	for _, v := range data {
		err := binary.Write(&buf, binary.LittleEndian, int32(v))
		if err != nil {
			panic("error payload raw")
		}
	}

	return &Payload{data: buf.Bytes()}
}

func (b Binaryer) Floats(data []float64) *Payload {
	var buf bytes.Buffer
	for _, v := range data {
		err := binary.Write(&buf, binary.LittleEndian, v)
		if err != nil {
			panic("error payload raw")
		}
	}

	return &Payload{data: buf.Bytes()}
}

func (b Binaryer) Format() string {
	return FormatRaw
}

func (b Binaryer) String(p *Payload) string {
	return ""
}

func (b Binaryer) Raw(p *Payload) []byte {
	return append(p.Header(), p.data...)
}

type Inline struct{}

func (i *Inline) Append(Type, name string, n int, p *Payload, enc Encoder) *DArray {
	// how to ensure we do not perform inlining for binary data?
	return &DArray{
		XMLName:            xml.Name{Local: "DataArray"},
		Type:               Type,
		Name:               name,
		Format:             enc.Format(),
		NumberOfComponents: n,
		Data:               enc.String(p),
	}
}

type Appending struct {
	Array *DArray // pointer to the external appending array
}

func (a *Appending) Append(Type, name string, n int, p *Payload, enc Encoder) *DArray {

	// appended data need to start with a single underscore
	if len(a.Array.Data) == 0 {
		a.Array.Data += "_"
	}

	// store offset in DArray
	da := &DArray{
		XMLName:            xml.Name{Local: "DataArray"},
		Type:               Type,
		Name:               name,
		Format:             FormatAppended,
		NumberOfComponents: n,
		Offset:             strconv.Itoa(a.Array.offset),
	}

	// raw format expects raw []byte
	if enc.Format() == FormatRaw {
		d := enc.Raw(p)
		a.Array.RawData = append(a.Array.RawData, d...)
		a.Array.offset += len(d)
		return da
	}

	// other formats expect string
	d := enc.String(p)
	a.Array.Data += d
	a.Array.offset += len(d)
	return da
}

// not sure if i like this... maybe store just as ints?
// make changes to remove this function?
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
