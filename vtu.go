package vtu

import (
	"compress/zlib"
	"encoding/xml"
	"fmt"
	"io"
	"os"
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
	ascii            = "ascii"
	FormatAppended   = "appended"
	FormatBinary     = "binary"
	FormatRaw        = "raw"
	ZlibCompressor   = "vtkZLibDataCompressor"
)

// header of the vtu Files
type Header struct {
	XMLName      xml.Name   `xml:"VTKFile"`
	Type         string     `xml:"type,attr"`
	Version      float64    `xml:"version,attr"`
	ByteOrder    string     `xml:"byte_order,attr"`
	Format       string     `xml:"-"`
	HeaderType   string     `xml:"header_type,attr,omitempty"` // todo do is this req?
	Compression  string     `xml:"compressor,attr,omitempty"`
	Compressor   Compressor `xml:"-"` // the method of data compression?
	Grid         Grid
	Append       bool `xml:"-"`
	AppendedData *DArray
}

// Construct new header describing the vtu file
func newHeader(t string, opts ...Option) *Header {
	h := &Header{
		Type:      t,
		Version:   2.0,
		ByteOrder: "LittleEndian",
		Grid:      Grid{XMLName: xml.Name{Local: t}},
	}

	// apply all options
	for _, opt := range opts {
		opt(h)
	}

	return h
}

// DataArray is a container with names and properties regarding a set of data
type DArray struct {
	XMLName            xml.Name //`xml:"DataArray"`
	Type               string   `xml:"type,attr,omitempty"`
	Name               string   `xml:"Name,attr,omitempty"`
	Format             string   `xml:"format,attr,omitempty"`
	NumberOfComponents int      `xml:"NumberOfComponents,attr,omitempty"`
	NumberOfTuples     int      `xml:"NumberOfTuples,attr,omitempty"`
	Offset             string   `xml:"offset,attr,omitempty"`
	Data               string   `xml:",chardata"`
	RawData            []byte   `xml:",innerxml"`
	Encoding           string   `xml:"encoding,attr,omitempty"`
	offset             int      `xml:"-"`
}

// return new DArray
func NewDArray(Type, Name, Format string, NoC int, Data string) *DArray {
	return &DArray{
		XMLName:            xml.Name{Local: "DataArray"},
		Type:               Type,   // data type
		Name:               Name,   // name of field
		Format:             Format, // ascii vs binary
		NumberOfComponents: NoC,    // number of components (x, y, z)
		Data:               Data,   // data converted to string
	}
}

// data: image or unstructured
type Grid struct {
	XMLName xml.Name
	Extent  string    `xml:"WholeExtent,attr,omitempty"`
	Origin  string    `xml:"Origin,attr,omitempty"`
	Spacing string    `xml:"Spacing,attr,omitempty"`
	Data    DataArray `xml:"FieldData,omitempty"`
	Pieces  []Partition
}

// Partition contains all vtu related data of a partition of the mesh, this
// partition can be the complete, or a subset of, the mesh. The VTU docs
// refer to a partition as a "Piece".
type Partition struct {
	XMLName        xml.Name  `xml:"Piece"`
	Extent         string    `xml:"Extent,attr,omitempty"`
	NumberOfPoints int       `xml:"NumberOfPoints,attr"`
	NumberOfCells  int       `xml:"NumberOfCells,attr"`
	Points         DataArray `xml:",omitempty"` // todo seems overly verbose?
	Cells          DataArray `xml:",omitempty"`
	Coordinates    DataArray `xml:",omitempty"`
	PointData      DataArray `xml:",omitempty"`
	CellData       DataArray `xml:",omitempty"`
}

func (h *Header) NewArray() DataArray {
	return h.createArray(false)
}

func (h *Header) NewFieldArray() DataArray {
	return h.createArray(true)
}

