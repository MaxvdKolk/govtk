package vtu

import (
	"encoding/xml"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// DataArray represent shte inner data containers of the VTK XML structure.
// The format allows for multiple of these DataArrays to be present, e.g. to
// represent PointData, CellData, etc. The DataArray might contain multiple
// fields, implemented by the DArray struct.
type DataArray struct {
	// Name of the XML element, e.g. PointData, CellData, etc.
	XMLName xml.Name

	// A collection of data sets within this XML element.
	Data []*DArray

	// appended holds a pointer to an external DArray. This allows us
	// to write appended data formats that do not store the actual data
	// inline of the DataArray XML element. However, these attach the
	// data to a single, external DArray. The []*DArray will only hold
	// an offset towards the starting point of its data within the
	// external, appended DArray.
	appended *DArray

	// fieldData is true when the to be stored data is intended as
	// fieldData, i.e. global data to the XML VTK format. This could hold
	// time steps or other generic data that is not represent at cells
	// or points.
	fieldData bool

	// Encoder holds an encoder interface, which encodes the provided
	// data towards Ascii, Binary, or Raw formats.
	encoder encoder

	// The compressor holds an compressor interface, which allows to
	// compress the provided data before writing to Binary or Raw formats.
	compressor compressor
}

// DArray represent the innermost DataArray element containing various \
// properties of the data, and the data itself.
type DArray struct {
	XMLName xml.Name
	Type    string `xml:"type,attr,omitempty"`
	Name    string `xml:"Name,attr,omitempty"`
	Format  string `xml:"format,attr,omitempty"`

	// DataArray typically requires to specifuy NumberOfComponents,
	// however, when writing fieldData (global data) the format requires
	// to specify NumberOfTuples instead.
	NumberOfComponents int `xml:"NumberOfComponents,attr,omitempty"`
	NumberOfTuples     int `xml:"NumberOfTuples,attr,omitempty"`

	// The actual data to be stored, always represent as a set of bytes
	Data []byte `xml:",innerxml"`

	// Encoding is only required for Raw values
	Encoding string `xml:"encoding,attr,omitempty"`

	// Offset holds a pointer to int, as we want to omit these values for
	// any DArray that does not require offset, while we do not want to
	// consider Offset = 0 as an empty value. Thus, by making this a
	// pointer, the xml encoding only considers it empty when equal to nil.
	Offset *int `xml:"offset,attr,omitempty"`
}

// Provides a new DArray with properties set except the data fields
func NewDArray(xmlName, dtype, name, format string) *DArray {
	return &DArray{
		XMLName: xml.Name{Local: xmlName},
		Type:    dtype,
		Name:    name,
		Format:  format,
	}
}

// dataType tries to extract the data type, e.g. uint32, float64, etc., from
// the emtpy interface.
func (da *DataArray) dataType(data interface{}) string {
	switch data.(type) {
	case []int:
		return "UInt32"
	case []float64:
		return "Float64"
	}

	// todo add err test
	return ""
}

// Add adds data to the data array. The data can be stored inline or
// appended to a single storage
func (da *DataArray) add(name string, n int, data interface{}) {
	// encode
	payload := da.encoder.binarise(data)

	// compress
	payload, err := da.compressor.compress(payload)
	if err != nil {
		log.Fatal(err)
	}

	bytes := da.encoder.encode(payload) // error check here

	// add err check
	dtype := da.dataType(data)
	format := da.encoder.format()

	// get a new data array
	arr := NewDArray("DataArray", dtype, name, format)

	// set components
	if da.fieldData {
		arr.NumberOfTuples = n
	} else {
		arr.NumberOfComponents = n
	}

	// inline: save data and append
	if da.appended == nil {
		arr.Data = bytes
		da.Data = append(da.Data, arr)
		return
	}

	format = FormatAppended

	// appended data is required to start with underscore ("_")
	if len(da.appended.Data) == 0 {
		da.appended.Data = []byte("_")
	}

	// set offset: subtract 1 to correct for underscore
	arr.Offset = new(int)
	*arr.Offset = len(da.appended.Data) - 1

	// store data, append array
	da.appended.Data = append(da.appended.Data, bytes...)
	da.Data = append(da.Data, arr)
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
