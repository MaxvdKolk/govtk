package govtk

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"reflect"
)

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

// Linear cell types in VTK
//
// Refer to Fig.2 https://vtk.org/wp-content/uploads/2015/04/file-formats.pdf
// for the local element numbering
const (
	Vertex = iota + 1
	PolyVertex
	Line
	PolyLine
	Triangle
	TriangleStrip
	Polygon
	Pixel
	Quad
	Tetra
	Voxel
	Hexahedron
	Wedge
	Pyramid
)

// Non-linear cell types in VTK
//
// Refer to Fig.3 https://vtk.org/wp-content/uploads/2015/04/file-formats.pdf
// for the local element numbering
const (
	QuadraticEdge = iota + 21
	QuadraticTriangle
	QuadraticQuad
	QuadraticTetra
	QuadraticHexahedron
)

// header of the vtu Files
type Header struct {
	XMLName     xml.Name `xml:"VTKFile"`
	Type        string   `xml:"type,attr"`
	Version     float64  `xml:"version,attr"`
	ByteOrder   string   `xml:"byte_order,attr"`
	HeaderType  string   `xml:"header_type,attr,omitempty"`
	Compression string   `xml:"compressor,attr,omitempty"`
	Grid        Grid
	Appended    *darray

	format     string
	compressor compressor

	// maps user's element label towards vtk's element type
	labelType map[int]int

	// On true writes Legacy (*.vtk) format
	legacy bool
}

// header options
type Option func(h *Header) error

// Construct new header describing the vtu file
func newHeader(t string, opts ...Option) (*Header, error) {
	h := &Header{
		Type:      t,
		Version:   1.0,
		ByteOrder: "LittleEndian",
		Grid:      Grid{XMLName: xml.Name{Local: t}},
		format:    formatBinary,
		//compressor: zlibCompression{},
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

// bounds int values represent the extent of the grid or piece.
type bounds [6]int

// newBounds validates the provided values of the bounds before returing
// the bounds. The bounds should be sorted, i.e. min before max values, and
// should have dimensionality >= 2. If not, the function returns an error,
// indicating a problem with the provided bounds.
func newBounds(x0, x1, y0, y1, z0, z1 int) (bounds, error) {
	if x0 > x1 || y0 > y1 || z0 > z1 {
		msg := "Extent values should be sorted low - high"
		return bounds{}, fmt.Errorf(msg)
	}

	dim := 3
	for _, n := range [3]int{x1 - x0, y1 - y0, z1 - z0} {
		if n == 0 {
			dim--
		}
	}
	if dim < 2 {
		msg := "Image requires at least two dimensions"
		return bounds{}, fmt.Errorf(msg)
	}
	return bounds{x0, x1, y0, y1, z0, z1}, nil
}

// zeroDim returns the index of the zeroth dimension in the bounds. In case
// all dimensions are non-zero, the function returns -1
func (b bounds) zeroDim() int {
	for i := 0; i < len(b); i += 2 {
		if b[i+1]-b[i] == 0 {
			return i / 2
		}
	}
	return -1
}

// evaluates the number of cells based on the extent of the domain
func (b bounds) numCells() int {
	nc := 1
	for i := 0; i < len(b); i += 2 {
		if b[i+1]-b[i] > 0 {
			nc *= (b[i+1] - b[i])
		}
	}
	return nc
}

// evaluates the number of points based on the extent of the domain
func (b bounds) numPoints() int {
	np := 1
	for i := 0; i < len(b); i += 2 {
		if b[i+1]-b[i] > 0 {
			np *= (b[i+1] - b[i] + 1)
		}
	}
	return np
}

func (b bounds) String() string {
	return fmt.Sprintf("%d %d %d %d %d %d",
		b[0], b[1], b[2], b[3], b[4], b[5],
	)
}

func (b bounds) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: fmt.Sprint(b)}, nil
}