func (h *Header) createArray(fieldData bool) DataArray {

	var a Appender

	a = &Inline{}

	// todo improve this stuf...
	// ensure the storage is present
	if h.Append {
		if h.AppendedData == nil {
			h.AppendedData = &DArray{XMLName: xml.Name{Local: "AppendedData"}}

			if h.Format == FormatRaw {
				h.AppendedData.Encoding = "raw"
			} else {
				h.AppendedData.Encoding = "base64"
			}
		}

		a = &Appending{Array: h.AppendedData}
	}

	var c Compressor
	if h.Compressor != nil {
		c = h.Compressor
	} else {
		c = &NoCompressor{}
	}

	switch h.Format {
	case ascii:
		return &Array{
			fieldData:  fieldData,
			Compressor: c,
			Appender:   &Inline{},
			Encoder:    Asciier{},
		}
	case FormatBinary:
		return &Array{
			fieldData:  fieldData,
			Compressor: c,
			Appender:   a,
			Encoder:    Base64er{},
		}
	case FormatRaw:
		return &Array{
			fieldData:  fieldData,
			Compressor: c,
			Appender:   a,
			Encoder:    Binaryer{},
		}
	default:
		panic("not sure what array to add")
	}
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

		if lp.Points != nil {
			panic("points allready set")
		}

		lp.Points = h.NewArray()
		lp.NumberOfPoints = len(data) / 3
		lp.Points.Floats("Points", 3, data)
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
			h.cellData(name, data)
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
		h.cellData(name, data)
	}
}

func Cells(conn [][]int) Option {
	return func(h *Header) {
		lp := h.lastPiece()

		if lp.Cells != nil {
			panic("connectivity already set")
		}

		lp.NumberOfCells = len(conn)

		lp.Cells = h.NewArray()
		lp.Cells.Ints("connectivity", 1, conn[0])
		lp.Cells.Ints("offsets", 1, []int{len(conn[0])})
		lp.Cells.Ints("types", 1, []int{10})
	}
}

func FieldData(name string, data []float64) Option {
	return func(h *Header) {

		if h.Grid.Data == nil {
			h.Grid.Data = h.NewFieldArray()
		}

		h.Grid.Data.Floats(name, len(data), data)
	}
}

func Coordinates(x, y, z []float64) Option {
	return func(h *Header) {
		lp := h.lastPiece()

		if lp.Coordinates != nil {
			panic("Coordinates were already set")
		}

		lp.NumberOfPoints = len(x)

		lp.Coordinates = h.NewArray()
		lp.Coordinates.Floats("x_coordinates", 1, x)
		lp.Coordinates.Floats("y_coordinates", 1, y)
		lp.Coordinates.Floats("z_coordinates", 1, z)
	}
}

// header options
type Option func(h *Header)

func Ascii() Option {
	return func(h *Header) {
		h.Format = ascii
	}
}

// The binary VTU format is actually base64 encoded to not break xml
func Binary() Option {
	return func(h *Header) {
		h.Format = FormatBinary
	}
}

func Raw() Option {
	return func(h *Header) {
		h.Format = FormatRaw
		h.Append = true
		h.HeaderType = "UInt32" // combine this into an internal setting maybe?
	}
}

func Appended() Option {
	return func(h *Header) {
		h.Append = true
		h.HeaderType = "UInt32"
	}
}

func Compressed() Option {
	return CompressedLevel(zlib.DefaultCompression)
}

func CompressedLevel(level int) Option {
	return func(h *Header) {
		h.HeaderType = "UInt32"
		h.Compression = ZlibCompressor // todo update names
		h.Compressor = &Zlib{level: level}
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

// Save opens a file and writes the xml
func (h *Header) Save(filename string) error {
	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		return err
	}
	return h.Write(f)
}

// Encodes the xml towards any io.Writer
func (h *Header) Write(w io.Writer) error {
	return xml.NewEncoder(w).Encode(h)
}

// Add data specified on points
func (h *Header) pointData(name string, data []float64) {
	lp := h.lastPiece()

	if lp.PointData == nil {
		lp.PointData = h.NewArray()
	}

	if len(data)%lp.NumberOfPoints > 0 {
		panic("data does not distribute")
	}

	nc := len(data) / lp.NumberOfPoints
	lp.PointData.Floats(name, nc, data)
}

func (h *Header) cellData(name string, data []float64) {
	lp := h.lastPiece()

	if lp.CellData == nil {
		lp.CellData = h.NewArray()
	}

	if len(data)%lp.NumberOfCells > 0 {
		panic("data does not distribute")
	}

	nc := len(data) / lp.NumberOfCells
	lp.CellData.Floats(name, nc, data)
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
