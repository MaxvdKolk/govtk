package vtu

import (
	"encoding/xml"
	"fmt"
	"gonum.org/v1/gonum/mat"
	"os"
	"strconv"
	"strings"
)

/* todo
would it be an idea to separate the pacakge more strictly..., it could
also be possible to see this as a "vtu" writer package... Something that
solely writes out vtu packages and does not require the mesh, nor contains
functions that specifically writes the mesh. That should be inmplemented
inside the mesh part...?

similarly for the xdf? all that can be separate entities that write out
information to files...
*/

// an array that is able to create <DataArray> tag
type DArray struct {
	XMLName            xml.Name `xml:"DataArray"`
	Type               string   `xml:"type,attr"`
	Name               string   `xml:"Name,attr"`
	Format             string   `xml:"format,attr"`
	NumberOfComponents int      `xml:"NumberOfComponents,attr,omitempty"`
	Data               string   `xml:",chardata"`
}

// return new DArray
func NewDArray(Type, Name, Format string, NoC int, Data string) *DArray {
	return &DArray{
		Type:               Type,   // data type
		Name:               Name,   // name of field
		Format:             Format, // ascii vs binary
		NumberOfComponents: NoC,    // number of components (x, y, z)
		Data:               Data,   // data converted to string
	}
}

func IntArray(name, format string, n int, data []int) *DArray {
	d := intToString(data, " ")
	return NewDArray("UInt32", name, format, n, d)
}

func FloatArray(name, format string, n int, data []float64) *DArray {
	d := floatToString(data, " ")
	return NewDArray("Float64", name, format, n, d)
}

// i guess this is natural in a language without generics?
func floatToString(data []float64, sep string) string {
	if len(data) == 0 {
		panic("no data supplied to the VTU output")
	}

	s := make([]string, len(data))
	for i, d := range data {
		s[i] = fmt.Sprintf("%f", d)
	}

	return strings.Join(s, sep)
}

func intToString(data []int, sep string) string {
	if len(data) == 0 {
		panic("no data supplied")
	}

	s := make([]string, len(data))
	for i, d := range data {
		s[i] = strconv.Itoa(d)
	}

	return strings.Join(s, sep)
}

// Data types
const (
	CellData = iota + 1
	PointData
)

// creates the <Piece> tag, i.e. the mesh data
type XMLMesh struct {
	XMLName        xml.Name  `xml:"Piece"`
	NumberOfPoints string    `xml:"NumberOfPoints,attr"`
	NumberOfCells  string    `xml:"NumberOfCells,attr"`
	Points         *DArray   `xml:"Points>DataArray"`
	Cells          []*DArray `xml:"Cells>DataArray"`
	CellData       []*DArray `xml:"CellData>DataArray"`
	PointData      []*DArray `xml:"PointData>DataArray"`
}

// creates a general <VTU> wrapper around a Piece
// in theory, maybe multiple meshes generates multiple pieces? Not sure?
type VTU struct {
	XMLName    xml.Name `xml:"VTKFile"`
	Type       string   `xml:"type,attr"`
	Version    float64  `xml:"version,attr"`
	ByteOrder  string   `xml:"byte_order,attr"`
	HeaderType string   `xml:"header_type,attr"`
	Mesh       XMLMesh  `xml:"UnstructuredGrid>Piece"`
}

// creates a new VTU
func New(mesh XMLMesh) *VTU {
	return &VTU{
		Type:       "UnstructuredGrid",
		Version:    2.1,
		ByteOrder:  "LittleEndian",
		HeaderType: "UInt32",
		Mesh:       mesh,
	}
}

var gmshToVTK = map[int]int{
	1:  3,
	2:  5,
	3:  9,
	4:  10,
	5:  12,
	8:  21,
	9:  22,
	11: 24,
	15: 1,
	16: 23,
}

func ToVTK(in []int) []int {

	out := make([]int, len(in))

	for i := 0; i < len(in); i++ {
		out[i] = gmshToVTK[in[i]]
	}

	return out
}

// convert mesh into vtu output
//func writeVTU(m *Mesh) *VTU {
//
//	format := "ascii"
//
//	// convert mesh into xml format
//	xmlMesh := XMLMesh{
//		NumberOfPoints: strconv.Itoa(m.numPoints()),
//		NumberOfCells:  strconv.Itoa(m.numCells(m.dim)),
//		Points:         FloatArray("Points", format, 3, m.poIntArray()),
//		Cells: []*DArray{
//			IntArray("connectivity", format, 1, m.cells[m.dim][0].indices),
//			IntArray("offsets", format, 1, m.cells[m.dim][0].offsets[1:]),
//			IntArray("types", format, 1, toVTK(m.cells[m.dim][0].types))},
//		PointData: []*DArray{FloatArray("Coords", format, 3, m.poIntArray())},
//		CellData:  []*DArray{},
//	}
//
//	// add additional data to the xml structure
//	// todo there should be a routine to easily add these field?
//	xmlMesh.CellData = append(xmlMesh.CellData, IntArray("types", format, 1, toVTK(m.cells[m.dim][0].types)))
//
//	xmlMesh.CellData = append(xmlMesh.CellData, IntArray("pgroup", format, 1, m.cells[m.dim][0].pgroup))
//
//	return NewVTU(xmlMesh)
//}

// Stores vector to the field defined by named constants. If the vector
// contains two components, it will be automatically expanded towards three
// components. In many cases, this will append an empty (zero) 3rd dimension,
// which makes visualisation of deformed configurations much more
// straightforward in paraview
func (vtu *VTU) AddVector(field int, v *mat.VecDense, name string) {
	// get number of entries in the mesh
	var str string
	if field == CellData {
		str = vtu.Mesh.NumberOfCells
	} else {
		str = vtu.Mesh.NumberOfPoints
	}
	np, _ := strconv.Atoi(str)

	// assert vector fits on number of points
	if v.Len()%np != 0 {
		msg := fmt.Sprintf("Vector %d entries != points %d", np, v.Len())
		panic(msg)
	}

	// initialise the float array to store
	var fA *DArray
	ncomp := v.Len() / np
	if ncomp != 2 {
		// extract directly from the raw vector data
		fA = FloatArray(name, "ascii", ncomp, v.RawVector().Data)
	} else {
		// insert zero's after each 2nd value in the vector
		res := make([]float64, np*3)
		for i := 0; i < np; i++ {
			for j := 0; j < 2; j++ {
				res[3*i+j] = v.AtVec(2*i + j)
			}
		}
		fA = FloatArray(name, "ascii", ncomp+1, res)
	}

	// store array in corresponding data
	if field == CellData {
		vtu.Mesh.CellData = append(vtu.Mesh.CellData, fA)
	} else {
		vtu.Mesh.PointData = append(vtu.Mesh.PointData, fA)
	}
}

// shorthand to store pointData
func (vtu *VTU) addPointVector(v *mat.VecDense, name string) {
	vtu.AddVector(PointData, v, name)
}

// shorthand to store CellData
func (vtu *VTU) addCellVector(v *mat.VecDense, name string) {
	vtu.AddVector(CellData, v, name)
}

// stores the vtu to disk
// todo the name now needs the include the extension
func (vtu *VTU) Save(name string) {
	// open the desired file
	f, err := os.Create(name)
	check(err)
	defer f.Close()

	// write data using the build in xml encoder
	e := xml.NewEncoder(f)
	err = e.Encode(vtu)
	check(err)
	msg := fmt.Sprintf("Written to '%s'.", name)
	fmt.Println(msg)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
