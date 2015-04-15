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

// want: better explanation to appear around skipping
// fun facts
// - goconvey has a reporter interface, but you need to fork them to add one
// - you can do skips with parameters
// - you'd have to wrap names manually
// - there's no transport available for additional flags (i'd like to treat "skip:todo" differently than "skip:needroot")

// also, ideally, in this stuff below, since we can't convert the top node
// into a skip and still have any powers of speech, it'd be nice to
// report a skip for every child (depth 1 is fine)... but alas, no real way unless we
// proxy literally every `Convey` method.  which is sounding better and better...
// but then that would require us to repeat copious amounts of gls use too, which is just
// way, way WAY further off the deep end than is even remotely appropriate for this.

/*
	Like `Convey`, but accepts a set of `ConveyRequirement`s before `Convey`'s
	usual arguments, and runs tests only if all requirements are satisified.

	If the requirements are not satisfied, prints a single child `Convey` node
	describing the requirements and a description of what's missing.
*/
func ConveyRequires(items ...interface{}) {
	// discover requirements
	var nReqs int
	var reqs []ConveyRequirement
	for i, it := range items {
		if req, ok := it.(ConveyRequirement); ok {
			reqs = append(reqs, req)
		} else {
			nReqs = i
			break
		}
	}
	var widest int
	for _, req := range reqs {
		if len(req.name) > widest {
			widest = len(req.name)
		}
	}
	// extract passthrough work descriptions
	passItems := items[nReqs:]
	action := passItems[len(passItems)-1]
	// check requirements
	var whynot bytes.Buffer
	var names []string
	allSat := true
	for _, req := range reqs {
		sat := req.predicate()
		allSat = allSat && sat
		names = append(names, req.name)
		fmt.Fprintf(&whynot, "requirement %*q: %v\n", widest+2, req.name, sat)
	}
	// act
	if allSat {
		passItems[len(passItems)-1] = func(c convey.C) {
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
		passItems[len(passItems)-1] = func(c convey.C) {
			//convey.SkipConvey(title, func() {})
			convey.Convey(title, nil)
			c.Println()
			c.Print(whynot.String())
		}
	}
	convey.Convey(passItems...)
}

type ConveyRequirement struct {
	name      string
	predicate func() bool
}

// Require that the tests are not running with the "short" flag enabled.
var LongRunRequirement = ConveyRequirement{"run long tests", func() bool { return !testing.Short() }}

// Require that the tests are running as uid 0 ('root').
var RootRequirement = ConveyRequirement{"running as root", func() bool { return os.Getuid() == 0 }}

// Require the environment supports namespaces.  (Warning: rough, based on blacklisting and guesswork.)
var CanNSRequirement = ConveyRequirement{"can namespace", func() bool {
	switch {
	case os.Getenv("TRAVIS") != "":
		// Travis's own virtualization appears to deny some of the magic bits we'd
		// like to set when exec'ing into a container.
		return false
	default:
		return true
	}
}}

// another way through all this would just be to make a function that resolves a slice of ConveyRequirements
// into a skipConvey token, but that's yet again blockaded by an interface that's pretty abruptly limiting
// the instant you step outside of the DSL: nothing like the `noSpecifier` or `skipConvey` tokens are exposed.

// wait, `nil` action funcs also hit the skip path, right?
// yeah, that'd work.  it's not really much different than what we ended up doing here... except it's chainable, for some definition of the word.
