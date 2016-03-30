package s3

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	"github.com/rlmcpherson/s3gof3r"
	"polydawn.net/repeatr/rio"
)

var s3Conf = &s3gof3r.Config{
	Concurrency: 10,
	PartSize:    20 * 1024 * 1024,
	NTry:        10,
	Md5Check:    false,
	Scheme:      "https",
	Client:      s3gof3r.ClientWithTimeout(15 * time.Second),
}

func makeS3reader(bucketName string, path string, keys s3gof3r.Keys) io.ReadCloser {
	s3 := s3gof3r.New("s3.amazonaws.com", keys)
	w, _, err := s3.Bucket(bucketName).GetReader(path, s3Conf)
	if err != nil {
		if err2, ok := err.(*s3gof3r.RespError); ok && err2.Code == "NoSuchKey" {
			panic(rio.DataDNE.New("not stored here"))
		} else {
			panic(rio.WarehouseIOError.Wrap(err))
		}
	}
	return w
}

func makeS3writer(bucketName string, path string, keys s3gof3r.Keys) io.WriteCloser {
	s3 := s3gof3r.New("s3.amazonaws.com", keys)
	w, err := s3.Bucket(bucketName).PutWriter(path, nil, s3Conf)
	if err != nil {
		panic(rio.WarehouseIOError.Wrap(err))
	}
	return w
}

func reloc(bucketName, oldPath, newPath string, keys s3gof3r.Keys) {
	s3 := s3gof3r.New("s3.amazonaws.com", keys)
	bucket := s3.Bucket(bucketName)
	// this is a POST at the bottom, and copies are a PUT.  whee.
	//w, err := s3.Bucket(bucketName).PutWriter(newPath, copyInstruction, s3Conf)
	// So, implement our own aws copy API.
	req, err := http.NewRequest("PUT", "", &bytes.Buffer{})
	if err != nil {
		panic(rio.WarehouseIOError.Wrap(err))
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
		panic(rio.WarehouseIOError.Wrap(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		panic(rio.WarehouseIOError.Wrap(newRespError(resp)))
	}
	// delete previous location
	if err := bucket.Delete(oldPath); err != nil {
		panic(rio.WarehouseIOError.Wrap(err))
	}
}

func newRespError(r *http.Response) *s3gof3r.RespError {
	e := new(s3gof3r.RespError)
	e.StatusCode = r.StatusCode
	b, _ := ioutil.ReadAll(r.Body)
	xml.NewDecoder(bytes.NewReader(b)).Decode(e) // parse error from response
	r.Body.Close()
	return e
}
