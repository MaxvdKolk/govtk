package govtk

import (
	"encoding/xml"
	"fmt"
	"os"
	"testing"
)

func TestAppendedData(t *testing.T) {
	vtu, _ := Image(Appended())

	if vtu.Appended == nil {
		t.Errorf("Nil pointer found at appended data.")
	}
	if vtu.Appended.XMLName.Local != "AppendedData" {
		t.Errorf("Wrong xml name for appended data array.")
	}

	// ascii + appended are not allowed together
	vtu, err := Image(Ascii(), Appended())
	if err == nil {
		t.Errorf("Appended and ascii should not be possible.")
	}
	vtu, err = Image(Appended(), Ascii())
	if err == nil {
		t.Errorf("Appended and ascii should not be possible.")
	}

	vtu, _ = Image(Raw())
	if vtu.Appended.Encoding != encodingRaw {
		t.Errorf("Wrong appended data encoding: got %v exp %v",
			vtu.Appended.Encoding, encodingRaw)
	}

	vtu, _ = Image(Appended(), Raw())
	if vtu.Appended.Encoding != encodingRaw {
		t.Errorf("Wrong appended data encoding: got %v exp %v",
			vtu.Appended.Encoding, encodingRaw)
	}

	vtu, _ = Image(Raw(), Appended())
	if vtu.Appended.Encoding != encodingRaw {
		t.Errorf("Wrong appended data encoding: got %v exp %v",
			vtu.Appended.Encoding, encodingRaw)
	}

	vtu, _ = Image(Appended(), Binary())
	if vtu.Appended.Encoding != encodingBase64 {
		t.Errorf("Wrong appended data encoding: got %v exp %v",
			vtu.Appended.Encoding, encodingBase64)
	}

	vtu, _ = Image(Binary(), Appended())
	if vtu.Appended.Encoding != encodingBase64 {
		t.Errorf("Wrong appended data encoding: got %v exp %v",
			vtu.Appended.Encoding, encodingBase64)
	}
}

func TestCompressionLevels(t *testing.T) {
	// ensure compressed level equal DefaultCompression
	vtu, _ := Image(Compressed())
	c, ok := vtu.compressor.(zlibCompression)
	if !ok {
		t.Errorf("Expected zlib compressor, got %T", vtu.compressor)
	}

	if c.level != DefaultCompression {
		t.Errorf("Expected default compression: %v got: %v",
			DefaultCompression, c.level)
	}

	// ensure level gets set
	vtu, _ = Image(CompressedLevel(BestSpeed))
	c, ok = vtu.compressor.(zlibCompression)
	if !ok {
		t.Errorf("Expected zlib compressor, got %T", vtu.compressor)
	}
	if c.level != BestSpeed {
		t.Errorf("Expected default compression: %v got: %v",
			BestSpeed, c.level)
	}

	// no compression should return a noCompressor instead
	vtu, _ = Image(CompressedLevel(NoCompression))
	_, ok = vtu.compressor.(noCompression)
	if !ok {
		t.Errorf("Expected no compressor, got %T", vtu.compressor)
	}
}

// Ensure image extent is written as expected and fails on wrong inputs.
func TestImageExtent(t *testing.T) {
	type pair struct {
		ext [6]int
		str string
	}

	// expected to succeed
	pairs := []pair{
		pair{[6]int{0, 5, 0, 10, 0, 15}, "0 5 0 10 0 15"},
		pair{[6]int{0, 0, 0, 10, 0, 15}, "0 0 0 10 0 15"},
		pair{[6]int{0, 1, 0, 1, 0, 0}, "0 1 0 1 0 0"},
		pair{[6]int{-1, 0, -1, 0, -1, 0}, "-1 0 -1 0 -1 0"},
	}
	for i, p := range pairs {
		opt := WholeExtent(p.ext[0], p.ext[1], p.ext[2], p.ext[3], p.ext[4], p.ext[5])
		im, err := Image(opt)
		if err != nil {
			t.Errorf("Cannot setup image with extent")
		}

		if fmt.Sprint(im.Grid.Extent) != p.str {
			t.Errorf("Wrong extent: got: %v, exp: %v", im.Grid.Extent, p.str)
		}

		if true {
			f, err := os.Create(fmt.Sprintf("im_ext_%d.vti", i))
			if err != nil {
				t.Errorf("cannot open file")
			}
			im.Write(f)
		}
	}

	// expected to fail
	pairs = []pair{
		pair{ext: [6]int{0, 1, 0, 0, 0, 0}, str: "incorrect dimension yz"},
		pair{ext: [6]int{0, 0, 0, 1, 0, 0}, str: "incorrect dimension xz"},
		pair{ext: [6]int{0, 0, 0, 0, 0, 1}, str: "incorrect dimension xy"},
		pair{ext: [6]int{1, 0, 1, 0, 1, 0}, str: "extend low - high values"},
	}
	for _, p := range pairs {
		opt := WholeExtent(p.ext[0], p.ext[1], p.ext[2], p.ext[3], p.ext[4], p.ext[5])
		_, err := Image(opt)
		if err == nil {
			t.Errorf("No error received for expected failed for '%v'", p.str)
		}
	}
}

