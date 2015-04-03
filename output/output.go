package output

import (
	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
)

type Output interface {
	// See docs for input/input.go ; this is very very presumptory ATM and is liable to violent change.
	Apply(rootPath string) <-chan Report
}

type Report struct {
	Err    *errors.Error // error, or nil if success.  All errors will be under `output.Error`'s type.
	Output def.Output    // this comes back with the Hash field set
}
