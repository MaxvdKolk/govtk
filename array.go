package vtu

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type DataArray interface {
	Append(*DArray)
	Ints(name string, n int, data []int)
	Floats(name string, n int, data []float64)
}

// bit weird it has both functions to string and []byte while one is allowed?
// maybe have String also be viable to just stringify itself?
type Encoder interface {
	Ints(data []int) *Payload
	Floats(data []float64) *Payload
	Format() string // todo: this can then be gone
	String() string
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
	fieldData  bool     // store as global field data on true
	Appender   Appender `xml:"-"`
	Encoder    Encoder  `xml:"-"`
	compressor compressor
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
	payload = a.compressor.compress(payload)
	a.Append(a.Appender.Append("UInt32", name, n, payload, a.Encoder))
}

func (a *Array) Floats(name string, n int, data []float64) {
	payload := a.Encoder.Floats(data)
	payload = a.compressor.compress(payload)
	a.Append(a.Appender.Append("Float64", name, n, payload, a.Encoder))
}

// todo could be modified by directly writing to a bytes.Buffer vs Floats/Ints?

type Asciier struct{}

func (a Asciier) Ints(data []int) *Payload {
	p := &Payload{head: new(bytes.Buffer), body: new(bytes.Buffer)}
	for _, v := range data {

		// the string representation
		err := binary.Write(p.body, binary.LittleEndian, strconv.Itoa(v))
		if err != nil {
			log.Fatal(strconv.Itoa(v), err)
		}

		// the separator
		err = binary.Write(p.body, binary.LittleEndian, " ")
		if err != nil {
			panic("error")
		}
	}

	// set header
	if err := p.setHeader(); err != nil {
		log.Fatalf("could not set header %v", err)
	}

	return p
}

func (a Asciier) Floats(data []float64) *Payload {
	p := &Payload{head: new(bytes.Buffer), body: new(bytes.Buffer)}
	for i, v := range data {
		// need to cast the strings to byte first
		// todo check this statement
		tmp := []byte(strconv.FormatFloat(v, 'f', 6, 32))
		err := binary.Write(p.body, binary.LittleEndian, tmp)
		if err != nil {
			log.Fatal(err)
		}

		// insert separator
		if i < len(data)-1 {
			err = binary.Write(p.body, binary.LittleEndian, []byte(" "))
			if err != nil {
				panic("error")
			}
		}
	}

	if err := p.setHeader(); err != nil {
		log.Fatalf("could not write header %v", err)
	}
	return p
}

func (a Asciier) Format() string {
	return ascii
}

func (a Asciier) String() string {
	return "dummy"
}

func (a Asciier) Raw(p *Payload) []byte {
	return p.body.Bytes()
}

type Base64er struct{}

func (b Base64er) Ints(data []int) *Payload {
	p := &Payload{head: new(bytes.Buffer), body: new(bytes.Buffer)}
	for _, v := range data {
		err := binary.Write(p.body, binary.LittleEndian, int32(v))
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := p.setHeader(); err != nil {
		log.Fatalf("could not write header %v", err)
	}
	return p
}

func (b Base64er) Floats(data []float64) *Payload {
	p := &Payload{head: new(bytes.Buffer), body: new(bytes.Buffer)}
	for _, v := range data {
		err := binary.Write(p.body, binary.LittleEndian, v)
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := p.setHeader(); err != nil {
		log.Fatalf("could not write header %v", err)
	}
	return p
}

// keep string for pretty print; remove for encoding
func (b Base64er) String() string {
	return "dummy"
}

// Encode encodes the payload to base64.
func (b Base64er) Raw(p *Payload) []byte {
	enc := base64.StdEncoding
	data := new(bytes.Buffer)
	encoder := base64.NewEncoder(enc, data)

	// write header
	if _, err := encoder.Write(p.head.Bytes()); err != nil {
		log.Fatal(err)
	}

	if p.compressed() {
		// header and body should be compressed separately
		err := encoder.Close()
		if err != nil {
			log.Fatal(err)
		}
		encoder = base64.NewEncoder(enc, data)
	}

	// write body
	if _, err := encoder.Write(p.body.Bytes()); err != nil {
		log.Fatal(err)
	}
	encoder.Close()

	return data.Bytes()
}

func (b Base64er) Format() string { return FormatBinary }

type Binaryer struct{}

func (b Binaryer) Ints(data []int) *Payload {
	p := &Payload{head: new(bytes.Buffer), body: new(bytes.Buffer)}
	for _, v := range data {
		err := binary.Write(p.body, binary.LittleEndian, int32(v))
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := p.setHeader(); err != nil {
		log.Fatalf("could not write header %v", err)
	}
	return p
}

func (b Binaryer) Floats(data []float64) *Payload {
	p := &Payload{head: new(bytes.Buffer), body: new(bytes.Buffer)}
	for _, v := range data {
		err := binary.Write(p.body, binary.LittleEndian, v)
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := p.setHeader(); err != nil {
		log.Fatalf("could not write header %v", err)
	}
	return p
}

func (b Binaryer) Format() string {
	return FormatRaw
}

func (b Binaryer) String() string {
	return ""
}

func (b Binaryer) Raw(p *Payload) []byte {
	return append(p.head.Bytes(), p.body.Bytes()...)
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
		Data:               enc.Raw(p),
	}
}

type Appending struct {
	Array *DArray // pointer to the external appending array
}

func (a *Appending) Append(Type, name string, n int, p *Payload, enc Encoder) *DArray {

	// appended data need to start with a single underscore
	if len(a.Array.Data) == 0 {
		a.Array.Data = []byte("_")
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

	// append new data bytes
	d := enc.Raw(p)
	a.Array.Data = append(a.Array.Data, d...)
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
