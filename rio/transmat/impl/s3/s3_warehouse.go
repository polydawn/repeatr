package s3

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"time"

	"github.com/rlmcpherson/s3gof3r"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/rio"
)

var s3Conf = &s3gof3r.Config{
	Concurrency: 10,
	PartSize:    20 * 1024 * 1024,
	NTry:        10,
	Md5Check:    false,
	Scheme:      "https",
	Client:      s3gof3r.ClientWithTimeout(15 * time.Second),
}

type Warehouse struct {
	coord      def.WarehouseCoord // user's string retained for messages
	bucketName string             // s3 bucket name
	pathPrefix string             // s3 path prefix
	keys       s3gof3r.Keys
	ctntAddr   bool
}

func NewWarehouse(coords rio.SiloURI, keys s3gof3r.Keys) *Warehouse {
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
		keys:       keys,
	}
	// whitelist scheme types.
	switch u.Scheme {
	case "s3":
	case "s3+ca":
		wh.ctntAddr = true
	case "":
		panic(&def.ErrConfigValidation{
			Msg: "missing scheme in warehouse URI; need a prefix, e.g. \"s3://\" or \"s3+ca://\"",
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
	s3 := s3gof3r.New("s3.amazonaws.com", wh.keys)
	r, _, err := s3.Bucket(wh.bucketName).GetReader(wh.getShelf(dataHash), s3Conf)
	if err == nil {
		return r
	}
	if err2, ok := err.(*s3gof3r.RespError); ok && err2.Code == "NoSuchKey" {
		panic(&def.ErrWareDNE{
			Ware: def.Ware{Type: string(Kind), Hash: string(dataHash)},
			From: wh.coord,
		})
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
	stagePath string // may be empty if !ctntAddr -- we upload in place because obj writes are already atomic in s3.
}

func (wh *Warehouse) openWriter() *writeController {
	wc := &writeController{
		warehouse: wh,
	}
	// Pick upload path.
	//  (We may not need a stage path at all, in which case it's preferable to
	//   avoid using one, since there is no reasonably cheap rename op.)
	var pth string
	if wh.ctntAddr {
		pth = wh.getStageShelf()
		wc.stagePath = pth
	} else {
		pth = wh.pathPrefix
	}
	// Open writer.
	s3 := s3gof3r.New("s3.amazonaws.com", wh.keys)
	w, err := s3.Bucket(wh.bucketName).PutWriter(pth, nil, s3Conf)
	if err != nil {
		panic(&def.ErrWarehouseProblem{
			Msg:    err.Error(),
			During: "save",
			From:   wh.coord,
		})
	}
	wc.writer = w
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
	// flush and check errors on the final write to s3.
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

	// if a stating path was used, relocate the temp object to the real path.
	if wc.stagePath != "" {
		finalPath := wc.warehouse.getShelf(saveAs)
		reloc(wc.warehouse.bucketName, wc.stagePath, finalPath, wc.warehouse.keys)
	}
}

// as close as we can get to `mv` on an s3 object.
func reloc(bucketName, oldPath, newPath string, keys s3gof3r.Keys) error {
	s3 := s3gof3r.New("s3.amazonaws.com", keys)
	bucket := s3.Bucket(bucketName)
	// this is a POST at the bottom, and copies are a PUT.  whee.
	//w, err := s3.Bucket(bucketName).PutWriter(newPath, copyInstruction, s3Conf)
	// So, implement our own aws copy API.
	req, err := http.NewRequest("PUT", "", &bytes.Buffer{})
	if err != nil {
		return err
	}
	req.URL.Scheme = s3Conf.Scheme
	req.URL.Host = fmt.Sprintf("%s.%s", bucketName, s3.Domain)
	req.URL.Path = path.Clean(fmt.Sprintf("/%s", newPath))
	// Communicate the copy source object with a header.
	// Be advised that if this object doesn't exist, amazon reports that as a 404... yes, a 404 that has nothing to do with the query URI.
	req.Header.Add("x-amz-copy-source", path.Join("/", bucketName, oldPath))
	bucket.Sign(req)
	resp, err := s3Conf.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return newRespError(resp)
	}
	// delete previous location
	if err := bucket.Delete(oldPath); err != nil {
		return err
	}
	return nil
}

// matches unexported helper inside s3gof3r; we need it because we had to implement a custom api method.
func newRespError(r *http.Response) *s3gof3r.RespError {
	e := new(s3gof3r.RespError)
	e.StatusCode = r.StatusCode
	b, _ := ioutil.ReadAll(r.Body)
	xml.NewDecoder(bytes.NewReader(b)).Decode(e) // parse error from response
	r.Body.Close()
	return e
}
