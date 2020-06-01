# GoVTK
A Go package to write data to VTK XML files. 

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/maxvdkolk/govtk)
![Go](https://github.com/MaxvdKolk/govtk/workflows/Go/badge.svg)

*Note: still work in progress. Not all features are finished / implemented* 

The package supports a variety of VTK XML styles to be written, 
i.e. image data (.vti), rectilinear grids (.vtr), structured grids (.vts), and unstructured grids (*.vtu). 
Each format allows to write the XML using ascii, base64, or binary encoding. The 
data can be compressed using ```zlib``` by means of the Go standard library ```compress/zlib```. 

## Usage
Four different formats are supported by their constructor: 
- `Image()` for `.vti` files 
- `Rectilinear()` for `.vtr` files 
- `Structured()` for `.vts` files 
- `Unstructured()` for `.vtu` files

These support three types of encoding: `Ascii()`, 
`Base64()`, and `Raw()` (binary data). The latter two result 
in significantly smaller files. Note, `Base64()` does maintain
valid XML, while `Raw()` does not. 
Additionally, `Base64()` and `Raw()` encoding allow 
to compress the data before encoding by passing `Compressed()`,
to further reduces file size. 

Data is added by `Add()`. Point/Cell data can be inserted as 
`Add(Data("fieldname", data))`, which infers if the data should
be place on the points or cells. Alternatively, the `CellData`
and `PointData` allow to directly write data to either 
points or cells. Global, generic data fields can be 
added by `FieldData`. 

The file is written to disk by 
either `Save(filename string)` or `Write(w io.Writer)`. 
For each file type a small example is presented. 

### Image data 
```go               
vti, err := govtk.Image(govtk.WholeExtend(0, nx, 0, ny, 0, nz))
data := make([]float64, (nx+1)*(ny+1)*(nz+1))
vti.Add(govtk.Data("fieldname", data))
vti.Save("image.vti") 
```

### Rectilinear 
```go
vtr, err := govtk.Rectilinear(govtk.WholeExtend(0, nx, 0, ny, 0, nz)) 

// store x, y, z coordinates a 1D vectors 
x := make([]float64, nx+1)
y := make([]float64, ny+1)
z := make([]float64, nz+1)

// fill x, y, z accordingly 
// ... 

// store coordinates
vtr.Add(govtk.Points(x, y, z)) 
vtr.Add(govtk.Data(...)) 
vtr.Save("rectilinear.vtr") 
```

### Structured grid
```go
vts, err := govtk.Structured(govtk.WholeExtend(0, nx, 0, ny, 0, nz)) 

// store all x,y,z tuples for each node
xyz := make([]float64, 3*(nx+1)*(ny+1)*(nz+1))

// fill xyz with each coordinate 
// ... 

vts.Add(govtk.Points(xyz))
vts.Add(govtk.Data(...))
vts.Save("structured.vts") 
```

### Unstructured grid 
```go
vtu, err := govtk.Unstructured() 

// store all x,y,z tuples for each node 
vtu.Add(Points(...))

// provide cell connectivity, e.g. two tet elements 
conn := []int{0, 1, 2, 3, 0, 4, 5, 6}
offset := []int{0, 4, 6} // start of each element in conn 
labels := []int{20, 20} // VTK labels 

vtu.Add(govtk.Cells(conn, offset, labels))

// store data and save 
vtu.Add(govtk.Data(...))
vtk.Save("unstructured.vtu" 
```

## Paraview data file format (PVD)
The package also allows to write `PVD` collections. These 
[ParaviewData](https://www.paraview.org/Wiki/ParaView/Data_formats#PVD_File_Format) 
files allow to store references to multiple VTK XML files, 
which might be located anywhere on disk. Each file can save a
time step, part id, and group name as attribute. The time steps
are directly available in ParaView, which is beneficial for data of
transient time series. 

For example to store a time series: 
```go

pvd, err := govtk.NewPVD(govtk.Directory("./mypvd")) 

// some time series 
for t := 0; t < 100; t++ {

    // generate output data 
    vti, err := govtk.Image(govtk.WholeExtend(0, nx, 0, ny, 0, nz), govkt.Raw()) 
    vti.Add(govtk.Data(...))
    
    // attach to PVD collection
    pvd.Add(vti, govtk.Time(float64(t))
}

// save PVD main file
pvd.Save(filepath.Join(pvd.Dir(), "pvd.pvd"))
```
Provides the following directory
```
myvpd/
    file_000.vti 
    file_001.vti
    ...
    file_099.vti
    pvd.pvd
```

## Legacy format 
*Not yet supported* 

## Encoding and compression settings 
Basic encoding and compression is controlled by: 
```go 
govtk.Ascii()     // plain ascii 
govtk.Binary()    // base64 encoding 
govtk.Raw()       // plain binary 

// compression only applies to `govtk.Binary()` and `govtk.Raw()`
govtk.Compressed()               // applies govtk.DefaultCompression
govtk.CompressedLevel(level int) // applies received compression level 

// compression levels are directly taken from `compress/zlib`
const ( 
    govtk.NoCompression      = zlib.NoCompression 
    govtk.BestSpeed          = zlib.BestSpeed
    govtk.BestCompression    = zlib.BestCompression
    govtk.DefaultCompression = zlib.DefaultCompression
    govtk.HuffmanOnly        = zlib.HuffmanOnly
) 
```

## Command-line tools 
*to be implemented*

## Installation
Install the package
```
go get -u -v github.com/maxvdkolk/govtk/...
```

## References 
https://vtk.org/Wiki/VTK_XML_Formats
https://vtk.org/wp-content/uploads/2015/04/file-formats.pdf
https://github.com/jipolanco/WriteVTK.jl
