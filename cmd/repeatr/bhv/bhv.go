package cmdbhv

import (
	"fmt"

	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
)

const (
	EXIT_SUCCESS      = 0
	EXIT_BADARGS      = 1
	EXIT_UNKNOWNPANIC = 2  // same code as golang uses when the process dies naturally on an unhandled panic.
	EXIT_JOB          = 10 // used to indicate a job reported a nonzero exit code (from cli commands that execute a single job).
	EXIT_USER         = 3  // grab bag for general user input errors (try to make a more specific code if possible/useful)
	EXIT_PERMS        = 4  // exit code when the repeatr process doesn't have sufficient permissions/capabilities to do the work.
)

type ErrExit struct {
	Message string
	Code    int
}

func (e ErrExit) Error() string {
	return e.Message
}

var TryPlanToExit = meep.TryPlan{
	{ByType: &def.ErrConfigParsing{}, Handler: func(e error) {
		panic(&ErrExit{e.Error(), EXIT_BADARGS})
	}},
	{ByType: &def.ErrConfigValidation{}, Handler: func(e error) {
		panic(&ErrExit{e.Error(), EXIT_BADARGS})
	}},
	{ByType: &def.ErrWarehouseUnavailable{}, Handler: func(e error) {
		panic(&ErrExit{e.Error(), EXIT_USER})
	}},
	{ByType: &def.ErrWarehouseProblem{}, Handler: func(e error) {
		panic(&ErrExit{e.Error(), EXIT_USER})
	}},
	{ByType: &def.ErrWareDNE{}, Handler: func(e error) {
		panic(&ErrExit{e.Error(), EXIT_USER})
	}},
	{ByType: &def.ErrWareCorrupt{}, Handler: func(e error) {
		panic(&ErrExit{e.Error(), EXIT_USER})
	}},
}

type ErrBadArgs struct {
	meep.TraitAutodescribing
	Message string
}

func ErrMissingParameter(paramName string) error {
	return meep.Meep(&ErrBadArgs{
		Message: fmt.Sprintf("%q is a required parameter", paramName),
	})
}

type ErrPermissions struct {
	meep.TraitAutodescribing
	Message string // TODO currently freetext, would be nice to refine but not yet clear what will be the most meaningful
}
