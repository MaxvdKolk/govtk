package govtk

import (
	"strings"
	"testing"
)

// Ensure DArray's start out empty and with nil pointer offset
func TestNewDArray(t *testing.T) {
	xmlName := "xmlName"
	dtype := "dtype"
	name := "name"
	format := "format"
	da := newDArray(xmlName, dtype, name, format)

	if !strings.EqualFold(da.XMLName.Local, xmlName) {
		t.Errorf("Wrong identifier")
	}
	if !strings.EqualFold(da.Type, dtype) {
		t.Errorf("Wrong identifier")
	}
	if !strings.EqualFold(da.Name, name) {
		t.Errorf("Wrong identifier")
	}
	if !strings.EqualFold(da.Format, format) {
		t.Errorf("Wrong identifier")
	}
	if len(da.Data) > 0 {
		t.Errorf("New darray should start without data")
	}
	if da.NumberOfComponents > 0 && da.NumberOfTuples > 0 {
		t.Errorf("New darray should have no components/tuples")
	}
	if da.Offset != nil {
		t.Errorf("New darray should start without offset pointer %v", da.Offset)
	}
}

// test expected values provide the right data type descriptions
func TestDataType(t *testing.T) {
	da := &dataArray{}

	type pair struct {
		val interface{}
		str string
	}
	// values that should parse
	pairs := []pair{
		pair{val: float32(1.0), str: "Float32"},
		pair{val: float64(1.0), str: "Float64"},
		pair{val: int(1), str: "UInt32"},
		pair{val: int32(1), str: "UInt32"},
		pair{val: int64(1), str: "UInt64"},
	}
	for _, pair := range pairs {
		str, err := da.dataType(pair.val)
		if err != nil {
			t.Error(err)
		}

		if !strings.EqualFold(str, pair.str) {
			t.Error("Datatype strings are not equal")
		}
	}

	// values that should return an error
	pairs = []pair{
		pair{val: string("1.0"), str: ""},
	}
	for _, pair := range pairs {
		_, err := da.dataType(pair.val)
		if err == nil {
			t.Error("Should return error")
		}

	}
}
