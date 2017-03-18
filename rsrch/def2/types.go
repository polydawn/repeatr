package def

import (
	"fmt"
	"strings"
)

type Ware struct {
	Kind string
	Hash string
}

func ParseWare(s string) Ware {
	split := strings.Split(s, ":")
	if len(split) != 2 {
		return Ware{"invalid", fmt.Sprintf("%q", s)}
	}
	return Ware{split[0], split[1]}
}
func (w Ware) String() string {
	return w.Kind + ":" + w.Hash
}

type Formula struct {
	Inputs    Inputs    // Part of setup defn.
	Action    Action    // Part of setup defn.
	SaveSlots SaveSlots // Part of setup defn.

	Results *Results // Review: possible to write this as part of same document, but usually makes sense to keep them separate.

	Warehousing *Warehousing // Review: possible to write this as part of same document, but usually makes sense to keep them separate.
}

type Inputs map[string]Input

// union-style
type Input struct {
	Name string

	EnvValue   EnvValue
	Hostname   string
	Filesystem Filesystem
}

type EnvValue struct{ Name, Value string }

type Filesystem struct {
	Path string
	Ware Ware
}

type SaveSlots map[string]SaveSlot

// unison-style
type SaveSlot struct {
	Name string

	EnvValue   string // why not
	Filesystem FilesystemSlot
}

type FilesystemSlot struct {
	Path string
	Kind string
}

// union-style
type Action struct {
	// An array of strings to hand as args to exec -- creates a single process.
	Exec []string

	// An array of strings to feed as a script to a bash shell.
	// This means each string is subject to bash's env var rules, substitutions, etc.
	// The shell will be initialized with `set -e` -- any commands that error
	// will terminate the script immediately.
	// (You can build the same behavior using an Exec action and
	// `[]string{'bash', '-c', '...'}`; this is purely for convenience.)
	Script []string

	// Specify some basic rearrangements to the filesystem.
	// Since these are not turing complete, and all behaviors are implemented
	// within Repeatr (e.g. no gnutils/busybox/etc version to specify),
	// a formula with a Reshuffle action can be evaluated anywhere, even
	// without needing containers.
	//
	// Not yet implemented!
	// Also, should probably include its own versioning, so we can grow the
	// utilities included on their own schedule without causing breakage.
	Reshuffle interface{}
}

type Results struct {
	RunErr   error                  // Any error in running.  Means containment failed, not the task itself.
	ExitCode int                    // Exit code of the action.  Zero means success.  -1 in case of RunErr.
	Saved    map[string]interface{} // ? REVIEW: If SaveSlot only supported Filesystem, this would be map[string]Ware and be done with it.
}

type WarehouseCoord string

type Warehousing struct {
	Names    map[string]WarehouseCoord
	InputUse map[string][]string // map[inputName][]warehouseAlias  // DUBIOUS, see comment block in example
	SaveUse  map[string]string   // map[saveslotName]warehouseAlias // DUBIOUS, see comment block in example
}
