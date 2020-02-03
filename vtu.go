package vtu

import (
	"encoding/xml"
	"fmt"
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

// todo add mapping of types
//var gmshToVTK = map[int]int{
//	1:  3,
//	2:  5,
//	3:  9,
//	4:  10,
//	5:  12,
//	8:  21,
//	9:  22,
//	11: 24,
//	15: 1,
//	16: 23,
//}

const (
	ImageData        = "ImageData"
	RectilinearGrid  = "RectilinearGrid"
	StructuredGrid   = "StructuredGrid"
	UnstructuredGrid = "UnstructuredGrid"
	PData            = "PointData"
	CData            = "CellData"
	FData            = "FieldData"
	ascii            = "ascii"
	binary           = "binary"
)

// header of the vtu Files
type Header struct {
	XMLName   xml.Name `xml:"VTKFile"`
	Type      string   `xml:"type,attr"`
	Version   float64  `xml:"version,attr"`
	ByteOrder string   `xml:"byte_order,attr"`
	//HeaderType string   `xml:"header_type,attr"` // todo do is this req?
	Grid   Grid
	Format string
}

// Construct new header describing the vtu file
func newHeader(t string, opts ...Option) *Header {
	h := &Header{
		Type:      t,
		Version:   2.0,
		ByteOrder: "LittleEndian",
		//HeaderType: "Uint32",
		Grid: Grid{XMLName: xml.Name{Local: t}},
	}

	// apply all options
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// DataArray is a container with names and properties regarding a set of data
type DArray struct {
	XMLName            xml.Name `xml:"DataArray"`
	Type               string   `xml:"type,attr,omitempty"`
	Name               string   `xml:"Name,attr,omitempty"`
	Format             string   `xml:"format,attr,omitempty"`
	NumberOfComponents int      `xml:"NumberOfComponents,attr,omitempty"`
	NumberOfTuples     int      `xml:"NumberOfTuples,attr,omitempty"`
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

// data: image or unstructured
type Grid struct {
	XMLName xml.Name
	Extent  string `xml:"WholeExtent,attr,omitempty"`
	Origin  string `xml:"Origin,attr,omitempty"`
	Spacing string `xml:"Spacing,attr,omitempty"`
	Data    Array  `xml:"FieldData,omitempty"`
	Pieces  []Partition
}

// Partition contains all vtu related data of a partition of the mesh, this
// partition can be the complete, or a subset of, the mesh. The VTU docs
// refer to a partition as a "Piece".
type Partition struct {
	XMLName        xml.Name `xml:"Piece"`
	Extent         string   `xml:"Extent,attr,omitempty"`
	NumberOfPoints int      `xml:"NumberOfPoints,attr"`
	NumberOfCells  int      `xml:"NumberOfCells,attr"`
	Points         Array    `xml:",omitempty"` // todo seems overly verbose?
	Cells          Array    `xml:",omitempty"`
	Coordinates    Array    `xml:",omitempty"`
	PointData      Array    `xml:",omitempty"`
	CellData       Array    `xml:",omitempty"`
}

// Array represents a set of data arrays, with a flexible name
// by not specifying it, we can keep it variable
// Array{XMLName: xml.Name{Local: "foo"}} to name it foo
type Array struct {
	XMLName xml.Name
	Data    []*DArray // any number of data arrays should be possible
}

// Set applies a set of Options to the header
func (h *Header) Add(ops ...Option) {
	for _, op := range ops {
		op(h)
	}
}

// Add points
func Points(data []float64) Option {
	return func(h *Header) {
		lp := h.lastPiece()
		d := &lp.Points
		name := "Points"
		if len(d.Data) > 0 {
			panic("points allready set")
		}
		lp.NumberOfPoints = len(data) / 3
		d.Data = append(d.Data, FloatArray(name, h.Format, 3, data))
	}
}

func Piece(opts ...func(p *Partition)) Option {
	return func(h *Header) {
		p := Partition{}
		for _, opt := range opts {
			opt(&p)
		}
		h.Grid.Pieces = append(h.Grid.Pieces, p)
	}
}

// todo add some private functions that add the actual data, these
// functions can then just be wrappers around the internal api calls?
func Data(name string, data []float64) Option {
	return func(h *Header) {

		lp := h.lastPiece()

		if lp.NumberOfPoints == lp.NumberOfCells {
			panic("num cells == num points, cannot infer")
		}

		if len(data)%lp.NumberOfPoints == 0 {
			h.pointData(name, data)
			return
		}

		if len(data)%lp.NumberOfCells == 0 {
			nc := len(data) / lp.NumberOfCells
			cd := &lp.CellData
			cd.Data = append(cd.Data, FloatArray(name, h.Format, nc, data))
			return
		}
	}
}

func PointData(name string, data []float64) Option {
	return func(h *Header) {
		h.pointData(name, data)
	}
}

func CellData(name string, data []float64) Option {
	return func(h *Header) {
		lp := h.lastPiece()

		if len(data)%lp.NumberOfCells > 0 {
			panic("data does not distribute")
		}
		nc := len(data) / lp.NumberOfCells

		cd := &lp.CellData
		cd.Data = append(cd.Data, FloatArray(name, h.Format, nc, data))
	}
}

func Cells(conn [][]int) Option {
	return func(h *Header) {
		lp := h.lastPiece()
		d := &lp.Cells
		if len(d.Data) > 0 {
			panic("connectivity already set")
		}

		lp.NumberOfCells = len(conn)
		d.Data = append(d.Data, IntArray("connectivity", h.Format, 1, conn[0]))
		d.Data = append(d.Data, IntArray("offsets", h.Format, 1, []int{len(conn[0])}))
		d.Data = append(d.Data, IntArray("types", h.Format, 1, []int{10}))

	}
}

func FieldData(name string, data []float64) Option {
	return func(h *Header) {
		pd := &h.Grid.Data
		tmpdata := floatToString(data, " ")

		// todo fix this
		tmp := &DArray{
			Type:           "Float64", // data type
			Name:           name,      // name of field
			Format:         h.Format,  // ascii vs binary
			NumberOfTuples: len(data), // number of components (x, y, z)
			Data:           tmpdata,   // data converted to string
		}

		pd.Data = append(pd.Data, tmp)
	}
}

func Coordinates(x, y, z []float64) Option {
	return func(h *Header) {
		lp := h.lastPiece()
		d := &lp.Coordinates
		if len(d.Data) > 0 {
			panic("Coordinates were already set")
		}
		lp.NumberOfPoints = len(x)

		d.Data = append(d.Data, FloatArray("x_coordinates", h.Format, 1, x))
		d.Data = append(d.Data, FloatArray("y_coordinates", h.Format, 1, y))
		d.Data = append(d.Data, FloatArray("z_coordinates", h.Format, 1, z))
	}
}

// header options
type Option func(h *Header)

func Ascii() Option {
	return func(h *Header) {
		h.Format = ascii
	}
}

func WholeExtent(x0, x1, y0, y1, z0, z1 int) Option {
	f := func(h *Header) {
		str := fmt.Sprintf("%d %d %d %d %d %d", x0, x1, y0, y1, z0, z1)
		h.Grid.Extent = str
		//h.Grid.Extent = []int{x0, x1, y0, y1, z0, z1}
	}
	return f
}

func Origin(x, y, z float64) Option {
	return func(h *Header) {
		h.Grid.Origin = fmt.Sprintf("%f %f %f", x, y, z)
	}
}

func Spacing(dx, dy, dz float64) Option {
	return func(h *Header) {
		h.Grid.Spacing = fmt.Sprintf("%f %f %f", dx, dy, dz)
	}
}

func Extent(x0, x1, y0, y1, z0, z1 int) func(p *Partition) {
	f := func(p *Partition) {
		str := fmt.Sprintf("%d %d %d %d %d %d", x0, x1, y0, y1, z0, z1)
		p.Extent = str

		// need to detect zero elements in either direction, then we
		// are just trying to output a single slice (e.g. 2d plane view)
		p.NumberOfCells = (x1 - x0) * (y1 - y0) * (z1 - z0)
		p.NumberOfPoints = (x1 - x0 + 1) * (y1 - y0 + 1) * (z1 - z0 + 1)
	}
	return f
}

// Create file with image data format
func Image(opts ...Option) *Header {
	return newHeader(ImageData, opts...)
}

// Create file with rectilinear grid format
func Rectilinear(opts ...Option) *Header {
	return newHeader(RectilinearGrid, opts...)
}

// Create file with structured format
func Structured(opts ...Option) *Header {
	return newHeader(StructuredGrid, opts...)
}

// Create file with unstructured format
func Unstructured(opts ...Option) *Header {
	v := newHeader(UnstructuredGrid, opts...)
	return v
}

// todo change this to allow a writer interface?
func (h *Header) Write(filename string) {
	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		panic(fmt.Sprintf("error %v", err))
	}

	enc := xml.NewEncoder(f)
	err = enc.Encode(h)
	if err != nil {
		panic(fmt.Sprintf("error %v", err))
	}
}

// Add data specified on points
func (h *Header) pointData(name string, data []float64) {
	lp := h.lastPiece()
	if len(data)%lp.NumberOfPoints > 0 {
		panic("data does not distribute")
	}
	nc := len(data) / lp.NumberOfPoints
	pd := &lp.PointData
	pd.Data = append(pd.Data, FloatArray(name, h.Format, nc, data))
	return
}

// Returns pointer to last piece in the mesh
func (h *Header) lastPiece() *Partition {

	// detect situations without a piece
	if len(h.Grid.Pieces) == 0 {
		switch h.Type {
		case ImageData, RectilinearGrid, StructuredGrid:
			b := stringToInts(h.Grid.Extent)
			//h.Partition(Extent(b[0], b[1], b[2], b[3], b[4], b[5]))
			h.Add(Piece(Extent(b[0], b[1], b[2], b[3], b[4], b[5])))
		default:
			//h.Partition()
			h.Add(Piece())
		}
	}

	return &h.Grid.Pieces[len(h.Grid.Pieces)-1]
}

// custom marshaller? this would avoid pointers?
// in this case, we dont worry about checking nil, just check lengths?
func (a Array) MarshalXML(e *xml.Encoder, start xml.StartElement) (err error) {
	if len(a.Data) == 0 {
		return nil
	}

	err = e.EncodeToken(start)
	if err != nil {
		return
	}

	err = e.Encode(a.Data)
	if err != nil {
		return
	}
	return e.EncodeToken(xml.EndElement{Name: start.Name})
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
