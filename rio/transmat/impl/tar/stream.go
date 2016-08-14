package tar

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"go.polydawn.net/repeatr/lib/guid"
	"go.polydawn.net/repeatr/rio"
)

func makeReader(dataHash rio.CommitID, warehouseCoords rio.SiloURI) io.ReadCloser {
	u, err := url.Parse(string(warehouseCoords))
	if err != nil {
		panic(rio.ConfigError.New("failed to parse URI: %s", err))
	}
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

// summarizes behavior of basically all transports where tar is used as the fs metaphor... they're just one blob
// ... nvm, haven't actually thought of anything that needs more than io.ReadCloser yet
//type soloStreamReader struct {
//	io.Reader
//	io.Closer
//}

func makeWriteController(warehouseCoords rio.SiloURI) StreamingWarehouseWriteController {
	u, err := url.Parse(string(warehouseCoords))
	if err != nil {
		panic(rio.ConfigError.New("failed to parse URI: %s", err))
	}
	controller := &fileWarehouseWriteController{
		pathPrefix: u.Path,
	}
	switch u.Scheme {
	case "file+ca":
		controller.ctntAddr = true
		fallthrough
	case "file":
		// Pick a random upload path
		controller.pathPrefix = filepath.Join(u.Host, controller.pathPrefix) // file uris don't have hosts
		if controller.ctntAddr {
			controller.tmpPath = filepath.Join(controller.pathPrefix, ".tmp.upload."+guid.New())
		} else {
			controller.tmpPath = filepath.Join(path.Dir(controller.pathPrefix), ".tmp.upload."+path.Base(controller.pathPrefix)+"."+guid.New())
		}
		// Check if warehouse path exists.
		// Warehouse is expected to exist already; transmats
		//  should *not* create one whimsically, that's someone else's responsibility.
		warehouseBasePath := filepath.Dir(controller.tmpPath)
		if _, err := os.Stat(warehouseBasePath); err != nil {
			panic(rio.WarehouseUnavailableError.New("Warehouse unavailable: %q %s", warehouseBasePath, err))
		}
		// Open file to shovel data into
		file, err := os.OpenFile(controller.tmpPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(rio.WarehouseIOError.New("Unable to write %q: %s", controller.tmpPath, err))
		}
		controller.stream = file
		return controller
	case "http+ca":
		fallthrough
	case "http":
		fallthrough
	case "https+ca":
		fallthrough
	case "https":
		panic(rio.ConfigError.New("http transports are only supported for read-only use"))
	case "":
		panic(rio.ConfigError.New("missing scheme in warehouse URI; need a prefix, e.g. \"file://\" or \"http://\""))
	default:
		panic(rio.ConfigError.New("unsupported scheme in warehouse URI: %q", u.Scheme))
	}
}

type StreamingWarehouseWriteController interface {
	Writer() io.Writer
	Commit(dataHash rio.CommitID)
}

type fileWarehouseWriteController struct {
	stream     io.WriteCloser
	tmpPath    string
	pathPrefix string
	ctntAddr   bool
}

func (wc *fileWarehouseWriteController) Writer() io.Writer {
	return wc.stream
}
func (wc *fileWarehouseWriteController) Commit(dataHash rio.CommitID) {
	wc.stream.Close()
	var finalPath string
	if wc.ctntAddr {
		finalPath = path.Join(wc.pathPrefix, string(dataHash))
	} else {
		finalPath = wc.pathPrefix
	}
	os.Rename(wc.tmpPath, finalPath)
}
