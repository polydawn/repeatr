package testutil

import (
	"bytes"
	"fmt"
	"os"
	"strings"
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

/*
	Decorates a GoConvey test to check a set of `ConveyRequirement`s,
	returning a dummy test func that skips (with an explanation!) if any
	of the requirements are unsatisfied; if all is well, it yields
	the real test function unchanged.
*/
func Requires(action interface{}, requirements ...ConveyRequirement) func(c convey.C) {
	// examine requirements
	var widest int
	for _, req := range requirements {
		if len(req.Name) > widest {
			widest = len(req.Name)
		}
	}
	// check requirements
	var whynot bytes.Buffer
	var names []string
	allSat := true
	for _, req := range requirements {
		sat := req.Predicate()
		allSat = allSat && sat
		names = append(names, req.Name)
		fmt.Fprintf(&whynot, "requirement %*q: %v\n", widest+2, req.Name, sat)
	}
	// act
	if allSat {
		return func(c convey.C) {
			// attempted: inserting another convey that makes a single 'true=true' assertion so we see the prereqs and a green check mark.
			// doesn't work: doing so causes a leaf node, in which everything is run :/ even if skipped, the remaining `So` that aren't
			// in another block get attached to it, which makes verrry odd reading, and causes an extra repetition of anything
			// that isn't in another convey block.
			//	convey.SkipConvey(title, func() { convey.So(true, convey.ShouldBeTrue) })
			switch action := action.(type) {
			case func():
				action()
			case func(c convey.C):
				action(c)
			}
		}
	} else {
		title := "Prereqs: " + strings.Join(names, ", ")
		return func(c convey.C) {
			convey.Convey(title, nil)
			c.Println()
			c.Print(whynot.String())
		}
	}
}

type ConveyRequirement struct {
	Name      string
	Predicate func() bool
}

// Require that the tests are not running with the "short" flag enabled.
var RequiresLongRun = ConveyRequirement{"run long tests", func() bool { return !testing.Short() }}

// Require that the tests are running as uid 0 ('root').
var RequiresRoot = ConveyRequirement{"running as root", func() bool { return os.Getuid() == 0 }}

// Require the environment supports namespaces.  (Warning: rough, based on blacklisting and guesswork.)
var RequiresNamespaces = ConveyRequirement{"can namespace", func() bool {
	switch {
	case os.Getenv("TRAVIS") != "":
		// Travis's own virtualization appears to deny some of the magic bits we'd
		// like to set when exec'ing into a container.
		return false
	default:
		return true
	}
}}
