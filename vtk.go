package govtk

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"reflect"
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

// we could make a type vtktype string?
const (
	// VTK XML types.
	imageData        = "ImageData"
	rectilinearGrid  = "RectilinearGrid"
	structuredGrid   = "StructuredGrid"
	unstructuredGrid = "UnstructuredGrid"

	// Data format representations
	formatAscii    = "ascii"
	formatBinary   = "binary"
	formatRaw      = "raw"
	formatAppended = "appended"

	// Methods of encoding binary data in the VTK
	// Note: encodingRaw breaks XML standards.
	encodingRaw    = "raw"
	encodingBase64 = "base64"

	// Identifier for zlib compressed data in VTK XML
	zlibCompressor = "vtkZLibDataCompressor"
)

// header of the vtu Files
type Header struct {
	XMLName     xml.Name `xml:"VTKFile"`
	Type        string   `xml:"type,attr"`
	Version     float64  `xml:"version,attr"`
	ByteOrder   string   `xml:"byte_order,attr"`
	HeaderType  string   `xml:"header_type,attr,omitempty"` // todo do is this req?
	Compression string   `xml:"compressor,attr,omitempty"`
	Grid        Grid
	Appended    *darray

	format     string
	compressor compressor

	// On true writes Legacy (*.vtk) format
	legacy bool
}

// header options
// todo add const of default options (i.e. compressed + base64)
type Option func(h *Header) error

// Construct new header describing the vtu file
func newHeader(t string, opts ...Option) (*Header, error) {
	h := &Header{
		Type:       t,
		Version:    2.0,
		ByteOrder:  "LittleEndian",
		Grid:       Grid{XMLName: xml.Name{Local: t}},
		format:     formatBinary, // improve with better default settings
		compressor: noCompression{},
	}

	// apply all options
	for _, opt := range opts {
		if err := opt(h); err != nil {
			return nil, err
		}
	}

	return h, nil
}

// data: image or unstructured
type Grid struct {
	XMLName xml.Name
	Extent  string     `xml:"WholeExtent,attr,omitempty"`
	Origin  string     `xml:"Origin,attr,omitempty"`
	Spacing string     `xml:"Spacing,attr,omitempty"`
	Data    *dataArray `xml:"FieldData,omitempty"`
	Pieces  []partition
}

// Partition contains all vtu related data of a partition of the mesh, this
// partition can be the complete, or a subset of, the mesh. The VTU docs
// refer to a partition as a "Piece".
type partition struct {
	XMLName        xml.Name   `xml:"Piece"`
	Extent         string     `xml:"Extent,attr,omitempty"`
	NumberOfPoints int        `xml:"NumberOfPoints,attr"`
	NumberOfCells  int        `xml:"NumberOfCells,attr"`
	Points         *dataArray `xml:",omitempty"` // todo seems overly verbose?
	Cells          *dataArray `xml:",omitempty"`
	Coordinates    *dataArray `xml:",omitempty"`
	PointData      *dataArray `xml:",omitempty"`
	CellData       *dataArray `xml:",omitempty"`
}

func (h *Header) NewArray() *dataArray {
	return h.createArray(false)
}

func (h *Header) NewFieldArray() *dataArray {
	return h.createArray(true)
}

// setAppendedData defines a darray to store the appended data when this has
// not been set. Otherwise, the function updates the encoding method to
// either raw or base64. By default, the appended data assumse base64 encoding.
func (h *Header) setAppendedData() {
	var enc string
	switch h.format {
	case formatRaw:
		enc = encodingRaw
	case formatBinary:
		enc = encodingBase64
	default:
		enc = encodingBase64
	}

	if h.Appended != nil {
		h.Appended.Encoding = enc
	} else {
		h.Appended = &darray{
			XMLName:  xml.Name{Local: "AppendedData"},
			Encoding: enc}
	}
}

// createArray creates a dataArray with encoder matchting the its format.
func (h *Header) createArray(fieldData bool) *dataArray {
	var enc encoder
	switch h.format {
	case formatAscii:
		enc = asciier{}
	case formatBinary:
		enc = base64er{}
	case formatRaw:
		enc = binaryer{}
	}

	return newDataArray(enc, h.compressor, fieldData, h.Appended)
}

