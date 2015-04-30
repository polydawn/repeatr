package testutil

import (
	"os"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func Convey_IfHaveRoot(items ...interface{}) {
	if os.Getuid() == 0 {
		convey.Convey(items...)
	} else {
		convey.SkipConvey(items...)
	}
}

/*
	Run tests if we think the environment supports namespaces; skip otherwise.

	(This is super rough; really it just expresses whether or not
	ns-init runs, based on trial and error.)
*/
func Convey_IfCanNS(items ...interface{}) {
	// Travis's own virtualization appears to deny some of the magic bits we'd
	// like to set when exec'ing into a container.
	switch {
	case os.Getenv("TRAVIS") != "":
		convey.SkipConvey(items...)
	default:
		convey.Convey(items...)
	}
}

func Convey_IfSlowTests(items ...interface{}) {
	if testing.Short() {
		convey.SkipConvey(items...)
	} else {
		convey.Convey(items...)
	}
}
