package vtu

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type DataArray interface {
	Append(*DArray)
	add(name string, n int, data interface{})
}

type Appender interface {
	Append(Type, name string, n int, p *Payload, enc encoder) *DArray
}

// as we use the DataArray interface now, the Data []*DArray could be anything
// that could be something with string data, but also something with raw data?
type Array struct {
	XMLName    xml.Name
	Data       []*DArray
	fieldData  bool     // store as global field data on true
	appender   Appender `xml:"-"`
	encoder    encoder
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

func (a *Array) add(name string, n int, data interface{}) {
	payload := a.encoder.binarise(data)
	payload = a.compressor.compress(payload)

	var format string

	switch data.(type) {
	case []int:
		format = "UInt32"
	case []float64:
		format = "Float64"
	}

	a.Append(a.appender.Append(format, name, n, payload, a.encoder))
}

type Inline struct{}

func (i *Inline) Append(Type, name string, n int, p *Payload, enc encoder) *DArray {
	return &DArray{
		XMLName:            xml.Name{Local: "DataArray"},
		Type:               Type,
		Name:               name,
		Format:             enc.format(),
		NumberOfComponents: n,
		Data:               enc.encode(p),
	}
}

type Appending struct {
	Array *DArray // pointer to the external appending array
}

func (a *Appending) Append(Type, name string, n int, p *Payload, enc encoder) *DArray {

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
	d := enc.encode(p)
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
