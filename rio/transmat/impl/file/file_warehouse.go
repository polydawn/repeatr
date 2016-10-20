package file

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
	"go.polydawn.net/repeatr/rio"
)

type Warehouse struct {
	coord def.WarehouseCoord // user's string retained for messages
	url   *url.URL
}

func NewWarehouse(coords rio.SiloURI) *Warehouse {
	// verify schema is sensible up front.
	u, err := url.Parse(string(coords))
	if err != nil {
		panic(&def.ErrConfigValidation{
			Msg: fmt.Sprintf("failed to parse URI: %s", err),
		})
	}
	switch u.Scheme {
	case "file+ca":
	case "file":
	case "http+ca":
	case "http":
	case "https+ca":
	case "https":
	case "":
		panic(&def.ErrConfigValidation{
			Msg: "missing scheme in warehouse URI; need a prefix, e.g. \"file://\" or \"http://\"",
		})
	default:
		panic(&def.ErrConfigValidation{
			Msg: fmt.Sprintf("unsupported scheme in warehouse URI: %q", u.Scheme),
		})
	}
	// stamp out a warehouse handle.
	wh := &Warehouse{
		coord: def.WarehouseCoord(coords),
		url:   u,
	}
	return wh
}

/*
	Check if the warehouse exists and can be contacted.

	Returns nil if contactable; if an error, the message will be
	an end-user-meaningful description of why the warehouse is out of reach.
*/
func (wh *Warehouse) Ping(writable bool) error {
	during := "fetch"
	if writable {
		during = "save"
	}

	u := wh.url
	switch u.Scheme {
	case "file+ca":
		pth := filepath.Join(u.Host, u.Path) // file uris don't have hosts
		stat, err := os.Stat(pth)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			return &def.ErrWarehouseUnavailable{
				Msg:    fmt.Sprintf("file+ca warehouse must be dir: %s is not a dir", pth),
				During: during,
				From:   wh.coord,
			}
		}
		return nil
	case "file":
		pth := filepath.Join(u.Host, u.Path) // file uris don't have hosts
		pth = filepath.Dir(pth)              // drop the last bit: the file need not exist yet if we're gonna write there
		stat, err := os.Stat(pth)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			return &def.ErrWarehouseUnavailable{
				Msg:    fmt.Sprintf("file warehouse must inside a dir: %s is not a dir", pth),
				During: during,
				From:   wh.coord,
			}
		}
		return nil
	case "http", "https":
		resp, err := http.Get(u.String())
		if err != nil {
			return &def.ErrWarehouseUnavailable{
				Msg:    fmt.Sprintf("could not dial http warehouse: %s", err),
				During: during,
				From:   wh.coord,
			}
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return &def.ErrWarehouseUnavailable{
				Msg:    fmt.Sprintf("could not dial http warehouse: status %s", resp.Status),
				During: during,
				From:   wh.coord,
			}
		}
		return nil
	case "http+ca", "https+ca":
		// CA-mode plain network warehouses are a little weird.  You're
		// certainly allowed to forbid listing the parent dir of wares,
		// which means it's pretty hard for us to tell if this is gonna go south.
		resp, err := http.Get(u.String())
		if err != nil {
			return &def.ErrWarehouseUnavailable{
				Msg:    fmt.Sprintf("could not dial http+ca warehouse: %s", err),
				During: during,
				From:   wh.coord,
			}
		}
		resp.Body.Close()
		// Ignore status code, per above reasoning about dir listings.
		return nil
	default:
		panic(meep.Meep(
			&meep.ErrProgrammer{},
			meep.Cause(fmt.Errorf("inconsistent validation")),
		))
	}
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
	case "file+ca":
		u.Path = filepath.Join(u.Path, string(dataHash))
		fallthrough
	case "file":
		u.Path = filepath.Join(u.Host, u.Path) // file uris don't have hosts
		file, err := os.OpenFile(u.Path, os.O_RDONLY, 0644)
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
	case "http+ca":
		u.Path = path.Join(u.Path, string(dataHash))
		u.Scheme = "http"
		fallthrough
	case "http":
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
	case "https+ca":
		u.Path = path.Join(u.Path, string(dataHash))
		u.Scheme = "https"
		fallthrough
	case "https":
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