// Set applies a set of Options to the header
func (h *Header) Add(ops ...Option) error {
	for _, op := range ops {
		if err := op(h); err != nil {
			return err
		}
	}
	return nil
}

// Add points
func Points(data []float64) Option {
	return func(h *Header) error {
		lp := h.lastPiece()

		if lp.Points != nil {
			return fmt.Errorf("Points allready set")
		}

		lp.Points = h.NewArray()
		lp.NumberOfPoints = len(data) / 3
		return lp.Points.add("Points", 3, data)
	}
}

func Piece(opts ...func(p *partition)) Option {
	return func(h *Header) error {
		p := partition{}
		for _, opt := range opts {
			opt(&p)
		}
		h.Grid.Pieces = append(h.Grid.Pieces, p)
		return nil
	}
}

// Data adds data to the file. When len(data) matched the number of points
// or cells, the data is written accordingly. For ambiguous cases, the
// function returns an error. The PointData and CellData calls should then be
// considered instead.
func Data(name string, data interface{}) Option {
	return func(h *Header) error {

		lp := h.lastPiece()

		if lp.NumberOfPoints == lp.NumberOfCells {
			return fmt.Errorf("num cells == num points, cannot infer")
		}

		n := reflect.ValueOf(data).Len()
		if n%lp.NumberOfPoints == 0 {
			return h.pointData(name, data)
		}

		if n%lp.NumberOfCells == 0 {
			return h.cellData(name, data)
		}

		return nil
	}
}

// PointData writes the data to point data.
func PointData(name string, data interface{}) Option {
	return func(h *Header) error {
		return h.pointData(name, data)
	}
}

// CellData writes the data to cell data.
func CellData(name string, data interface{}) Option {
	return func(h *Header) error {
		return h.cellData(name, data)
	}
}

func Cells(conn [][]int) Option {
	return func(h *Header) error {
		lp := h.lastPiece()

		if len(lp.Cells.Data) != 0 {
			return fmt.Errorf("Connectivity already set")
		}

		lp.NumberOfCells = len(conn)
		lp.Cells = h.NewArray()

		err := lp.Cells.add("connectivity", 1, conn[0])
		if err != nil {
			return err
		}

		err = lp.Cells.add("offsets", 1, []int{len(conn[0])})
		if err != nil {
			return err
		}

		err = lp.Cells.add("types", 1, []int{10})
		if err != nil {
			return err
		}

		return nil
	}
}

func FieldData(name string, data []float64) Option {
	return func(h *Header) error {

		if h.Grid.Data == nil {
			h.Grid.Data = h.NewFieldArray()
		}

		return h.Grid.Data.add(name, len(data), data)
	}
}

func Coordinates(x, y, z []float64) Option {
	return func(h *Header) error {
		lp := h.lastPiece()

		if len(lp.Coordinates.Data) != 0 {
			return fmt.Errorf("Coordinates already set")
		}

		lp.NumberOfPoints = len(x)
		lp.Coordinates = h.NewArray()

		err := lp.Coordinates.add("x_coordinates", 1, x)
		if err != nil {
			return err
		}

		err = lp.Coordinates.add("y_coordinates", 1, y)
		if err != nil {
			return err
		}

		err = lp.Coordinates.add("z_coordinates", 1, z)
		if err != nil {
			return err
		}

		return nil
	}
}

func Ascii() Option {
	return func(h *Header) error {
		if h.Appended != nil {
			msg := "Cannot use '%v' encvoding with appended data"
			return fmt.Errorf(msg, formatAscii)
		}

		h.format = formatAscii
		return nil
	}
}

// The binary VTU format is actually base64 encoded to not break xml
func Binary() Option {
	return func(h *Header) error {
		h.format = formatBinary
		return nil
	}
}

func Raw() Option {
	return func(h *Header) error {
		h.format = formatRaw
		h.setAppendedData()
		h.HeaderType = "UInt32" // combine this into an internal setting maybe?
		return nil
	}
}

func Appended() Option {
	return func(h *Header) error {
		if h.format == formatAscii {
			msg := "Cannot use appended data with format '%v'"
			return fmt.Errorf(msg, h.format)
		}
		h.setAppendedData()
		h.HeaderType = "UInt32"
		return nil
	}
}