func TestImageFormat(t *testing.T) {

	// bounds
	nx, ny, nz := 100, 100, 100

	// settings
	opts := make([]Option, 0, 0)
	opts = append(opts, WholeExtent(0, nx, 0, ny, 0, nz))
	opts = append(opts, Spacing(0.1, 0.1, 0.1))
	opts = append(opts, Origin(0, 0, 0))
	opts = append(opts, Raw(), CompressedLevel(NoCompression))

	// coordinates
	coords := make([]float64, 0, 0)
	xc := make([]float64, 0, 0)
	yc := make([]float64, 0, 0)
	zc := make([]float64, 0, 0)
	for k := 0; k < nx+1; k++ {
		for j := 0; j < ny+1; j++ {
			for i := 0; i < nz+1; i++ {
				coords = append(coords, float64(i))
				coords = append(coords, float64(j))
				coords = append(coords, float64(k))
				xc = append(xc, float64(i))
				yc = append(yc, float64(i))
				zc = append(zc, float64(i))
			}
		}
	}

	// cell data
	cdint := make([]int, nx*ny*nz)
	cdint32 := make([]int32, nx*ny*nz)
	cdfloat := make([]float64, nx*ny*nz)
	for i, _ := range cdint {
		cdint[i] = int(i)
		cdint32[i] = int32(i)
		cdfloat[i] = float64(i)
	}

	// assign data
	im, err := Image(opts...)
	if err != nil {
		t.Errorf("Problem setting options %v", err)
	}

	if err := im.Add(Data("C", coords)); err != nil {
		t.Errorf("Problem adding point data %v", err)
	}

	if err := im.Add(Data("B", coords)); err != nil {
		t.Errorf("Problem adding point data %v", err)
	}

	if err := im.Add(Data("cdi", cdint)); err != nil {
		t.Errorf("Problem adding int cell data %v", err)
	}

	if err := im.Add(Data("cdi32", cdint32)); err != nil {
		t.Errorf("Problem adding int cell data %v", err)
	}

	if err := im.Add(Data("cdf", cdfloat)); err != nil {
		t.Errorf("Problem adding float cell data %v", err)
	}

	im.Save("image.vti")
}

func TestImage(t *testing.T) {

	nx, ny, nz := 10, 10, 10

	coords := make([]float64, 0, 0)
	xc := make([]float64, 0, 0)
	yc := make([]float64, 0, 0)
	zc := make([]float64, 0, 0)
	for k := 0; k < nx+1; k++ {
		for j := 0; j < ny+1; j++ {
			for i := 0; i < nz+1; i++ {
				coords = append(coords, float64(i))
				coords = append(coords, float64(j))
				coords = append(coords, float64(k))

				//coords = append(coords, float64(1.0))
				//coords = append(coords, float64(1.0))
				//coords = append(coords, float64(1.0))

				xc = append(xc, float64(i))
				yc = append(yc, float64(i))
				zc = append(zc, float64(i))
			}
		}
	}

	opts := make([]Option, 0, 0)
	opts = append(opts, WholeExtent(0, nx, 0, ny, 0, nz))
	opts = append(opts, Spacing(0.1, 0.1, 0.1))
	opts = append(opts, Origin(0, 0, 0))

	asc := append(opts, Ascii())

	// image file
	str, _ := Image(asc...)
	//	str.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))
	str.Add(Data("C", coords), Data("B", coords))

	//str.Add(FieldData("F", []float64{1.0}))
	str.Save("im.vti")

	bin := append(opts, Binary())
	str, _ = Image(bin...)
	str.Add(FieldData("F", []float64{1.0}))
	str.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))
	str.Add(Data("C", coords), Data("B", coords))
	str.Save("binary.vti")

	bin = append(opts, Binary(), Appended())
	str, _ = Image(bin...)
	str.Add(FieldData("F", []float64{1.0}))
	str.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))
	str.Add(Data("C", coords), Data("B", coords))
	str.Save("binary_appended.vti")

	bin = append(opts, Binary(), Appended(), Compressed())
	str, _ = Image(bin...)
	str.Add(FieldData("F", []float64{1.0}))
	str.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))
	str.Add(Data("C", coords), Data("B", coords))
	str.Save("binary_appended_compressed.vti")

	bin = append(opts, Binary(), Compressed())
	str, _ = Image(bin...)
	str.Add(FieldData("F", []float64{1.0}))
	str.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))
	str.Add(Data("C", coords), Data("B", coords))
	str.Save("binary_compressed.vti")

	bin = append(opts, Raw())
	str, _ = Image(bin...)
	str.Add(FieldData("F", []float64{1.0}))
	str.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))
	str.Add(Data("C", coords), Data("B", coords))
	str.Save("binary_raw.vti")

	bin = append(opts, Raw(), Compressed())
	str, _ = Image(bin...)
	str.Add(FieldData("F", []float64{1.0}))
	str.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))
	str.Add(Data("C", coords), Data("B", coords))
	str.Save("binary_raw_compressed.vti")

	// rectilinear file
	//str = Rectilinear(WholeExtent(0, nx, 0, ny, 0, nz), Ascii())
	//str.Add(Coordinates(xc, yc, zc), PointData("C", coords))
	//str.Save("im.vtr")

	//// structured grid
	//str = Structured(WholeExtent(0, nx, 0, ny, 0, nz), Ascii())
	//str.Add(Points(coords), PointData("C", coords))
	//str.Save("im.vts")

	//t.Error()
}

