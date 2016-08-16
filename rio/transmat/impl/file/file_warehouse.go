package file

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"go.polydawn.net/repeatr/rio"
)

type Warehouse struct {
	coords *url.URL
}

func NewWarehouse(coords rio.SiloURI) *Warehouse {
	// verify schema is sensible up front.
	u, err := url.Parse(string(coords))
	if err != nil {
		panic(rio.ConfigError.New("could not parse warehouse URI: %s", err))
	}
	switch u.Scheme {
	case "file+ca":
	case "file":
	case "http+ca":
	case "http":
	case "https+ca":
	case "https":
	case "":
		panic(rio.ConfigError.New("missing scheme in warehouse URI; need a prefix, e.g. \"file://\" or \"http://\""))
	default:
		panic(rio.ConfigError.New("unsupported scheme in warehouse URI: %q", u.Scheme))
	}
	// stamp out a warehouse handle.
	wh := &Warehouse{u}
	return wh
}

/*
	Check if the warehouse exists and can be contacted.

	Returns nil if contactable; if an error, the message will be
	an end-user-meaningful description of why the warehouse is out of reach.
*/
func (wh *Warehouse) Ping() error {
	u := wh.coords
	switch u.Scheme {
	case "file+ca":
		pth := filepath.Join(u.Host, u.Path) // file uris don't have hosts
		stat, err := os.Stat(pth)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			return rio.WarehouseUnavailableError.New("file+ca warehouse must be dir: %s is not a dir", pth)
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
			return rio.WarehouseUnavailableError.New("file warehouse must inside a dir: %s is not a dir", pth)
		}
		return nil
	case "http", "https":
		resp, err := http.Get(u.String())
		if err != nil {
			return rio.WarehouseUnavailableError.New("could not dial http warehouse: %s", err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return rio.WarehouseUnavailableError.New("could not dial http warehouse: %s", err)
		}
		return nil
	case "http+ca", "https+ca":
		// CA-mode plain network warehouses are a little weird.  You're
		// certainly allowed to forbid listing the parent dir of wares,
		// which means it's pretty hard for us to tell if this is gonna go south.
		resp, err := http.Get(u.String())
		if err != nil {
			return rio.WarehouseUnavailableError.New("could not dial http+ca warehouse: %s", err)
		}
		resp.Body.Close()
		// Ignore status code, per above reasoning about dir listings.
		return nil
	case "":
		panic(rio.ConfigError.New("missing scheme in warehouse URI; need a prefix, e.g. \"file://\" or \"http://\""))
	default:
		panic(rio.ConfigError.New("unsupported scheme in warehouse URI: %q", u.Scheme))
	}
}

func (wh *Warehouse) makeReader(dataHash rio.CommitID) io.ReadCloser {
	u := wh.coords
	switch u.Scheme {
	case "file+ca":
		u.Path = filepath.Join(u.Path, string(dataHash))
		fallthrough
	case "file":
		u.Path = filepath.Join(u.Host, u.Path) // file uris don't have hosts
		file, err := os.OpenFile(u.Path, os.O_RDONLY, 0644)
		if err != nil {
			if os.IsNotExist(err) {
				panic(rio.DataDNE.New("Unable to read %q: %s", u.String(), err))
			} else {
				panic(rio.WarehouseUnavailableError.New("Unable to read %q: %s", u.String(), err))
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
			panic(rio.WarehouseUnavailableError.New("Unable to fetch %q: %s", u.String(), err))
		}
		switch resp.StatusCode {
		case 200:
			return resp.Body
		case 404:
			panic(rio.DataDNE.New("Fetch %q: not found", u.String()))
		default:
			panic(rio.WarehouseIOError.New("Unable to fetch %q: http status %s", u.String(), resp.Status))
		}
	case "https+ca":
		u.Path = path.Join(u.Path, string(dataHash))
		u.Scheme = "https"
		fallthrough
	case "https":
		resp, err := http.Get(u.String())
		if err != nil {
			panic(rio.WarehouseUnavailableError.New("Unable to fetch %q: %s", u.String(), err))
		}
		switch resp.StatusCode {
		case 200:
			return resp.Body
		case 404:
			panic(rio.DataDNE.New("Fetch %q: not found", u.String()))
		default:
			panic(rio.WarehouseIOError.New("Unable to fetch %q: http status %s", u.String(), resp.Status))
		}
	case "":
		panic(rio.ConfigError.New("missing scheme in warehouse URI; need a prefix, e.g. \"file://\" or \"http://\""))
	default:
		panic(rio.ConfigError.New("unsupported scheme in warehouse URI: %q", u.Scheme))
	}
}
