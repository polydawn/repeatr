package streamer

import (
	"io"
	"math"
	"os"
	"sync"

	"github.com/spacemonkeygo/errors"
	"github.com/ugorji/go/codec"
)

var _ Mux = &CborMux{}

type CborMux struct {
	// the fd tip handles append; reads all use ReadAt and track their offsets individually
	file  *os.File
	codec *codec.Encoder
	wmu   sync.Mutex
}

type cborMuxRow struct {
	Label int    `json:"l"`
	Msg   []byte `json:"m,omitempty"`
	Sig   int    `json:"x,omitempty"` // 1->closed
}

func CborFileMux(filePath string) Mux {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0755)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}
	file.Write([]byte{codec.CborStreamArray})
	return &CborMux{
		file:  file,
		codec: codec.NewEncoder(file, new(codec.CborHandle)),
	}
	// consider using runtime.`SetFinalizer to close?  currently unhandled.
}

func (m *CborMux) write(label int, msg []byte) {
	m.wmu.Lock()
	defer m.wmu.Unlock()
	const magic_RAW = 0
	const magic_UTF8 = 1
	//	m.codec.MustEncode(cborMuxRow{
	//		Label: label,
	//		Msg:   msg,
	//	})
	_, enc := codec.GenHelperEncoder(m.codec)
	enc.EncodeMapStart(2)
	enc.EncodeString(magic_UTF8, "l")
	enc.EncodeInt(int64(label))
	enc.EncodeString(magic_UTF8, "m")
	enc.EncodeStringBytes(magic_RAW, msg)
}

func (m *CborMux) Close() {
	m.wmu.Lock()
	defer m.wmu.Unlock()
	m.file.Write([]byte{0xff}) // should be `codec.CborStreamBreak`, plz update upstream for vis
	// don't *actually* close, because readers can still be active on the same fd.
	// TODO further writes should be forced into an error state
}

func (m *CborMux) Appender(label int) io.WriteCloser {
	return &cborMuxAppender{m, label}
}

type cborMuxAppender struct {
	m     *CborMux
	label int
}

func (a *cborMuxAppender) Write(msg []byte) (int, error) {
	a.m.write(a.label, msg)
	return len(msg), nil
}

func (a *cborMuxAppender) Close() error {
	a.m.wmu.Lock()
	defer a.m.wmu.Unlock()
	const magic_UTF8 = 1
	_, enc := codec.GenHelperEncoder(a.m.codec)
	enc.EncodeMapStart(2)
	enc.EncodeString(magic_UTF8, "l")
	enc.EncodeInt(int64(a.label))
	enc.EncodeString(magic_UTF8, "x")
	enc.EncodeInt(int64(1))
	return nil
}

func (m *CborMux) Reader(labels ...int) io.Reader {
	// asking for a reader for a label that was never used will never
	// hit a close flag, so... don't do that?
	r := io.NewSectionReader(m.file, 1, math.MaxInt64/2)
	// TODO offset of one because that's the array open!
	// do something much more sane than skip it, please
	return &cborMuxReader{
		labels: &intset{labels},
		codec:  codec.NewDecoder(r, new(codec.CborHandle)),
	}
}

type cborMuxReader struct {
	labels *intset // remove them as we hit their close
	codec  *codec.Decoder
}

func (r *cborMuxReader) Read(msg []byte) (int, error) {
	var row cborMuxRow
	err := r.codec.Decode(&row)
	if err == io.EOF {
		// we don't pass EOF up unless our cbor says we're closed.
		// this could be a "temporary" EOF and appends will still be incoming.
		// TODO: we're also effectively required to block here, because otherwise the reader may spin.
		return 0, nil
	} else if err != nil {
		panic(err)
	}
	switch row.Sig {
	case 1:
		r.labels.Remove(row.Label)
		if r.labels.Empty() {
			return 0, io.EOF
		}
	default:
		if r.labels.Contains(row.Label) {
			// FIXME: handle overflow...
			return copy(msg, row.Msg), nil

		}
	}
	return 0, nil
}

type intset struct {
	s []int
}

func (s *intset) Contains(i int) bool {
	for _, v := range s.s {
		if i == v {
			return true
		}
	}
	return false
}

func (s *intset) Remove(i int) {
	old := s.s
	s.s = make([]int, 0, len(old))
	for _, v := range old {
		if v != i {
			s.s = append(s.s, v)
		}
	}
}

func (s *intset) Empty() bool {
	return len(s.s) == 0
}
