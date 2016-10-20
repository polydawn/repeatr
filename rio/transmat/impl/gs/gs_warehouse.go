package gs

import (
	"fmt"
	"io"
	"net/url"
	"path"
	"path/filepath"

	"golang.org/x/oauth2"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/rio"
)

type Warehouse struct {
	coord      def.WarehouseCoord // user's string retained for messages
	bucketName string             // s3 bucket name
	pathPrefix string             // s3 path prefix
	token      *oauth2.Token
	ctntAddr   bool
}

func NewWarehouse(coords rio.SiloURI, token *oauth2.Token) *Warehouse {
	// verify schema is sensible up front.
	u, err := url.Parse(string(coords))
	if err != nil {
		panic(&def.ErrConfigValidation{
			Msg: fmt.Sprintf("failed to parse URI: %s", err),
		})
	}
	// stamp out a warehouse handle.
	wh := &Warehouse{
		coord:      def.WarehouseCoord(coords),
		bucketName: u.Host,
		pathPrefix: u.Path,
		token:      token,
	}
	// whitelist scheme types.
	switch u.Scheme {
	case "gs":
	case "gs+ca":
		wh.ctntAddr = true
	case "":
		panic(&def.ErrConfigValidation{
			Msg: "missing scheme in warehouse URI; need a prefix, e.g. \"gs://\" or \"gs+ca://\"",
		})
	default:
		panic(&def.ErrConfigValidation{
			Msg: fmt.Sprintf("unsupported scheme in warehouse URI: %q", u.Scheme),
		})
	}
	return wh
}

/*
	Returns nil if the warehouse is expected to be readable;
	returns `*def.ErrWarehouseUnavailable` if not.

	This is implementation lacks a way to tell if a warehouse is available
	before actually trying the operation, so this method is
	stubbed to always return success.
*/
func (wh *Warehouse) PingReadable() error {
	// TODO there may be some way to ask if the root exists, but test fixtures
	//  are needed to suss out whether that's distinguishable from no-auth, etc.
	return nil
}

/*
	Returns nil if the warehouse is expected to be writable;
	returns `*def.ErrWarehouseUnavailable` if not.

	This is implementation lacks a way to tell if a warehouse is available
	before actually trying the operation, so this method is
	stubbed to always return success.
*/
func (wh *Warehouse) PingWritable() error {
	return nil
}

/*
	Return the path inside a bucket expected for a given piece of data.
*/
func (wh *Warehouse) getShelf(dataHash rio.CommitID) string {
	if wh.ctntAddr {
		return filepath.Join(wh.pathPrefix, string(dataHash))
	}
	return wh.pathPrefix
}

/*
	Return a temp path (has random suffixes to avoid collisions) for uploads.
	The path will be a sibling of the final destination.
*/
func (wh *Warehouse) getStageShelf() string {
	if wh.ctntAddr {
		return filepath.Join(wh.pathPrefix, ".tmp.upload."+guid.New())
	}
	return filepath.Join(path.Dir(wh.pathPrefix), ".tmp.upload."+path.Base(wh.pathPrefix)+"."+guid.New())
}

/*
	Return a reader for the raw binary content of the ware.

	May panic with:

	  - `*def.ErrWareDNE` -- if the ware does not exist.
	  - `*def.ErrWarehouseProblem` -- for most other problems in fetch.
	  - Note that `*def.WarehouseUnavailableError` is *not* a valid panic here;
	    we have already pinged, so failure to answer now is considered a problem.
*/
func (wh *Warehouse) openReader(dataHash rio.CommitID) io.ReadCloser {
	service, err := makeGsObjectService(wh.token)
	if err != nil {
		panic(&def.ErrWarehouseProblem{
			Msg:    err.Error(),
			During: "fetch",
			Ware:   def.Ware{Type: string(Kind), Hash: string(dataHash)},
			From:   wh.coord,
		})
	}
	response, err := service.Get(wh.bucketName, wh.getShelf(dataHash)).Download()
	if err == nil {
		return response.Body
	}
	panic(&def.ErrWarehouseProblem{
		Msg:    err.Error(),
		During: "fetch",
		Ware:   def.Ware{Type: string(Kind), Hash: string(dataHash)},
		From:   wh.coord,
	})
}

type writeController struct {
	warehouse *Warehouse
	writer    io.WriteCloser
	writerErr <-chan error
	stagePath string // may be empty if !ctntAddr -- we upload in place because obj writes are already atomic in s3.
}

func (wh *Warehouse) openWriter() *writeController {
	wc := &writeController{
		warehouse: wh,
	}
	// Pick upload path.
	var pth string
	if wh.ctntAddr {
		pth = wh.getStageShelf()
		wc.stagePath = pth
	} else {
		pth = wh.pathPrefix
	}
	// Open writer.
	//  We end up getting an error chan back as well, because the api underneath
	//   actually takes a reader, and we bounce things through a pipe with a goroutine
	//   so that we can get the same directionality of interface as everyone else.
	makeGsWriter(wh.bucketName, pth, wh.token)
	return wc
}

/*
	Commit the current data as the given hash.
	Caller must be an adult and specify the hash truthfully.
	Closes the writer and invalidates any future use.

	May panic with:

	  - `*def.ErrWarehouseProblem` -- in the event of IO errors committing.
*/
func (wc *writeController) Commit(saveAs rio.CommitID) {
	// flush and check errors on the final write to.
	// be advised that this close method does *a lot* of work aside from connection termination.
	// also calling it twice causes the library to wigg out and delete things, i don't even.
	if err := wc.writer.Close(); err != nil {
		panic(&def.ErrWarehouseProblem{
			Msg:    fmt.Sprintf("failed to commit: %s", err),
			During: "save",
			Ware:   def.Ware{Type: string(Kind), Hash: string(saveAs)},
			From:   wc.warehouse.coord,
		})
	}
	// gather any error from the routine flushing our pipe through.
	for err := range wc.writerErr {
		panic(&def.ErrWarehouseProblem{
			Msg:    fmt.Sprintf("failed to commit: %s", err),
			During: "save",
			Ware:   def.Ware{Type: string(Kind), Hash: string(saveAs)},
			From:   wc.warehouse.coord,
		})
	}

	// if a stating path was used, relocate the temp object to the real path.
	if wc.stagePath != "" {
		finalPath := wc.warehouse.getShelf(saveAs)
		reloc(wc.warehouse.bucketName, wc.stagePath, finalPath, wc.warehouse.token)
	}
}
