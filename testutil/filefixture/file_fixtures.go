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

// TODO -- we need to segment these fixtures even MORE by minimum features.
// - BaseDirNonEpochMtime -- optional to support (not all tar inputs datum will!)
// - filesystems that differ only by modtime -- use these for the hash uniqueness subtable

// slightly varied structure.  empty dirs; maxdepth=2.
var Alpha Fixture = Fixture{"Alpha",
	[]FixtureFile{
		{fs.Metadata{Name: ".", Mode: 0755, ModTime: time.Unix(1000, 2000)}, nil},
		{fs.Metadata{Name: "./a", Mode: 01777, ModTime: time.Unix(3000, 2000)}, nil},
		{fs.Metadata{Name: "./b", Mode: 0750, ModTime: time.Unix(5000, 2000)}, nil},
		{fs.Metadata{Name: "./b/c", Mode: 0664, ModTime: time.Unix(7000, 2000)}, []byte("zyx")},
	},
}.defaults()

// flat structure.  all files.  convenient for checking mounts work with plain 'ls' output.
var Beta Fixture = Fixture{"Beta",
	[]FixtureFile{
		{fs.Metadata{Name: "."}, nil},
		{fs.Metadata{Name: "./1"}, []byte{}},
		{fs.Metadata{Name: "./2"}, []byte{}},
		{fs.Metadata{Name: "./3"}, []byte{}},
	},
}.defaults()

var Beta_Hash string = "9GYDihlrhHQRNPV10lms35kogosBekjqJVYzTj0O5H-QJYTU7vf0YAgh3XBWKKBC"

// describes a file where part of the path to it contains a symlink.  should be rejected by sane systems.
var Breakout Fixture = Fixture{"Breakout",
	[]FixtureFile{
		{fs.Metadata{Name: "."}, nil},
		{fs.Metadata{Name: "./danger", Linkname: "/tmp"}, nil},
		{fs.Metadata{Name: "./danger/dangerzone"}, nil},
		{fs.Metadata{Name: "./danger/dangerzone/laaaaanaaa"}, []byte("WHAT")},
	},
}.defaults() // this is *not* included in `All` because it's actually a little scary.

// deep and varied structures.  files and dirs.
// subtle: a dir with a sibling that's a suffix of its name (can trip up dir/child adjacency sorting).
// subtle: a file with a sibling that's a suffix of its name (other half of the test, to make sure the prefix doesn't create an incorect tree node).
var Gamma Fixture = Fixture{"Gamma",
	[]FixtureFile{
		{fs.Metadata{Name: "."}, nil},
		{fs.Metadata{Name: "./etc"}, nil},
		{fs.Metadata{Name: "./etc/init.d/"}, nil},
		{fs.Metadata{Name: "./etc/init.d/service-p"}, []byte("p!")},
		{fs.Metadata{Name: "./etc/init.d/service-q"}, []byte("q!")},
		{fs.Metadata{Name: "./etc/init/"}, nil},
		{fs.Metadata{Name: "./etc/init/zed"}, []byte("grue")},
		{fs.Metadata{Name: "./etc/trick"}, []byte("sib")},
		{fs.Metadata{Name: "./etc/tricky"}, []byte("sob")},
		{fs.Metadata{Name: "./var"}, nil},
		{fs.Metadata{Name: "./var/fun"}, []byte("zyx")},
	},
}.defaults()

var All []Fixture = []Fixture{
	Alpha,
	Beta,
	Gamma,
}
