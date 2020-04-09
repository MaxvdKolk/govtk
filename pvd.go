package govtk

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
)

type PVD struct {
	XMLName    xml.Name  `xml:"VTKFile"`
	Type       string    `xml:"type,attr"`
	Collection []dataSet `xml:"Collection>DataSet"`
	directory  string    // where to store the subfiles
}

func NewPVD(dir string) (*PVD, error) {
	// create directory if not present
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	return &PVD{Type: "Collection", Collection: make([]dataSet, 0), directory: dir}, nil
}

type dataSet struct {
	XMLName  xml.Name `xml:"DataSet"`
	TimeStep float64  `xml:"timestep,attr"`
	Group    string   `xml:"group,attr"`
	Part     int      `xml:"part,attr"`
	File     string   `xml:"file,attr"`
}

func (p *PVD) Add(h *Header, time float64) error {
	// save reference
	d := dataSet{
		TimeStep: time,
		File:     path.Join(p.directory, fmt.Sprintf("ex_%d.vti", len(p.Collection))),
	}
	p.Collection = append(p.Collection, d)

	// store file
	return h.Save(d.File)
}

func (p *PVD) Write(w io.Writer) error {
	_, err := w.Write([]byte(xml.Header))
	if err != nil {
		return err
	}
	return xml.NewEncoder(w).Encode(p)
}
