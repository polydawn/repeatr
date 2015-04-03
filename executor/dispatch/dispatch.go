package executordispatch

import (
	"os"
	"path/filepath"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/chroot"
	"polydawn.net/repeatr/executor/nsinit"
	"polydawn.net/repeatr/executor/null"
)

// TODO: This should not require a global string -> class map :|
// Should attempt to reflect-find, trying main package name first.
// Will make simpler to use extended transports, etc.

func Get(desire string) executor.Executor {
	var executor executor.Executor

	switch desire {
	case "null":
		executor = &null.Executor{}
	case "nsinit":
		executor = &nsinit.Executor{}
	case "chroot":
		executor = &chroot.Executor{}
	default:
		panic(def.ValidationError.New("No such executor %s", desire))
	}

	// Set the base path to operate from
	executor.Configure(filepath.Join(os.TempDir(), "repeatr", "executor", desire))

	return executor
}
