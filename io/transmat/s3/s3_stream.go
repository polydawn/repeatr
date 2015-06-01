package s3

import (
	"io"
	"time"

	"github.com/rlmcpherson/s3gof3r"
	"polydawn.net/repeatr/io"
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
		panic(integrity.WarehouseConnectionError.Wrap(err))
	}
	return w
}
