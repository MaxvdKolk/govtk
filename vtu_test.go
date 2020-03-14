package vtu

import (
	"encoding/xml"
	"fmt"
	"os"
	"testing"
	//	"compress/zlib"
)

/* todo
- store provided data not directly as strings; convert at moment of writing
- tensor (?) field data
- base64 encoding per xml piece
- appended format
- binary (raw) encoding in appended format
- compression zlib for base64 and raw encoding
- test formats
*/

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
	if vtu.Appended.Encoding != "raw" {
		t.Errorf("Wrong appended data encoding: got %v exp %v",
			vtu.Appended.Encoding, "raw")
	}

	vtu, _ = Image(Appended(), Raw())
	if vtu.Appended.Encoding != "raw" {
		t.Errorf("Wrong appended data encoding: got %v exp %v",
			vtu.Appended.Encoding, "raw")
	}

	vtu, _ = Image(Raw(), Appended())
	if vtu.Appended.Encoding != "raw" {
		t.Errorf("Wrong appended data encoding: got %v exp %v",
			vtu.Appended.Encoding, "raw")
	}

	vtu, _ = Image(Appended(), Binary())
	if vtu.Appended.Encoding != "base64" {
		t.Errorf("Wrong appended data encoding: got %v exp %v",
			vtu.Appended.Encoding, "base64")
	}

	vtu, _ = Image(Binary(), Appended())
	if vtu.Appended.Encoding != "base64" {
		t.Errorf("Wrong appended data encoding: got %v exp %v",
			vtu.Appended.Encoding, "base64")
	}
}

func TestImage(t *testing.T) {

	nx, ny, nz := 20, 20, 20

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

	fmt.Println("asci...")
	// image file
	str := Image(asc...)
	//	str.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))
	str.Add(Data("C", coords), Data("B", coords))

	//str.Add(FieldData("F", []float64{1.0}))

	str.Save("im.vti")
	fmt.Println("done asci...")

	//bin := append(opts, Binary(), CompressedLevel(zlib.BestCompression), Appended())
	//bin := append(opts, Bin, Appended())ary(), Appended())
	bin := append(opts, Raw(), Compressed())
	//bin := append(opts, Binary(), Compressed())
	//bin := append(opts, Binary(), CompressedLevel(zlib.BestSpeed))
	fmt.Println("binary...")
	str = Image(bin...)
	str.Add(FieldData("F", []float64{1.0}))
	str.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))
	str.Add(Data("C", coords), Data("B", coords))
	str.Save("bim.vti")
	fmt.Println("done binary...")

	os.Exit(1)

	//bin = append(opts, Binary())
	bin = append(opts, Binary(), Compressed())
	//bin := append(opts, Binary(), Appended())
	//bin := append(opts, Binary(), Compressed())
	//bin := append(opts, Binary(), CompressedLevel(zlib.BestSpeed))
	str = Image(bin...)
	str.Add(FieldData("F", []float64{1.0}))
	str.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))
	str.Add(Data("C", coords), Data("B", coords))
	str.Save("bim2.vti")

	bin = append(opts, Raw(), Compressed())
	//bin := append(opts, Binary(), Appended())
	//bin := append(opts, Binary(), Compressed())
	//bin := append(opts, Binary(), CompressedLevel(zlib.BestSpeed))
	str = Image(bin...)
	str.Add(FieldData("F", []float64{1.0}))
	str.Add(FieldData("G", []float64{1.0, 2.0, 3.0}))
	str.Add(Data("C", coords), Data("B", coords))
	str.Save("bim3.vti")

	// rectilinear file
	str = Rectilinear(WholeExtent(0, nx, 0, ny, 0, nz), Ascii())
	str.Add(Coordinates(xc, yc, zc), PointData("C", coords))
	str.Save("im.vtr")

	// structured grid
	str = Structured(WholeExtent(0, nx, 0, ny, 0, nz), Ascii())
	str.Add(Points(coords), PointData("C", coords))
	str.Save("im.vts")

	t.Fail()
}

func TestVTU(t *testing.T) {
	coords := []float64{0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0}

	test := Unstructured(Raw(), Compressed())
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
