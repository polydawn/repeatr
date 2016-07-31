package executordispatch

import (
	"path/filepath"

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
	var execr executor.Executor

	switch desire {
	case "null":
		execr = &null.Executor{}
	case "chroot":
		execr = &chroot.Executor{}
	case "runc":
		execr = &runc.Executor{}
	default:
		panic(executor.ConfigError.New("No such executor %s", desire))
	}

	// Set the base path to operate from
	execr.Configure(filepath.Join(jank.Base(), "executor", desire))

	return execr
}
