package tar

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/rio"
)

type Warehouse struct {
	coord    def.WarehouseCoord // user's string retained for messages
	url      *url.URL
	ctntAddr bool
}

func NewWarehouse(coords rio.SiloURI) *Warehouse {
	// verify schema is sensible up front.
	u, err := url.Parse(string(coords))
	if err != nil {
		panic(&def.ErrConfigValidation{
			Msg: fmt.Sprintf("failed to parse URI: %s", err),
		})
	}
	// stamp out a warehouse handle.
	wh := &Warehouse{
		coord: def.WarehouseCoord(coords),
		url:   u,
	}
	// whitelist scheme types.
	switch u.Scheme {
	case "file":
	case "file+ca":
		wh.ctntAddr = true
		u.Scheme = "file"
	case "http":
	case "http+ca":
		wh.ctntAddr = true
		u.Scheme = "http"
	case "https":
	case "https+ca":
		wh.ctntAddr = true
		u.Scheme = "https"
	case "":
		panic(&def.ErrConfigValidation{
			Msg: "missing scheme in warehouse URI; need a prefix, e.g. \"file://\" or \"http://\"",
		})
	default:
		panic(&def.ErrConfigValidation{
			Msg: fmt.Sprintf("unsupported scheme in warehouse URI: %q", u.Scheme),
		})
	}
	return wh
}

/*
	Return a reader for the raw binary content of the ware.

	May panic with:

	  - `*def.ErrWareDNE` -- if the ware does not exist.
	  - `*def.ErrWarehouseProblem` -- for most other problems in fetch.
	  - Note that `*def.WarehouseUnavailableError` is *not* a valid panic here;
	    we have already pinged, so failure to answer now is considered a problem.
*/
func (wh *Warehouse) makeReader(dataHash rio.CommitID) io.ReadCloser {
	u := wh.url
	switch u.Scheme {
	case "file":
		pth := filepath.Join(u.Host, u.Path) // file uris don't have hosts
		if wh.ctntAddr {
			pth = filepath.Join(pth, string(dataHash))
		}
		file, err := os.OpenFile(pth, os.O_RDONLY, 0644)
		if err != nil {
			// Raise DNE for file-not-found; raise WarehouseProblem for anything less routine.
			if os.IsNotExist(err) {
				panic(&def.ErrWareDNE{
					Ware: def.Ware{Type: string(Kind), Hash: string(dataHash)},
					From: wh.coord,
				})
			} else {
				panic(&def.ErrWarehouseProblem{
					Msg:    err.Error(),
					During: "fetch",
					Ware:   def.Ware{Type: string(Kind), Hash: string(dataHash)},
					From:   wh.coord,
				})
			}
		}
		return file
	case "http":
		fallthrough
	case "https":
		if wh.ctntAddr {
			u, _ = url.Parse(u.String()) // copy
			u.Path = filepath.Join(u.Path, string(dataHash))
		}
		resp, err := http.Get(u.String())
		if err != nil {
			panic(&def.ErrWarehouseProblem{
				Msg:    err.Error(),
				During: "fetch",
				Ware:   def.Ware{Type: string(Kind), Hash: string(dataHash)},
				From:   wh.coord,
			})
		}
		switch resp.StatusCode {
		case 200:
			return resp.Body
		case 404:
			panic(&def.ErrWareDNE{
				Ware: def.Ware{Type: string(Kind), Hash: string(dataHash)},
				From: wh.coord,
			})
		default:
			panic(&def.ErrWarehouseProblem{
				Msg:    fmt.Sprintf("http status %s", resp.Status),
				During: "fetch",
				Ware:   def.Ware{Type: string(Kind), Hash: string(dataHash)},
				From:   wh.coord,
			})
		}
	default:
		panic(meep.Meep(
			&meep.ErrProgrammer{},
			meep.Cause(fmt.Errorf("inconsistent validation")),
		))
	}
}

