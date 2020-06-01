package govtk

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PVD represent a ParaViewData (PVD) format. The PVD file contains a single
// Collection, which contains multiple data sets. Each data set has a reference
// to a VTK file that might live anywhere on disk. Additionally, the data set
// allows to attach a time step, a group, and a part identifier. Providing the
// time steps of these files will ensure proper visualisation in ParaView of
// transient data sets.
type PVD struct {
	XMLName xml.Name `xml:"VTKFile"`
	Type    string   `xml:"type,attr"`

	// Collection holds a data set, where each file is given by a dataSet.
	Collection []dataSet `xml:"Collection>DataSet"`

	// If no explicit file names are provided, all data files in the PVD
	// collection are written into this directory.
	dir string

	// FIXME replace by func that user can supply to set formatting
	// Default filename formattin string `file_%0d.%s`.
	filenameFormat string

	// If fullpath is true, the absolute path is stored inside the PVD
	// collection. By default only relative paths with respect to `pvd.dir`
	// are written.
	fullpath bool
}

// Initialise a PVD collection with options.
func NewPVD(opts ...PVDOption) (*PVD, error) {
	pvd := &PVD{
		Type:       "Collection",
		Collection: make([]dataSet, 0),
	}

	defaults := []PVDOption{
		Directory("."),
		SetFileFormat("file_%03d.%s"),
		RelativeFilenames(),
	}
	opts = append(defaults, opts...)

	for _, opt := range opts {
		if err := opt(pvd); err != nil {
			return nil, err
		}
	}
	return pvd, nil
}

// Options for the PVD collection.
type PVDOption func(pvd *PVD) error

// Dir returns the directory of the PVD collection.
func (pvd *PVD) Dir() string {
	return pvd.dir
}

// AbsoluteFilenames ensures the full pathnames are written in the PVD header.
func AbsoluteFilenames() PVDOption {
	return func(pvd *PVD) error {
		pvd.fullpath = true
		return nil
	}
}

// RelativeFilenames ensures only relative pathnames are written in PVD header.
func RelativeFilenames() PVDOption {
	return func(pvd *PVD) error {
		pvd.fullpath = false
		return nil
	}
}

// Directory sets the directory to store the files of the PVD collection.
func Directory(dir string) PVDOption {
	return func(pvd *PVD) error {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.Mkdir(dir, os.ModePerm)
			if err != nil {
				return err
			}
		}
		pvd.dir = dir
		return nil
	}
}

// SetFileFormat sets the formatting string for the automatic file naming. This
// requires a single %d for the file number and a %s for the extension.
func SetFileFormat(format string) PVDOption {
	return func(pvd *PVD) error {
		pvd.filenameFormat = format
		return nil
	}
}

// dataSet captures an XML document with properties for the PVD collection. The
// dataSet contains a link to its filepath, with corresponding properties as the
// time step, part identifier, and group identifier.
type dataSet struct {
	XMLName  xml.Name `xml:"DataSet"`
	TimeStep float64  `xml:"timestep,attr"`
	Group    string   `xml:"group,attr,omitempty"`
	Part     int      `xml:"part,attr"`
	Filename string   `xml:"file,attr"`

	writer io.Writer
}

// Len returns the number of files currently hold in the collection.
func (pvd *PVD) Len() int {
	return len(pvd.Collection)
}

// Option for the dataSet.
type DSOption func(d *dataSet) error

// File sets the filename for the current header. Note: the filename is joined
// with the base directory of the PVD file.
func Filename(filename string) DSOption {
	return func(d *dataSet) error {
		d.Filename = filename
		return nil
	}
}

// Writer assigns a preferred io.Writer to write this header of the PVD
// structure. Note: the user must ensure the file name this `io.Writer` writes
// to is identical to the path of `path.Join(PVD.dir, Filename)`.
// TODO: Filename appends to base directory of PVD, might be unclear interface.
func Writer(w io.Writer) DSOption {
	return func(d *dataSet) error {
		if w == nil {
			return fmt.Errorf("nil interface provided.")
		}
		d.writer = w
		return nil
	}
}

// Time sets the time step attribute to a header in the PVD structure.
func Time(time float64) DSOption {
	return func(d *dataSet) error {
		d.TimeStep = time
		return nil
	}
}

// Part sets the part attribute to a header in the PVD structure.
func Part(part int) DSOption {
	return func(d *dataSet) error {
		if part < 0 {
			return fmt.Errorf("Part cannot be negative: %d", part)
		}
		d.Part = part
		return nil
	}
}

// Group sets the group attribute to a header in the PVD structure.
func Group(group string) DSOption {
	return func(d *dataSet) error {
		d.Group = group
		return nil
	}
}

// PVD(Directory("/foo"))

// foo/
// 	main.pvd
// 	file_1.vts
// 	file_2.vts

// Add adds a header to the PVD structure. The user can provide multiple
// DSOption functions to set additional details of the provided header.
func (pvd *PVD) Add(h *Header, opts ...DSOption) error {
	d := dataSet{}

	var err error

	filename := fmt.Sprintf(pvd.filenameFormat, pvd.Len(), h.FileExtension())

	// FIXME: handle the paths in a nicer way.
	path := filename
	if pvd.fullpath {
		path, err = filepath.Abs(filepath.Join(pvd.Dir(), filename))
		if err != nil {
			return err
		}
	}

	// default settings
	defaults := []DSOption{
		Time(float64(pvd.Len())),
		Filename(path),
	}

	// apply defaults followed by user settings
	opts = append(defaults, opts...)
	for _, opt := range opts {
		if err := opt(&d); err != nil {
			return err
		}
	}

	// ensure the file has an extension matching the header format
	// TODO should we insert the extension ourself?
	if filepath.Ext(d.Filename) == "" {
		d.Filename += fmt.Sprintf(".%s", h.FileExtension())
	}

	// store the data set
	pvd.Collection = append(pvd.Collection, d)

	// if writer provided, we use it
	if d.writer != nil {
		return h.Write(d.writer)
	}

	// reset path to relative path only, including dir if required
	if !pvd.fullpath {
		path = filepath.Join(pvd.Dir(), filename)
	}

	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		return err
	}
	return h.Write(bufio.NewWriter(f))
}

// Write writes the PVD as encoded XML to the provided io.Writer.
func (pvd *PVD) Write(w io.Writer) error {
	_, err := w.Write([]byte(xml.Header))
	if err != nil {
		return err
	}
	return xml.NewEncoder(w).Encode(pvd)
}

// Save opens a file and writes the XML to file.
func (pvd *PVD) Save(filename string) error {
	if filepath.Ext(filename) == "" {
		filename += ".pvd"
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return pvd.Write(bufio.NewWriter(f))
}
