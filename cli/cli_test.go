package cli

import (
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
		App.Run(baseArgs)
	})

	testutil.Convey_IfCanNS("Within an environment that can run namespaces", t, func() {
		testutil.Convey_IfHaveRoot("It should run a basic example", func() {
			App.Run(append(baseArgs, "run", "-i", "lib/integration/basic.json"))
		})
	})

}
