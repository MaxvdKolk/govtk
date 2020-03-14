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
	FormatAscii      = "ascii"
	FormatAppended   = "appended"
	FormatBinary     = "binary"
	FormatRaw        = "raw"
	ZlibCompressor   = "vtkZLibDataCompressor"
)

// header of the vtu Files
type Header struct {
	XMLName     xml.Name   `xml:"VTKFile"`
	Type        string     `xml:"type,attr"`
	Version     float64    `xml:"version,attr"`
	ByteOrder   string     `xml:"byte_order,attr"`
	Format      string     `xml:"-"`
	HeaderType  string     `xml:"header_type,attr,omitempty"` // todo do is this req?
	Compression string     `xml:"compressor,attr,omitempty"`
	compressor  compressor // compression method
	Grid        Grid
	Appended    *DArray
}

// header options
type Option func(h *Header) error

// Construct new header describing the vtu file
func newHeader(t string, opts ...Option) (*Header, error) {
	h := &Header{
		Type:       t,
		Version:    2.0,
		ByteOrder:  "LittleEndian",
		Grid:       Grid{XMLName: xml.Name{Local: t}},
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
	Data    *DataArray `xml:"FieldData,omitempty"`
	Pieces  []Partition
}

// Partition contains all vtu related data of a partition of the mesh, this
// partition can be the complete, or a subset of, the mesh. The VTU docs
// refer to a partition as a "Piece".
type Partition struct {
	XMLName        xml.Name   `xml:"Piece"`
	Extent         string     `xml:"Extent,attr,omitempty"`
	NumberOfPoints int        `xml:"NumberOfPoints,attr"`
	NumberOfCells  int        `xml:"NumberOfCells,attr"`
	Points         *DataArray `xml:",omitempty"` // todo seems overly verbose?
	Cells          *DataArray `xml:",omitempty"`
	Coordinates    *DataArray `xml:",omitempty"`
	PointData      *DataArray `xml:",omitempty"`
	CellData       *DataArray `xml:",omitempty"`
}

func (h *Header) NewArray() *DataArray {
	return h.createArray(false)
}

func (h *Header) NewFieldArray() *DataArray {
	return h.createArray(true)
}

// setAppendedData defines a DArray to store the appended data when this has
// not been set. Otherwise, the function updates the encoding method to
// either raw or base64. By default, the appended data assumse base64 encoding.
func (h *Header) setAppendedData() {
	var enc string
	switch h.Format {
	case FormatRaw:
		enc = "raw"
	case FormatBinary:
		enc = "base64"
	default:
		enc = "base64"
	}

	if h.Appended != nil {
		h.Appended.Encoding = enc
	} else {
		h.Appended = &DArray{
			XMLName:  xml.Name{Local: "AppendedData"},
			Encoding: enc}
	}
}

func (h *Header) createArray(fieldData bool) *DataArray {
	var enc encoder
	switch h.Format {
	case FormatAscii:
		enc = Asciier{}
	case FormatBinary:
		enc = Base64er{}
	case FormatRaw:
		enc = Binaryer{}
	default:
		panic("not sure what array to add")
	}

	return NewDataArray(enc, h.compressor, fieldData, h.Appended)
}

// Set applies a set of Options to the header
func (h *Header) Add(ops ...Option) {
	for _, op := range ops {
		op(h)
	}
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

func Piece(opts ...func(p *Partition)) Option {
	return func(h *Header) error {
		p := Partition{}
		for _, opt := range opts {
			opt(&p)
		}
		h.Grid.Pieces = append(h.Grid.Pieces, p)
		return nil
	}
}

// todo add some private functions that add the actual data, these
// functions can then just be wrappers around the internal api calls?
func Data(name string, data []float64) Option {
	return func(h *Header) error {

		lp := h.lastPiece()

		if lp.NumberOfPoints == lp.NumberOfCells {
			return fmt.Errorf("num cells == num points, cannot infer")
		}

		if len(data)%lp.NumberOfPoints == 0 {
			return h.pointData(name, data)
		}

		if len(data)%lp.NumberOfCells == 0 {
			return h.cellData(name, data)
		}

		return nil
	}
}

func PointData(name string, data []float64) Option {
	return func(h *Header) error {
		return h.pointData(name, data)
	}
}

func CellData(name string, data []float64) Option {
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
			return fmt.Errorf(msg, FormatAscii)
		}

		h.Format = FormatAscii
		return nil
	}
}

// The binary VTU format is actually base64 encoded to not break xml
func Binary() Option {
	return func(h *Header) error {
		h.Format = FormatBinary
		return nil
	}
}

func Raw() Option {
	return func(h *Header) error {
		h.Format = FormatRaw
		h.setAppendedData()
		h.HeaderType = "UInt32" // combine this into an internal setting maybe?
		return nil
	}
}

func Appended() Option {
	return func(h *Header) error {
		if h.Format == FormatAscii {
			msg := "Cannot use appended data with format '%v'"
			return fmt.Errorf(msg, h.Format)
		}
		h.setAppendedData()
		h.HeaderType = "UInt32"
		return nil
	}
}

func Compressed() Option {
	return CompressedLevel(zlib.DefaultCompression)
}

func CompressedLevel(level int) Option {
	return func(h *Header) error {
		h.HeaderType = "UInt32"
		h.Compression = ZlibCompressor // todo update names
		h.compressor = zlibCompression{}
		return nil
	}
}

func WholeExtent(x0, x1, y0, y1, z0, z1 int) Option {
	f := func(h *Header) error {
		str := fmt.Sprintf("%d %d %d %d %d %d", x0, x1, y0, y1, z0, z1)
		h.Grid.Extent = str
		//h.Grid.Extent = []int{x0, x1, y0, y1, z0, z1}
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
func Image(opts ...Option) (*Header, error) {
	return newHeader(ImageData, opts...)
}

// Create file with rectilinear grid format
func Rectilinear(opts ...Option) (*Header, error) {
	return newHeader(RectilinearGrid, opts...)
}

// Create file with structured format
func Structured(opts ...Option) (*Header, error) {
	return newHeader(StructuredGrid, opts...)
}

// Create file with unstructured format
func Unstructured(opts ...Option) (*Header, error) {
	return newHeader(UnstructuredGrid, opts...)
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
// The header is omitted for FormatRaw as this is actually not compliant with
// the xml standard.
func (h *Header) Write(w io.Writer) error {
	if h.Format != FormatRaw {
		_, err := w.Write([]byte(xml.Header))
		if err != nil {
			return err
		}
	}
	return xml.NewEncoder(w).Encode(h)
}

// Add data specified on points
func (h *Header) pointData(name string, data []float64) error {
	lp := h.lastPiece()

	if lp.PointData == nil {
		lp.PointData = h.NewArray()
	}

	if len(data)%lp.NumberOfPoints > 0 {
		return fmt.Errorf("Data does not distribute over points")
	}

	nc := len(data) / lp.NumberOfPoints
	return lp.PointData.add(name, nc, data)
}

func (h *Header) cellData(name string, data []float64) error {
	lp := h.lastPiece()

	if lp.CellData == nil {
		lp.CellData = h.NewArray()
	}

	if len(data)%lp.NumberOfCells > 0 {
		return fmt.Errorf("Data does not distribute over cells")
	}

	nc := len(data) / lp.NumberOfCells
	return lp.CellData.add(name, nc, data)
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
