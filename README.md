# GoVTK
A Go package to write data to VTK XML files. 

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/maxvdkolk/govtk)
![Go](https://github.com/MaxvdKolk/govtk/workflows/Go/badge.svg)

*Note: still work in progress. Not all features are finished / implemented* 

The package supports a variety of VTK XML styles to be written, 
i.e. image data (.vti), rectilinear grids (.vtr), structured grids (.vts), and unstructured grids (*.vtu). 
Each format allows to write the XML using ascii, base64, or binary encoding. The 
data can be compressed using ```zlib``` by means of the Go standard library ```compress/zlib```

## Installation
Install the package
```
go get -u -v github.com/maxvdkolk/govtk/...
```
Which provides the `govtk` command as well as the package: 
```go 
import ("github.com/maxvdkolk/govtk")
```

## Usage
The document is available here: ```add link to docs```. 
Below follow some basic examples on usage of the package. Additionally,
examples are found at ```add link to example files```. 

In general, the package works by the following principle: first you create 
the VTK header by constructing a specific format, e.g. ```Image()```, 
```Structured()```, ```Unstructured()```. These setup the header with 
basic settings. All these functions take a set of parameters, e.g. 
```Ascii()```, ```Binary()```, or ```Base64()```. There are also 
format specific settings, such as ```Spacing()``` and ```Origin```, as
required by the image data. 

After construction, data is added through the ```Add()``` routine. This
allows a variable number of arguments to be processed. Finally, one can
close the file by writing to a name, ```Write(filename)```, or by 
saving to a provided ```io.Writer```, ```Save(io.Writer)```. 

### Image data 
```go 
im := govtk.Image(govtk.Binary(), govtk.Compressed(), 
                  govtk.Origin(0, 0, 0), govtk.Spacing(1, 1, 1))
im.Add(govtk.Data(...))
im.Save("file")
```

### Structured grid
```go
im := govtk.Structured(govtk.Binary(), govtk.Compressed()) 
im.Add(govtk.Points(...))
im.Add(govtk.Data(...))
im.Save("file") 
```

### Unstructured grid 
Simple example with a small mesh 
More complex example should live in the examples with custom mappings 


### Multi-block files 
```to be implemented```
Allows to combine multiple VTK XML files in a single file, i.e. a 
header file is created that points to a set of other VTK XML files, 
which actually hold the data. 

```go
mb := MultiBlock(Binary(), Compressed()) 
mb.AddFile(h...*Header) 
mb.Close()
```

### Paraview data file format (PVD)
```to be implemented```

### Parallel files PVTK
```to be implemented```

### Legacy format 
Besides the various forms of the XML VTK formats, the libary also has (minimal) support for the simple legacy VTK formats. 

### Options and settings 

## Command-line tools 
```to be implemented``` 

## References 
https://vtk.org/Wiki/VTK_XML_Formats
https://vtk.org/wp-content/uploads/2015/04/file-formats.pdf
