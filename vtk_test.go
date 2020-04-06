package govtk

import (
	"fmt"
	"os"
	"testing"
)

func TestExtentPresent(t *testing.T) {
	img, err := Image()
	if err != nil {
		t.Error(err)
	}
	if err := img.Save("test.vti"); err == nil {
		t.Error("Saving without providing extent should fail")
	}
	if err := img.Add(Data("test", []float64{1, 2, 3})); err == nil {
		t.Error("Adding data without extent should fail")
	}
}

// test if we can push different data types towards fielddata fields.
func TestFieldData(t *testing.T) {
	// todo this test is quite verbose, can be made better
	img, err := Image(WholeExtent(0, 1, 0, 1, 0, 1))
	if err != nil {
		t.Error(err)
	}
	name := "field"
	if err := img.Add(FieldData(name, []float64{1.0})); err != nil {
		t.Error(err)
	}
	if err := img.Add(FieldData(name, []float64{1.0})); err == nil {
		t.Error("Duplicate FieldData fields should return error")
	}
	if err := img.Add(FieldData("int", 1)); err != nil {
		t.Errorf("Cannot write int: %v", err)
	}
	if err := img.Add(FieldData("int32", int32(1))); err != nil {
		t.Error("Cannot write int32")
	}
	if err := img.Add(FieldData("float32", float32(1))); err != nil {
		t.Error("Cannot write float32")
	}
	if err := img.Add(FieldData("float64", float64(1))); err != nil {
		t.Error("Cannot write float64")
	}
	if err := img.Add(FieldData("[]float64", []float64{1, 2, 3})); err != nil {
		t.Error("Cannot write []float64")
	}
	if err := img.Add(FieldData("[]int32", []int32{1, 2, 3})); err != nil {
		t.Error("Cannot write []int32")
	}
	if err := img.Add(FieldData("[]int", []int{1, 2, 3})); err != nil {
		t.Error("Cannot write []int")
	}
	if err := img.Save("fd.vti"); err != nil {
		t.Error(err)
	}
}

func TestPreventDuplicateFieldNames(t *testing.T) {
	vtu, err := Image(WholeExtent(0, 1, 0, 1, 0, 1))
	if err != nil {
		t.Error(err)
	}

	// insert a field with name str
	str := "test_name"
	err = vtu.Add(Data(str, []int{1}))
	if err != nil {
		t.Error(err)
	}

	// expect error for adding the same field
	err = vtu.Add(Data(str, []int{1}))
	if err == nil {
		t.Errorf("no duplicates: %v", err)
	}
}

func TestNumCellsPoints(t *testing.T) {
	type pair struct {
		b      bounds
		nc, np int
	}
	pairs := []pair{
		pair{b: bounds{0, 1, 0, 1, 0, 1}, nc: 1, np: 2 * 2 * 2},
		pair{b: bounds{0, 1, 0, 0, 0, 1}, nc: 1, np: 2 * 2},
		pair{b: bounds{0, 4, 0, 4, 0, 4}, nc: 4 * 4 * 4, np: 5 * 5 * 5},
	}
	for _, p := range pairs {
		nc := p.b.numCells()
		if nc != p.nc {
			msg := "Wrong number of cells: got %v, exp %v"
			t.Errorf(msg, nc, p.nc)
		}

		np := p.b.numPoints()
		if np != p.np {
			msg := "Wrong number of points: got %v, exp %v"
			t.Errorf(msg, np, p.np)
		}
	}
}

// Test some valid and invalid bounds. Ensure invalid bounds return err.
func TestBounds(t *testing.T) {
	// should succeed
	vals := []bounds{
		bounds{0, 1, 0, 1, 0, 1},
		bounds{-1, 0, -1, 0, -1, 0},
		bounds{0, 0, 0, 1, 0, 1},
		bounds{0, 1, 0, 0, 0, 1},
		bounds{0, 1, 0, 1, 0, 0},
	}
	for _, v := range vals {
		b, err := newBounds(v[0], v[1], v[2], v[3], v[4], v[5])
		if err != nil {
			t.Errorf("Error processing bounds: %v", v)
		}
		for i, _ := range b {
			if b[i] != v[i] {
				t.Errorf("Wrong bounds, got: %v, exp: %v", b[i], v[i])
			}
		}
	}

	// should fail
	vals = []bounds{
		bounds{1, 0, 0, 1, 0, 1},
		bounds{0, 1, 1, 0, 0, 1},
		bounds{0, 1, 0, 1, 1, 0},
		bounds{1, 1, 1, 1, 1, 0},
		bounds{0, 1, 1, 1, 1, 1},
		bounds{1, 1, 0, 1, 1, 1},
		bounds{0, 0, 0, 0, 0, 0},
		bounds{-1, -1, -1, -1, -1, -1},
	}
	for _, v := range vals {
		_, err := newBounds(v[0], v[1], v[2], v[3], v[4], v[5])
		if err == nil {
			t.Errorf("Invalid bound has nil error %v", v)
		}
	}
}

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

func TestInterleave(t *testing.T) {
	x := []int{1, 4, 7}
	y := []int{2, 5, 8}
	z := []int{3, 6, 9}
	res := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

	tmp, err := interleave(3, 0, x, y, z)
	if err != nil {
		t.Error(err)
	}
	xyz := tmp.([]int)
	for i, _ := range res {
		if res[i] != xyz[i] {
			msg := "Not equal after interleaving: exp %v, got %v"
			t.Errorf(msg, res, xyz)
		}
	}

	// ensure we get an error for unequal lengths
	if _, err := interleave(3, 0, append(x, 1), y, z); err == nil {
		t.Error("Interleave with unequal arrays should return error")
	}
}

func TestUnstructured(t *testing.T) {
	coords := []float64{
		0.0, 0.0, 0.0,
		1.0, 0.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0, 1.0,
		-1.0, 0.0, 0.0,
		0.0, -1.0, 0.0,
		0.0, 0.0, -1.0}

	labelType := make(map[int]int)
	labelType[20] = Tetra

	vtu, err := Unstructured(Raw(), Compressed(), SetLabelType(labelType))
	if err != nil {
		t.Error(err)
	}

	if err := vtu.Add(Points(coords)); err != nil {
		t.Error(err)
	}

	conn := []int{0, 1, 2, 3, 0, 4, 5, 6}
	offset := []int{0, 4, 8}
	labels := []int{20, 20}

	if err := vtu.Add(Cells(conn, offset, labels)); err != nil {
		t.Error(err)
	}

	if err := vtu.Save("unstr.vtu"); err != nil {
		t.Error(err)
	}
}
