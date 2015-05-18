package flak

import (
	"hash"
	"io"
)

/*
	Proxies a reader, hashing the stream as it's read.
	(This is useful if using `io.Copy` to move bytes from a reader to
	a writer, and you want to use that goroutine to power the hashing as
	well but replacing the writer with a multiwriter is out of bounds.)
*/
type HashingReader struct {
	R      io.Reader
	Hasher hash.Hash
}

func (r *HashingReader) Read(b []byte) (int, error) {
	n, err := r.R.Read(b)
	r.Hasher.Write(b[:n])
	return n, err
}
