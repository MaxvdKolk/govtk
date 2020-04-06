package govtk

import "testing"

func TestLegacyImage(t *testing.T) {
	img, err := Image(WholeExtent(0, 2, 0, 2, 0, 2), Raw())
	err = img.Add(Data("cellid", []int{0, 1, 2, 3, 4, 5, 6, 7}))
	img.Save("img.vti")
	img.Add(Legacy())
	if err != nil {
		t.Error(err)
	}
	if img.legacy != true {
		t.Errorf("Legacy should be set")
	}
	if err := img.Save("img.vtk"); err != nil {
		t.Error(err)
	}
}
