package cli

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/testutil"
)

var (
	// os flag parsing mandates the executable name
	baseArgs = []string{"repeatr"}
)

// These can be swapped wholesale; demonstration only.

func Test(t *testing.T) {

	// Run from the top-level directory to avoid "../" irritants.
	// Optional TODO: upgrade repeatr to understand how to be relative to a directory.
	err := os.Chdir("..")
	if err != nil {
		panic(err)
	}

	Convey("It should not crash without args", t, func() {
		Main(baseArgs, ioutil.Discard, ioutil.Discard)
	})

	Convey("It should run a basic example", t,
		testutil.Requires(
			testutil.RequiresRoot,
			testutil.RequiresNamespaces,
			func(c C) {
				w := testutil.Writer{c}
				Main(append(baseArgs, "run", "-i", "lib/integration/basic.json"), w, w)
			},
		),
	)
}
