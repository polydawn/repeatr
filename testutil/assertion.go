package testutil

import (
	"testing"
)

// tl;dr:
//  - `Assert*` methods are Fatalf if failed;
//  - `Want*` methods are Errorf if failed.

type thunk func(string, ...interface{})

func AssertNoError(t *testing.T, err error) { t.Helper(); lambdaNoError(t.Fatalf, err) }
func WantNoError(t *testing.T, err error)   { t.Helper(); lambdaNoError(t.Errorf, err) }
func lambdaNoError(act thunk, err error) {
	if err != nil {
		act("unexpected error: %s", err)
	}
}
