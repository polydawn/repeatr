package tar2

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
)

type Compression int

const (
	Uncompressed Compression = iota
	Bzip2
	Gzip
	Xz
)

func (compression *Compression) Extension() string {
	switch *compression {
	case Uncompressed:
		return "tar"
	case Bzip2:
		return "tar.bz2"
	case Gzip:
		return "tar.gz"
	case Xz:
		return "tar.xz"
	}
	return "[unknown]"
}

func DetectCompression(source []byte) Compression {
	// Compression detection patterns borrowed from docker/pkg/archive/archive.go, where they also reside under an Apache v2 license
	for compression, m := range map[Compression][]byte{
		Bzip2: {0x42, 0x5A, 0x68},
		Gzip:  {0x1F, 0x8B, 0x08},
		Xz:    {0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00},
	} {
		if bytes.Compare(m, source[:len(m)]) == 0 {
			return compression
		}
	}
	return Uncompressed
}

func Decompress(stream io.Reader) (io.Reader, error) {
	buf := bufio.NewReaderSize(stream, 8*1024)
	bs, err := buf.Peek(10)
	if err != nil {
		return nil, err
	}

	compression := DetectCompression(bs)
	switch compression {
	case Uncompressed:
		return buf, nil
	case Gzip:
		return gzip.NewReader(buf)
	case Bzip2:
		return bzip2.NewReader(buf), nil
	case Xz:
		return nil, fmt.Errorf("Unsupported compression format %s", (&compression).Extension())
	default:
		return nil, fmt.Errorf("Unsupported compression format %s", (&compression).Extension())
	}
}
