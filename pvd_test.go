package govtk

import (
	"os"
	"path/filepath"
	"testing"
)

// Test PVD and PVDOptions
// FIXME clean up the test definitions
func TestPVD(t *testing.T) {
	im_a, err := Image(WholeExtent(0, 5, 0, 5, 0, 5))
	if err != nil {
		t.Error(err)
	}
	im_b, err := Image(WholeExtent(0, 10, 0, 10, 0, 10))
	if err != nil {
		t.Error(err)
	}
	pvd, err := NewPVD()
	if err != nil {
		t.Error(err)
	}

	for _, file := range []*Header{im_a, im_b} {
		if err := pvd.Add(file); err != nil {
			t.Error(err)
		}
	}

	// ensure current path
	if pvd.Dir() != "." {
		t.Errorf("Wrong directory: got %v, exp %s", pvd.Dir(), ".")
	}

	// ensure expected filename
	names := []string{"file_000.vti", "file_001.vti"}
	for i, n := range names {
		if pvd.Collection[i].Filename != n {
			t.Errorf("Wrong filename: got %v, exp %s", pvd.Collection[i].Filename, n)
		}
	}

	// -- with directory
	pvd, err = NewPVD(Directory("./mypvd"))
	if err != nil {
		t.Error(err)
	}
	if pvd.Dir() != "./mypvd" {
		t.Errorf("Wrong directory: got %v, exp %s", pvd.Dir(), "./mypvd")
	}

	for _, file := range []*Header{im_a, im_b} {
		if err := pvd.Add(file); err != nil {
			t.Error(err)
		}
	}

	// ensure expected filename
	names = []string{"file_000.vti", "file_001.vti"}
	for i, n := range names {
		if pvd.Collection[i].Filename != n {
			t.Errorf("Wrong filename: got %v, exp %s", pvd.Collection[i].Filename, n)
		}
	}

	// -- with directory && formatting
	pvd, err = NewPVD(SetFileFormat("file_%d.%s"))
	if err != nil {
		t.Error(err)
	}
	for _, file := range []*Header{im_a, im_b} {
		if err := pvd.Add(file); err != nil {
			t.Error(err)
		}
	}

	// ensure expected filename
	names = []string{"file_0.vti", "file_1.vti"}
	for i, n := range names {
		if pvd.Collection[i].Filename != n {
			t.Errorf("Wrong filename: got %v, exp %s", pvd.Collection[i].Filename, n)
		}
	}

	// -- with absolute paths
	pvd, err = NewPVD(AbsoluteFilenames())
	if err != nil {
		t.Error(err)
	}
	for _, file := range []*Header{im_a, im_b} {
		if err := pvd.Add(file); err != nil {
			t.Error(err)
		}
	}

	// ensure expected filename
	names = []string{"file_000.vti", "file_001.vti"}
	for i, n := range names {
		n, err := filepath.Abs(n)
		if err != nil {
			t.Error(err)
		}
		if pvd.Collection[i].Filename != n {
			t.Errorf("Wrong filename: got %v, exp %s", pvd.Collection[i].Filename, n)
		}
	}

	// -- with absolute paths and directory
	pvd, err = NewPVD(AbsoluteFilenames(), Directory("./mypvd/"))
	if err != nil {
		t.Error(err)
	}
	for _, file := range []*Header{im_a, im_b} {
		if err := pvd.Add(file); err != nil {
			t.Error(err)
		}
	}

	// ensure expected filename
	names = []string{"mypvd/file_000.vti", "mypvd/file_001.vti"}
	for i, n := range names {
		n, err := filepath.Abs(n)
		if err != nil {
			t.Error(err)
		}
		if pvd.Collection[i].Filename != n {
			t.Errorf("Wrong filename: got %v, exp %s", pvd.Collection[i].Filename, n)
		}
	}

	f, err := os.Create(filepath.Join(pvd.Dir(), "mypvd.pvd"))
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
	if err := pvd.Write(f); err != nil {
		t.Error(err)
	}
}

// Ensure DSOptions has right effect on the dataSet.
func TestPVDFileProperties(t *testing.T) {
	im_a, err := Image(WholeExtent(0, 5, 0, 5, 0, 5))
	if err != nil {
		t.Error(err)
	}
	pvd, err := NewPVD()
	if err != nil {
		t.Error(err)
	}
	if err := pvd.Add(im_a, Time(0.0)); err != nil {
		t.Error(err)
	}
	if err := pvd.Add(im_a, Part(1)); err != nil {
		t.Error(err)
	}
	if err := pvd.Add(im_a, Group("group_one")); err != nil {
		t.Error(err)
	}
	if err := pvd.Add(im_a, Filename("myfile.file")); err != nil {
		t.Error(err)
	}
	if pvd.Collection[0].TimeStep != 0.0 {
		t.Errorf("Wrong time step: got %v, exp: %v", pvd.Collection[0].TimeStep, 0.0)
	}
	if pvd.Collection[1].Part != 1 {
		t.Errorf("Wrong part: got %v, exp %v", pvd.Collection[1].Part, 1)
	}
	if pvd.Collection[2].Group != "group_one" {
		t.Errorf("Wrong group: got %v, exp %v", pvd.Collection[2].Group, "group_one")
	}
	if pvd.Collection[3].Filename != "myfile.file" {
		t.Errorf("Wrong group: got %v, exp %v", pvd.Collection[3].Filename, "myfile.file")
	}
}

/*
func TestPVD_write(t *testing.T) {

	// combine two files
	nx, ny, nz := 5, 5, 5
	im_a, err := Image(WholeExtent(0, nx, 0, ny, 0, nz), Raw())
	if err != nil {
		t.Error(err)
	}

	tmp := make([]float64, (nx+1)*(ny+1)*(nz+1))
	cnt := 0
	for i := 0; i < nx+1; i++ {
		for j := 0; j < ny+1; j++ {
			for k := 0; k < nz+1; k++ {
				tmp[cnt] = float64(cnt)
				cnt++
			}
		}
	}
	if err := im_a.Add(Data("tmp", tmp)); err != nil {
		t.Error(err)
	}

	pvd, err := NewPVD(Directory("./pvd"))
	if err != nil {
		t.Error(err)
	}

	time := 0.0
	for i := 0; i < 10; i++ {
		if err := pvd.Add(im_a, Time(time), Part(i)); err != nil {
			t.Error(err)
		}
		time += float64(i)
	}

	f, err := os.Create("./pvd/mypvd.pvd")
	defer f.Close()
	if err != nil {
		t.Error(err)
	}
	if err := pvd.Write(f); err != nil {
		t.Error(err)
	}
}
*/
