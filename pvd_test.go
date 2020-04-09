package govtk

import (
	"os"
	"testing"
)

func TestPVD(t *testing.T) {

	// combine two files
	im_a, err := Image(WholeExtent(0, 5, 0, 5, 0, 5))
	if err != nil {
		t.Error(err)
	}
	im_b, err := Image(WholeExtent(0, 5, 0, 5, 0, 5))
	if err != nil {
		t.Error(err)
	}

	im_a.Save("vtm_a.vti")
	im_b.Save("vtm_b.vti")

	pvd, err := NewPVD("./pvd")
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 10; i++ {
		if err := pvd.Add(im_a, 0.1*float64(i)); err != nil {
			t.Error(err)
		}
	}

	f, err := os.Create("mypvd.pvd")
	defer f.Close()
	if err != nil {
		t.Error(err)
	}
	if err := pvd.Write(f); err != nil {
		t.Error(err)
	}
}
