package streamer

import (
	"io"
	"math"
	"os"
	"sync"
	"time"

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
	buf    []byte // any remaining bytes from the last incomplete read
}

func (r *cborMuxReader) Read(msg []byte) (n int, err error) {
	n, err = r.read(msg)
	for n == 0 && err == nil {
		// we're effectively required to block here, because otherwise the reader may spin;
		// this is not a clueful wait; but it does prevent pegging a core.
		// quite dumb in this case is also quite fool-proof.
		time.Sleep(1 * time.Millisecond)
		n, err = r.read(msg)
	}
	return
}

/*
	Internal read method; may return `(0,nil)` in a number of occations,
	all of which the public `Read` method will translate into a wait+retry
	so that higher level consumers of the Reader interface don't get stuck
	spin-looping.

	Specifically, these situations cause empty reads:
	  - hitting EOF on the backing file, but still having labelled streams
	    that haven't been closed (i.e. we expect the file to still be growing)
	  - absorbing a message that isn't selected by this reader's filters
	  - absorbing a message that's a signal and has no body
*/
func (r *cborMuxReader) read(msg []byte) (int, error) {
	// first, finish yielding any buffered bytes from prior incomplete reads.
	if len(r.buf) > 0 {
		n := copy(msg, r.buf)
		r.buf = r.buf[n:]
		return n, nil
	}
	// scan the file for more rows and work with any that match our labels.
	var row cborMuxRow
	err := r.codec.Decode(&row)
	if err == io.EOF {
		// we don't pass EOF up unless our cbor says we're closed.
		// this could be a "temporary" EOF and appends will still be incoming.
		return 0, nil
	} else if err != nil {
		panic(err)
	}
	switch row.Sig {
	case 1:
		r.labels.Remove(row.Label)
		if r.labels.Empty() {
			return 0, io.EOF
		} else {
			// still more labels must be closed before we're EOF
			return 0, nil
		}
	default:
		if r.labels.Contains(row.Label) {
			n := copy(msg, row.Msg)
			r.buf = row.Msg[n:]
			return n, nil
		} else {
			// consuming an uninteresting label
			return 0, nil
		}
	}
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
