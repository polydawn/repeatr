/*
	Utilities for describing test filesystems, setting them up, and checking
	that an extant filesystem matches a description.  Only intended to be
	imported by test code.
*/
package filefixture

import (
	"time"

	"polydawn.net/repeatr/lib/fs"
)

var Alpha Fixture = Fixture{"Alpha",
	[]FixtureFile{
		{fs.Metadata{Name: ".", Mode: 0755, ModTime: time.Unix(1000, 2000)}, nil},
		{fs.Metadata{Name: "./a", Mode: 01777, ModTime: time.Unix(3000, 2000)}, nil},
		{fs.Metadata{Name: "./b", Mode: 0750, ModTime: time.Unix(5000, 2000)}, nil},
		{fs.Metadata{Name: "./b/c", Mode: 0664, ModTime: time.Unix(7000, 2000)}, []byte("zyx")},
	},
}.defaults()

var Beta Fixture = Fixture{"Beta",
	[]FixtureFile{
		{fs.Metadata{Name: "."}, nil},
		{fs.Metadata{Name: "./1"}, []byte{}},
		{fs.Metadata{Name: "./2"}, []byte{}},
		{fs.Metadata{Name: "./3"}, []byte{}},
	},
}.defaults()

var All []Fixture = []Fixture{
	Alpha,
	Beta,
}
