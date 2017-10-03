package testutil

import (
	"reflect"
	"testing"
)

// tl;dr:
//  - `Assert*` methods are Fatalf if failed;
//  - `Want*` methods are Errorf if failed.

type thunk func(string, ...interface{})

func AssertNoError(t *testing.T, err error) { t.Helper(); lambdaNoError(t, t.Fatalf, err) }
func WantNoError(t *testing.T, err error)   { t.Helper(); lambdaNoError(t, t.Errorf, err) }
func lambdaNoError(t *testing.T, act thunk, err error) {
	t.Helper()
	if err != nil {
		act("unexpected error: %s", err)
	}
}

func AssertEqual(t *testing.T, got, want interface{}) { t.Helper(); lambdaEqual(t, t.Fatalf, got, want) }
func WantEqual(t *testing.T, got, want interface{})   { t.Helper(); lambdaEqual(t, t.Errorf, got, want) }
func lambdaEqual(t *testing.T, act thunk, got, want interface{}) {
	t.Helper()
	if reflect.DeepEqual(got, want) == false {
		act("expected equality: want %v, got %v", want, got)
	}
}
