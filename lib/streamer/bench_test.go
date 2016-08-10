package streamer

import (
	"io/ioutil"
	"os"
	"testing"
)

type benchFn func(b *testing.B, strm Mux)

func cbormuxLooper(b *testing.B, fn benchFn) {
	os.Remove("./logfile")
	for i := 0; i < b.N; i++ {
		strm := CborFileMux("./logfile")
		fn(b, strm)
		strm.(*CborMux).file.Close() // TODO inability to do this in api is mistake
		os.Remove("./logfile")
	}
}

// Writing : just bench the fsyncs of scribbling things down.

func BenchmarkCbormuxWriting(b *testing.B) {
	cbormuxLooper(b, _BenchmarkWriting)
}
func _BenchmarkWriting(b *testing.B, strm Mux) {
	a1 := strm.Appender(1)
	for i := 0; i < 16; i++ { // unsurprisingly, if you wiggle this const, it's linear
		a1.Write([]byte("asdf"))
	}
	a1.Close()
}

// WriteReadRead : all writes are done, then all reads are done (twice).

func BenchmarkCbormuxWriteReadRead(b *testing.B) {
	cbormuxLooper(b, _BenchmarkWriteReadRead)
}
func _BenchmarkWriteReadRead(b *testing.B, strm Mux) {
	a1 := strm.Appender(1)
	for i := 0; i < 16; i++ {
		a1.Write([]byte("asdf"))
	}
	a1.Close()
	ioutil.ReadAll(strm.Reader(1))
	ioutil.ReadAll(strm.Reader(1))
}

// PlayingTagInterleaved : writes (once) and reads (again twice) are interleaved -- but paired; reads never block.

func BenchmarkCbormuxPlayingTagInterleaved(b *testing.B) {
	cbormuxLooper(b, _BenchmarkPlayingTagInterleaved)
}
func _BenchmarkPlayingTagInterleaved(b *testing.B, strm Mux) {
	a1 := strm.Appender(1)
	r1 := strm.Reader(1)
	for i := 0; i < 16; i++ {
		a1.Write([]byte("asdf"))
		r1.Read(make([]byte, 4))
	}
	a1.Close()
}

// PlayingTagBlocking : what happens when writes come in from a slower goroutine, and we have reads in the bench routine?
// Reads are aligned, fwiw.

func BenchmarkCbormuxPlayingTagBlocking(b *testing.B) {
	cbormuxLooper(b, _BenchmarkPlayingTagBlocking)
}
func _BenchmarkPlayingTagBlocking(b *testing.B, strm Mux) {
	go func() {
		a1 := strm.Appender(1)
		for i := 0; i < 16; i++ {
			a1.Write([]byte("asdf"))
		}
		a1.Close()
	}()
	r1 := strm.Reader(1)
	for i := 0; i < 16; i++ {
		r1.Read(make([]byte, 4))
	}
	// block until eof (to avoid filedescriptor close race -- could also waitgroup the writer routine)
	r1.Read(make([]byte, 0))
}