func oldTestVTU(t *testing.T) {
	coords := []float64{0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0}

	test, _ := Unstructured(Raw(), Compressed())
	test.Add(FieldData("F", []float64{1.0}))
	test.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))

	test.Add(Piece())
	test.Add(Points(coords))
	conn := make([][]int, 1)
	conn[0] = []int{0, 1, 2, 3}

	test.Add(Cells(conn))

	test.Add(CellData("Test", []float64{1.0, 3.0}))
	test.Add(PointData("P", []float64{1.0, 3.0, 2.0, 4.0}))

	f, err := os.Create("env.vtu")
	if err != nil {
		panic("error")
	}
	defer f.Close()

	enc := xml.NewEncoder(f)
	err = enc.Encode(test)
	if err != nil {
		fmt.Println(err)
		panic("error")
	}

	test.Add(Piece())
	for i := 0; i < len(coords); i++ {
		coords[i] += 3.0
	}

	test.Add(Points(coords))
	test.Add(Cells(conn))

	test.Add(CellData("Test", []float64{2.0, 2.0}))
	test.Add(PointData("P", []float64{1.0, 3.0, 2.0, 4.0}))
	test.Add(FieldData("F", []float64{1.0}))

	test.Add(Piece())
	for i := 0; i < len(coords); i++ {
		coords[i] -= 6.0
	}
	test.Add(Points(coords))
	test.Add(Cells(conn))

	test.Add(CellData("Test", []float64{3.0, 1.0}))
	test.Add(PointData("P", []float64{1.0, 3.0, 2.0, 4.0}))
	test.Add(FieldData("F", []float64{1.0}))

	enc = xml.NewEncoder(os.Stdout)
	//enc.Indent("  ", "    ")
	//if err := enc.Encode(test); err != nil {
	//	fmt.Printf("error: %v\n", err)
	//}

	f, err = os.Create("env.vtu")
	if err != nil {
		panic("error")
	}
	defer f.Close()

	enc = xml.NewEncoder(f)
	err = enc.Encode(test)
	if err != nil {
		panic("error")
	}

	t.Fail()
}

/*
// some API calls

// consequetive points
points = [x, y, z, x, y, z] etc
// separate points
x = [x x x...]
y = [y y y...]
z = [z z z...]

// either the file is structured and we can use image data
vti := vtu.Image()
- domain bounding box, xmin, xmax etc
- number of points each dimension

// or the file is unstructred and we can completely unstructured data
vtu := vtu.Unstructured(np, nc)
- points, x, y, z, components
- cells + full connnectivity data
- offets, but maybe we can infer from connectivity?
- types, but maybe we can infer from connectivity?

// parallel files are just a wrapper on top of serial files, i.e. a header with
pointers to other files


// simple api for images
file := vtu.Image(origin, extend, spacing)
	file.AddPiece(extend)
		file.AddScalar(name, data)
			<DataArray>
		file.AddVector(name, data)
			<DataArray>
		file.AddTensor(name, data)
			<DataArray>
file.Save(filename)

// simple api for unstruc
file := vtu.Unstructred()

	file.AddPiece(npoints, ncells)
		file.Points(points)
		file.Connectivity(conn)
		file.AddScalar(name, data)
		file.AddVector(name, data)
		file.AddTensor(name, data)

	file.AddPiece(npoints, ncells)
		file.Points(points)
			<DataArray>
		file.Connectivity(conn)
			<DataArray>, <DataArray>, <DataArray>
		file.AddScalar(name, data)
			<DataArray>
		file.AddVector(name, data)
			<DataArray>
		file.AddTensor(name, data)
			<DataArray>

file.Save(filename)

*/
