package vtu

import (
	"encoding/xml"
	"fmt"
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

// NewDataArray returns a newly allocated DataArray with encoder, compressor,
// and fieldData flags. Optinal appended DArray pointer can be provided.
func NewDataArray(enc encoder, cmp compressor, fieldData bool, app *DArray) *DataArray {
	if app != nil {
		return &DataArray{
			appended:   app,
			fieldData:  fieldData,
			encoder:    enc,
			compressor: cmp,
		}
	}

	return &DataArray{
		fieldData:  fieldData,
		encoder:    enc,
		compressor: cmp,
	}
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

	// Encoding is only required for appended data values ("raw", "base64")
	Encoding string `xml:"encoding,attr,omitempty"`

	// Offset holds a pointer to int, as we want to omit these values for
	// any DArray that does not require offset, while we do not want to
	// consider Offset = 0 as an empty value. Thus, by making this a
	// pointer, the xml encoding only considers it empty when equal to nil.
	Offset *int `xml:"offset,attr,omitempty"`
}

// NewDArray provides a new DArray with properties set except the data fields
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
//
// TODO: compare to XML VTK requirements
func (da *DataArray) dataType(data interface{}) (string, error) {
	switch data.(type) {
	case int, int32, uint32, []int, []int32, []uint32:
		return "UInt32", nil
	case int64, uint64, []int64, []uint64:
		return "UInt64", nil
	case float64, []float64:
		return "Float64", nil
	case float32, []float32:
		return "Float32", nil
	}

	// todo add err test
	return "", fmt.Errorf("Cannot map data %v (%T) to type", data, data)
}

// Add adds data to the data array. The data can be stored inline or
// appended to a single storage
func (da *DataArray) add(name string, n int, data interface{}) error {
	// encode data into payload
	payload := da.encoder.binarise(data)

	// compress payload
	payload, err := da.compressor.compress(payload)
	if err != nil {
		return err
	}

	// encode payload as []byte
	bytes, err := da.encoder.encode(payload)
	if err != nil {
		return err
	}

	// extract data type to match XML VTK
	dtype, err := da.dataType(data)
	if err != nil {
		return err
	}

	var format string
	if da.appended != nil {
		format = "appended"
	} else {
		format = da.encoder.format()
	}

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
		return nil
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
	return nil
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
