package iofilter

import (
	"bufio"
	"bytes"
	"io"
)

var (
	non = []byte{}
	tab = []byte{'\t'}
	br  = []byte{'\n'}
)

// Proxies content by line, calling the proxied writer exactly once per line.
func LineBufferingWriter(w io.Writer) io.Writer {
	return LinePrefixingWriter(w, non)
}

func LineIndentingWriter(w io.Writer) io.Writer {
	return LinePrefixingWriter(w, tab)
}

func LinePrefixingWriter(w io.Writer, prefix []byte) io.Writer {
	return &Reframer{
		Delegate:  w,
		SplitFunc: bufio.ScanLines,
		Prefix:    prefix,
		Suffix:    br,
	}
}

func LineFlankingWriter(w io.Writer, prefix, suffix []byte) io.Writer {
	return &Reframer{
		Delegate:  w,
		SplitFunc: bufio.ScanLines,
		Prefix:    prefix,
		Suffix:    append(suffix[:], '\n'),
	}
}

var _ io.Writer = &Reframer{}

type Reframer struct {
	Delegate  io.Writer
	SplitFunc bufio.SplitFunc
	Prefix    []byte
	Suffix    []byte

	rem []byte
}

func (rfrm *Reframer) Write(b []byte) (int, error) {
	rfrm.rem = append(rfrm.rem, b...)
	for len(rfrm.rem) > 0 { // if loop until the buffer is exhausted, or another cond breaks out
		adv, tok, err := rfrm.SplitFunc(rfrm.rem, false)
		if err != nil {
			return len(b), err
		}
		if adv == 0 { // when we no longer have a full chunk, return
			return len(b), nil
		}
		// join all the things we're about to write, because we want them emitted as an atom.
		// (this may matter if the writer we're pushing into internally mutexes for sharing, for example.)
		rfrm.Delegate.Write(bytes.Join([][]byte{
			rfrm.Prefix,
			tok,
			rfrm.Suffix,
		}, []byte{}))
		rfrm.rem = rfrm.rem[adv:]
	}
	// we always state the entire range of bytes provided was written,
	//  because it is... if in buffer.  but we defintely don't need it re-sent.
	return len(b), nil
}
