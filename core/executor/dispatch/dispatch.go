package executordispatch

import (
	"path/filepath"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/executor"
	"go.polydawn.net/repeatr/core/executor/impl/chroot"
	"go.polydawn.net/repeatr/core/executor/impl/null"
	"go.polydawn.net/repeatr/core/executor/impl/runc"
	"go.polydawn.net/repeatr/core/jank"
)

// TODO: This should not require a global string -> class map :|
// Should attempt to reflect-find, trying main package name first.
// Will make simpler to use extended transports, etc.

func Get(desire string) executor.Executor {
	var executor executor.Executor

	switch desire {
	case "null":
		executor = &null.Executor{}
	case "chroot":
		executor = &chroot.Executor{}
	case "runc":
		executor = &runc.Executor{}
	default:
		panic(def.ValidationError.New("No such executor %s", desire))
	}

	// Set the base path to operate from
	executor.Configure(filepath.Join(jank.Base(), "executor", desire))

	return executor
}