// Compressed assigns the compressor using the DefaultCompression level.
func Compressed() Option {
	return CompressedLevel(DefaultCompression)
}

// CompressedLevel assigns the compressor using a specific compression level.
// Constants are provided: NoCompression, BestSpeed, BestCompression,
// DefaultCompression, and HuffmanOnly.
func CompressedLevel(level int) Option {
	return func(h *Header) error {
		h.HeaderType = "UInt32"

		if level == NoCompression {
			h.compressor = noCompression{}
			return nil
		}

		h.Compression = zlibCompressor // todo update names
		h.compressor = zlibCompression{level: level}
		return nil
	}
}

func WholeExtent(x0, x1, y0, y1, z0, z1 int) Option {
	f := func(h *Header) error {
		str := fmt.Sprintf("%d %d %d %d %d %d", x0, x1, y0, y1, z0, z1)
		h.Grid.Extent = str
		return nil
	}
	return f
}

func Origin(x, y, z float64) Option {
	return func(h *Header) error {
		h.Grid.Origin = fmt.Sprintf("%f %f %f", x, y, z)
		return nil
	}
}

func Spacing(dx, dy, dz float64) Option {
	return func(h *Header) error {
		h.Grid.Spacing = fmt.Sprintf("%f %f %f", dx, dy, dz)
		return nil
	}
}

func Extent(x0, x1, y0, y1, z0, z1 int) func(p *partition) {
	f := func(p *partition) {
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
func Image(opts ...Option) (*Header, error) {
	return newHeader(imageData, opts...)
}

// Create file with rectilinear grid format
func Rectilinear(opts ...Option) (*Header, error) {
	return newHeader(rectilinearGrid, opts...)
}

// Create file with structured format
func Structured(opts ...Option) (*Header, error) {
	return newHeader(structuredGrid, opts...)
}

// Create file with unstructured format
func Unstructured(opts ...Option) (*Header, error) {
	return newHeader(unstructuredGrid, opts...)
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

// Encodes the xml towards a io.Writer. Writes a xml header (i.e.
// xml.Header constant) to the buffer first for both ascii and base64 formats.
// The header is omitted for formatRaw as this is actually not compliant with
// the xml standard.
func (h *Header) Write(w io.Writer) error {
	if h.format != formatRaw {
		_, err := w.Write([]byte(xml.Header))
		if err != nil {
			return err
		}
	}
	return xml.NewEncoder(w).Encode(h)
}

// pointData is the internal routine to write data along points. The function
// returns an error if the data does not distribute over the number of points.
func (h *Header) pointData(name string, data interface{}) error {
	lp := h.lastPiece()

	if lp.PointData == nil {
		lp.PointData = h.NewArray()
	}

	n := reflect.ValueOf(data).Len()
	if n%lp.NumberOfPoints > 0 {
		return fmt.Errorf("Data does not distribute over points")
	}

	n /= lp.NumberOfPoints
	return lp.PointData.add(name, n, data)
}

// cellData is the internal routine to write data along cells. The function
// returns an error if the data does not distribute over the number of cells.
func (h *Header) cellData(name string, data interface{}) error {
	lp := h.lastPiece()

	if lp.CellData == nil {
		lp.CellData = h.NewArray()
	}

	n := reflect.ValueOf(data).Len()
	if n%lp.NumberOfCells > 0 {
		return fmt.Errorf("Data does not distribute over cells, len %v got %v",
			lp.NumberOfCells, n)
	}

	n /= lp.NumberOfCells
	return lp.CellData.add(name, n, data)
}

// Returns pointer to last piece in the mesh
func (h *Header) lastPiece() *partition {

	if len(h.Grid.Pieces) == 0 {
		switch h.Type {
		case imageData, rectilinearGrid, structuredGrid:
			b := stringToInts(h.Grid.Extent)
			h.Add(Piece(Extent(b[0], b[1], b[2], b[3], b[4], b[5])))
		default:
			h.Add(Piece())
		}
	}

	return &h.Grid.Pieces[len(h.Grid.Pieces)-1]
}