// data: image or unstructured
type Grid struct {
	XMLName xml.Name
	Extent  bounds     `xml:"WholeExtent,attr,omitempty"`
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
	Extent         bounds     `xml:"Extent,attr,omitempty"`
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

func Points(xyz ...interface{}) Option {
	return func(h *Header) error {
		switch h.Type {
		case rectilinearGrid:
			return h.coordinates(xyz...)
		case structuredGrid:
			return h.structuredPoints(xyz...)
		case unstructuredGrid:
			return h.unstructuredPoints(xyz...)
		}
		return nil
	}
}

// Points adds a st of coordinates to the structured grid. The points can be
// provided either as a single slice ordered x, y, z per point. Alternatively,
// the three components can be given individually, i.e. x y z as similar to
// rectilinear grids. These three components are the interleaved to obtain
// the right ordering. Finally, it is possible to provide only two out of
// three coordinates, e.g. x and z. In this case, the missing set of
// coordinates are filled with zeros.
func (h *Header) structuredPoints(xyz ...interface{}) error {
	if len(xyz) > 3 {
		msg := "Point data should be 1,2, or 3 dimensional, got: %d"
		return fmt.Errorf(msg, len(xyz))
	}

	lp, err := h.lastPiece()
	if err != nil {
		return err
	}
	if lp.Points != nil {
		return fmt.Errorf("Points allready set")
	}
	lp.Points = h.NewArray()

	// Flat data vector as (x0,y0,z0,x1,y1,z1...xn,yn,zn).
	if len(xyz) == 1 {
		n := reflect.ValueOf(xyz[0]).Len()
		if n != 3*lp.NumberOfPoints {
			msg := "Wrong number of values: exp: %d, got: %d"
			return fmt.Errorf(msg, 3*lp.NumberOfPoints, n)
		}
		return lp.Points.add("Points", 3, xyz[0])
	}

	// Interleave (x,y) or (x,y,z) data. For three-dimensional data it
	// interleaves x, y, z data directly, while for two-dimensional data
	// the empty dimension is filled with zeros. The empty dimension is
	// obtained by the domains extent.
	b := h.Grid.Extent
	dat, err := interleave(b.numPoints(), b.zeroDim(), xyz...)
	if err != nil {
		return err
	}
	return lp.Points.add("Points", 3, dat)
}

// unstructuredPoints adds a set of coordinates to the unstructured grid.
// difference from the rectilinearPoints or Coordinates as the number of
// points need to be inferred from the data, there is no extent that we
// can refer to
func (h *Header) unstructuredPoints(xyz ...interface{}) error {
	if len(xyz) > 3 {
		msg := "Point data should be 1,2, or 3 dimensional, got: %d"
		return fmt.Errorf(msg, len(xyz))
	}

	lp, err := h.lastPiece()
	if err != nil {
		return err
	}
	if lp.Points != nil {
		return fmt.Errorf("Points allready set")
	}
	lp.Points = h.NewArray()

	// Flat data vector as (x0,y0,z0,x1,y1,z1...xn,yn,zn).
	if len(xyz) == 1 {
		n := reflect.ValueOf(xyz[0]).Len()
		if n%3 > 0 {
			msg := "Length: %d does not distribute over 3 dimension"
			return fmt.Errorf(msg, n)
		}
		lp.NumberOfPoints = n / 3
		return lp.Points.add("Points", 3, xyz[0])
	}

	// Interleave (x,y,) or (x,y,z) data. For (x,y,) a zero is inserted
	// for the third dimension. Note: cannot distinguish the empty
	// dimension, therefore will fill x, y, and splice z with zeros.
	lp.NumberOfPoints = reflect.ValueOf(xyz[0]).Len()
	dat, err := interleave(lp.NumberOfPoints, len(xyz), xyz...)
	if err != nil {
		return err
	}
	return lp.Points.add("Points", 3, dat)
}

func Piece(opts ...func(p *partition) error) Option {
	return func(h *Header) error {
		p := new(partition)
		for _, opt := range opts {
			if err := opt(p); err != nil {
				return err
			}
		}
		h.Grid.Pieces = append(h.Grid.Pieces, *p)
		return nil
	}
}

// Data adds data to the file. When len(data) matched the number of points
// or cells, the data is written accordingly. For ambiguous cases, the
// function returns an error. The PointData and CellData calls should then be
// considered instead.
func Data(name string, data interface{}) Option {
	return func(h *Header) error {

		lp, err := h.lastPiece()
		if err != nil {
			return err
		}

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

// Cells sets the element connectivity of the cell in the unstructured grid.
// The cells are represented with three integer slices:
// - conn: points to coordinates (set by Points)
// - offsets: indicating starting point of each cell in conn
// - labels: element type labels for each cell (len(offsets)-1)
//
// The user can provide a map[int]int to map the provided labels to the
// corresponding VTK element types. This map is set by SetLabelType().
func Cells(conn, offset, labels []int) Option {
	return func(h *Header) error {
		lp, err := h.lastPiece()
		if err != nil {
			return err
		}

		if lp.Cells != nil {
			return fmt.Errorf("Connectivity already set")
		}
		lp.Cells = h.NewArray()

		// need to assert lengths probably...
		lp.NumberOfCells = len(labels)

		if err := lp.Cells.add("connectivity", 1, conn); err != nil {
			return err
		}

		if offset[0] == 0 {
			// the format does not require a leading zero
			offset = offset[1:]
		}
		if err := lp.Cells.add("offsets", 1, offset); err != nil {
			return err
		}

		labels, err := h.mapLabelToType(labels)
		if err != nil {
			return err
		}
		if err := lp.Cells.add("types", 1, labels); err != nil {
			return err
		}

		return nil
	}
}

// SetLabelType sets the labelType map in the header. The labelType is used
// in unstructured grids to map the user's provided element labels towards
// the internal labeling.
func SetLabelType(labelType map[int]int) Option {
	return func(h *Header) error {
		if labelType != nil {
			h.labelType = labelType
			return nil
		}
		return fmt.Errorf("Empty map labelType provided")
	}
}

// mapLabelToType maps the user's element labels towards the inter numbering
// by mapping the labels using the labelType set with SetLabelType.
//
// For an emtpy map, the function returns the unmodified, original labels.
func (h *Header) mapLabelToType(labels []int) ([]int, error) {
	if h.labelType == nil || len(labels) == 0 {
		return labels, nil
	}

	// writes a copy of the array, do not modify originals
	types := make([]int, len(labels))

	for i, label := range labels {
		t, ok := h.labelType[label]
		if !ok {
			return nil, fmt.Errorf("No map for label '%d'", label)
		}
		types[i] = t
	}
	return types, nil
}

func FieldData(name string, data []float64) Option {
	return func(h *Header) error {

		if h.Grid.Data == nil {
			h.Grid.Data = h.NewFieldArray()
		}

		return h.Grid.Data.add(name, len(data), data)
	}
}

// coordinates sets the coordinates for the rectilinear grid. The function
// accepts a variadic number of empty interfaces, however, we can only deal
// with (x, y), or (x, y, z) values. The first two being a two and
// three-dimensional version where the individual coordinates x, y, and possibly
// z are provided.
//
// The vectors are expected to have length nx, ny, nz respectively.
func (h *Header) coordinates(xyz ...interface{}) error {
	if h.Type != rectilinearGrid {
		return fmt.Errorf("Coordinates only apply to format %v",
			rectilinearGrid)
	}

	if len(xyz) == 0 || len(xyz) > 3 {
		msg := "Coordinates accepts 1 to 3 vectors, got: %d"
		return fmt.Errorf(msg, len(xyz))
	}

	lp, err := h.lastPiece()
	if err != nil {
		return err
	}
	if lp.Coordinates != nil {
		return fmt.Errorf("Coordinates already set")
	}
	lp.Coordinates = h.NewArray()

	dim := []string{"x", "y", "z"}

	for i, v := range xyz {

		// length data vs num points for dimension i
		l := reflect.ValueOf(v).Len()
		n := h.Grid.Extent[2*i+1] - h.Grid.Extent[2*i] + 1

		if l != n {
			msg := "Unexpected number of coordinates: %v, exp: %v"
			msg += " for dimension %s"
			return fmt.Errorf(msg, l, n, dim[i])
		}

		field := fmt.Sprintf("%s_coordinates", dim[i])
		err := lp.Coordinates.add(field, 1, v)
		if err != nil {
			return err
		}
	}
	return nil
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

// WholeExtent sets the extent of the Image, Rectilinear, or Structured grids.
// The extent requires a lower and upper value for each dimension, where a
// single dimension can be left empty, e.g. x1 - x0 == 0.
func WholeExtent(x0, x1, y0, y1, z0, z1 int) Option {
	f := func(h *Header) error {
		b, err := newBounds(x0, x1, y0, y1, z0, z1)
		if err != nil {
			return err
		}
		h.Grid.Extent = b
		return nil
	}
	return f
}

// Origin sets the origin of the VTK image, rectilinear, and structured grids.
func Origin(x, y, z float64) Option {
	return func(h *Header) error {
		h.Grid.Origin = fmt.Sprintf("%f %f %f", x, y, z)
		return nil
	}
}

// Spacing sets the spacing in x, y, z direction of the VTK image grids.
func Spacing(dx, dy, dz float64) Option {
	return func(h *Header) error {
		h.Grid.Spacing = fmt.Sprintf("%f %f %f", dx, dy, dz)
		return nil
	}
}

// Extent sets the part of the domain given in the current partition. This
// should be within WholeExtent.
func Extent(x0, x1, y0, y1, z0, z1 int) func(p *partition) error {
	f := func(p *partition) error {
		ext, err := newBounds(x0, x1, y0, y1, z0, z1)
		if err != nil {
			return err
		}
		p.Extent = ext

		// update dimensionality
		p.NumberOfCells = p.Extent.numCells()
		p.NumberOfPoints = p.Extent.numPoints()
		return nil
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

	// check essential properties that might break the format
	switch h.Type {
	case imageData:
		if h.Grid.Extent == (bounds{}) {
			msg := "%s has no or empty extent %#v"
			return fmt.Errorf(msg, h.Type, h.Grid.Extent)
		}
	}

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
	lp, err := h.lastPiece()
	if err != nil {
		return err
	}

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
	lp, err := h.lastPiece()
	if err != nil {
		return err
	}

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
func (h *Header) lastPiece() (*partition, error) {
	if len(h.Grid.Pieces) == 0 {
		switch h.Type {
		case imageData, rectilinearGrid, structuredGrid:
			b := h.Grid.Extent
			if b == (bounds{}) {
				msg := "%s has no or empty extent: %#v"
				return nil, fmt.Errorf(msg, h.Type, b)
			}
			h.Add(Piece(Extent(b[0], b[1], b[2], b[3], b[4], b[5])))
		default:
			h.Add(Piece())
		}
	}

	return &h.Grid.Pieces[len(h.Grid.Pieces)-1], nil
}

// Splice inserts the empty interface z in the slice of empty interfaces
// xyz at the given index idx. If the index is not inside the expected bounds,
// i.e. 0 <= idx < 3, the original slice of interfaces is returned.
// todo(max) there must be a more elegant way for this?
func splice(idx int, xyz []interface{}, z interface{}) []interface{} {
	s := make([]interface{}, 3)
	switch idx {
	case 0:
		s[0], s[1], s[2] = z, xyz[0], xyz[1]
	case 1:
		s[0], s[1], s[2] = xyz[0], z, xyz[1]
	case 2:
		s[0], s[1], s[2] = xyz[0], xyz[1], z
	default:
		return xyz
	}
	return s
}

// Interleave merges the xyz ...interface into a single slice. This is done
// to achieve the required ordering of: x0y0z0, x1y0z0, ... etc. The function
// requires the extent of the data to deterime the expected (and required)
// number of points. Both to allocate the expected result, as well as,
// to allocate any zeros that need to be inserted.
func interleave(np, zero int, xyz ...interface{}) (interface{}, error) {
	// ensure all components have equal length
	n := make([]int, len(xyz))
	for i, v := range xyz {
		n[i] = reflect.ValueOf(v).Len()
	}
	for _, v := range n {
		if v != np {
			msg := "Unequal component lengths: exp: %d got: %v"
			return nil, fmt.Errorf(msg, np, n)
		}
	}

	switch xyz[0].(type) {
	case []int:
		if len(xyz) == 2 {
			z := make([]int, np)
			xyz = splice(zero, xyz, z)
		}

		res := make([]int, np*len(xyz))

		for dim, x := range xyz {
			if _, ok := x.([]int); !ok {
				return nil, fmt.Errorf("Cannot cast %T to int", x)
			}
			for i, v := range x.([]int) {
				res[i*len(xyz)+dim] = v
			}
		}
		return res, nil
	case []float64:
		if len(xyz) == 2 {
			z := make([]float64, np)
			xyz = splice(zero, xyz, z)
		}

		res := make([]float64, np*len(xyz))

		for dim, x := range xyz {
			if _, ok := x.([]float64); !ok {
				return nil, fmt.Errorf("Cannot cast %T to float", x)
			}
			for i, v := range x.([]float64) {
				res[i*len(xyz)+dim] = v
			}
		}
		return res, nil
	default:
		msg := "interleave is not implemented for type '%T'"
		return nil, fmt.Errorf(msg, xyz[0])
	}
}