/*
	Returns nil if the warehouse is expected to be writable;
	returns `*def.ErrWarehouseUnavailable` if not.
*/
func (wh *Warehouse) PingWritable() error {
	u := wh.url
	switch u.Scheme {
	case "file":
		pth := filepath.Join(u.Host, u.Path) // file uris don't have hosts
		if wh.ctntAddr {
			// In ctntAddr mode, the path must be a dir and be writable.
			stat, err := os.Stat(pth)
			if err != nil {
				return &def.ErrWarehouseUnavailable{
					Msg:    fmt.Sprintf("error pinging: %s", err),
					During: "save",
					From:   wh.coord,
				}
			}
			if !stat.IsDir() {
				return &def.ErrWarehouseUnavailable{
					Msg:    fmt.Sprintf("file+ca warehouse must be dir: %s is not a dir", pth),
					During: "save",
					From:   wh.coord,
				}
			}
			return nil
		} else {
			// In non-ctntAddr mode, the *parent* of path must be a dir,
			//  and if the path exists then it must be a file.
			parentPath := filepath.Dir(pth)
			stat, err := os.Stat(parentPath)
			if err != nil {
				return &def.ErrWarehouseUnavailable{
					Msg:    fmt.Sprintf("error pinging: %s", err),
					During: "save",
					From:   wh.coord,
				}
			}
			if !stat.IsDir() {
				return &def.ErrWarehouseUnavailable{
					Msg:    fmt.Sprintf("file warehouse must be within dir: %s is not a dir", parentPath),
					During: "save",
					From:   wh.coord,
				}
			}
			stat, err = os.Stat(pth)
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil {
				return &def.ErrWarehouseUnavailable{
					Msg:    fmt.Sprintf("error pinging: %s", err),
					During: "save",
					From:   wh.coord,
				}
			}
			if !stat.Mode().IsRegular() {
				return &def.ErrWarehouseUnavailable{
					Msg:    fmt.Sprintf("file warehouse cannot overwrite non-file types", parentPath),
					During: "save",
					From:   wh.coord,
				}
			}
			return nil
		}
	case "http":
		fallthrough
	case "https":
		return &def.ErrWarehouseUnavailable{
			Msg:    "warehouses accessed by http are not writable with this transmat",
			During: "save",
			From:   wh.coord,
		}
	default:
		panic(meep.Meep(
			&meep.ErrProgrammer{},
			meep.Cause(fmt.Errorf("inconsistent validation")),
		))
	}
}

/*
	For "file" coords, return the (local) path expected for a given piece of data.
*/
func (wh *Warehouse) getShelf(dataHash rio.CommitID) string {
	pth := filepath.Join(wh.url.Host, wh.url.Path) // file uris don't have hosts
	if wh.ctntAddr {
		return filepath.Join(pth, string(dataHash))
	} else {
		return pth
	}
}

type writeController struct {
	warehouse     *Warehouse
	writer        io.WriteCloser
	stageFilePath string
}

func (wh *Warehouse) openWriter() *writeController {
	wc := &writeController{warehouse: wh}
	wc.openStageFile()
	return wc
}

func (wc *writeController) openStageFile() {
	u := wc.warehouse.url
	switch u.Scheme {
	case "file":
		// Pick a random upload path
		pth := filepath.Join(u.Host, u.Path) // file uris don't have hosts
		if wc.warehouse.ctntAddr {
			wc.stageFilePath = filepath.Join(pth, ".tmp.upload."+guid.New())
		} else {
			wc.stageFilePath = filepath.Join(path.Dir(pth), ".tmp.upload."+path.Base(pth)+"."+guid.New())
		}
		// Open file to shovel data into
		file, err := os.OpenFile(wc.stageFilePath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(&def.ErrWarehouseProblem{
				Msg:    fmt.Sprintf("failed to reserve temp space in warehouse: %s", err),
				During: "save",
				From:   wc.warehouse.coord,
			})
		}
		wc.writer = file
	case "http":
		fallthrough
	case "https":
		panic(&def.ErrConfigValidation{
			Msg: "http transports are only supported for read-only use",
		})
	default:
		panic(meep.Meep(
			&meep.ErrProgrammer{},
			meep.Cause(fmt.Errorf("inconsistent validation")),
		))
	}
}

/*
	Commit the current data as the given hash.
	Caller must be an adult and specify the hash truthfully.
	Closes the writer and invalidates any future use.

	May panic with:

	  - `*def.ErrWarehouseProblem` -- in the event of IO errors committing.
*/
func (wc *writeController) Commit(saveAs rio.CommitID) {
	wc.writer.Close()
	finalPath := wc.warehouse.getShelf(saveAs)
	if err := os.Rename(wc.stageFilePath, finalPath); err != nil {
		panic(&def.ErrWarehouseProblem{
			Msg:    fmt.Sprintf("failed to commit: %s", err),
			During: "save",
			Ware:   def.Ware{Type: string(Kind), Hash: string(saveAs)},
			From:   wc.warehouse.coord,
		})
	}
}
