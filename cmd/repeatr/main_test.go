package main

import (
	"bytes"
	"io"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func callMain(args []string, stdin io.Reader) (string, string, int) {
	if stdin == nil {
		stdin = &bytes.Buffer{}
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := Main(args, stdin, stdout, stderr)
	return stdout.String(), stderr.String(), code
}

func TestCLI(t *testing.T) {
	Convey("Test CLI", t, func() {
		Convey("Given no args", func() {
			stdout, stderr, code := callMain(
				[]string{"repeatr"}, nil,
			)
			Convey("We expect help text", func() {
				So(stderr, ShouldContainSubstring, "USAGE")
				So(stdout, ShouldEqual, "")
			})
			Convey("We expect an exit code = 0", func() {
				So(code, ShouldEqual, EXIT_SUCCESS)
			})
		})
		Convey("Given an invalid subcommand", func() {
			stdout, stderr, code := callMain(
				[]string{"repeatr", "notacommand"}, nil,
			)
			Convey("We expect an error message", func() {
				So(stderr, ShouldContainSubstring, "notacommand")
				So(stdout, ShouldEqual, "")
			})
			Convey("We expect an exit code = 1", func() {
				So(code, ShouldEqual, EXIT_BADARGS)
			})
		})
	})
}
