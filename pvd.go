package govtk

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

type PVD struct {
	XMLName    xml.Name  `xml:"VTKFile"`
	Type       string    `xml:"type,attr"`
	Collection []dataSet `xml:"Collection>DataSet"`
	dir        string    // where to store the subfiles
}

func NewPVD(dir string) (*PVD, error) {
	// create directory if not present
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	return &PVD{Type: "Collection", Collection: make([]dataSet, 0), dir: dir}, nil
}

type dataSet struct {
	XMLName  xml.Name `xml:"DataSet"`
	TimeStep float64  `xml:"timestep,attr"`
	Group    string   `xml:"group,attr,omitempty"`
	Part     int      `xml:"part,attr"`
	File     string   `xml:"file,attr"`
	//
	writer io.Writer
}

func (pvd *PVD) Len() int {
	return len(pvd.Collection)
}

type PVDOption func(d *dataSet) error

// File sets the filename for the current header. Note: the filename is joined
// with the base directory of the PVD file.
func File(filename string) PVDOption {
	return func(d *dataSet) error {
		d.File = filename
		return nil
	}
}

// Writer assigns a preferred io.Writer to write this header of the PVD
// structure. Note: the user must ensure the file name this `io.Writer` writes
// to is identical to the path of `path.Join(PVD.dir, Filename)`.
// TODO: Filename appends to base directory of PVD, might be unclear interface.
func Writer(w io.Writer) PVDOption {
	return func(d *dataSet) error {
		if w == nil {
			return fmt.Errorf("nil interpface provided.")
		}
		d.writer = w
		return nil
	}
}

// Time sets the time step attribute to a header in the PVD structure.
func Time(time float64) PVDOption {
	return func(d *dataSet) error {
		d.TimeStep = time
		return nil
	}
}

// Part sets the part attribute to a header in the PVD structure.
func Part(part int) PVDOption {
	return func(d *dataSet) error {
		if part < 0 {
			return fmt.Errorf("Part cannot be negative: %d", part)
		}
		d.Part = part
		return nil
	}
}

// Group sets the group attribute to a header in the PVD structure.
func Group(group string) PVDOption {
	return func(d *dataSet) error {
		d.Group = group
		return nil
	}
}

// Add adds a header to the PVD structure. The user can provide multiple
// PVDOption functions to set additional details of the provided header.
func (pvd *PVD) Add(h *Header, opts ...PVDOption) error {
	d := dataSet{}

	// default settings
	p := fmt.Sprintf("file_%d.%s", pvd.Len(), h.FileExtension())
	defaults := []PVDOption{
		Time(float64(pvd.Len())),
		File(p),
	}

	// apply defaults followed by user settings
	opts = append(defaults, opts...)
	for _, opt := range opts {
		if err := opt(&d); err != nil {
			return err
		}
	}

	// ensure the file has an extension matching the header format
	if filepath.Ext(d.File) == "" {
		d.File += fmt.Sprintf(".%s", h.FileExtension())
	}

	// store the data set
	pvd.Collection = append(pvd.Collection, d)

	// if writer provided, we use it
	if d.writer != nil {
		return h.Write(d.writer)
	}

	// otherwise, we create a buffered writer
	f, err := os.Create(path.Join(pvd.dir, d.File))
	defer f.Close()
	if err != nil {
		return err
	}
	return h.Write(bufio.NewWriter(f))
}

// Write writes the PVD as encoded XML to file.
func (p *PVD) Write(w io.Writer) error {
	_, err := w.Write([]byte(xml.Header))
	if err != nil {
		return err
	}
	return xml.NewEncoder(w).Encode(p)
}
