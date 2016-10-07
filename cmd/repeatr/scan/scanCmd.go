package scanCmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
	"go.polydawn.net/repeatr/rio"
)

func Scan(output, stderr io.Writer) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// args parse
		var warehouses def.WarehouseCoords
		if ctx.IsSet("where") {
			warehouses = def.WarehouseCoords{
				def.WarehouseCoord(ctx.String("where")),
			}
		}
		filters := &def.Filters{}
		try.Do(func() {
			filters.FromStringSlice(ctx.StringSlice("filter"))
		}).Catch(rio.ConfigError, func(err *errors.Error) {
			panic(meep.Meep(&cmdbhv.ErrBadArgs{
				Message: "malformed filter argument: could not parse: " + err.Message()}))
		}).Done()
		filters.InitDefaultsOutput()
		outputSpec := def.Output{
			Type:       ctx.String("kind"),
			Warehouses: warehouses,
			Filters:    filters,
			MountPath:  ctx.String("place"),
		}
		if outputSpec.Type == "" {
			panic(cmdbhv.ErrMissingParameter("kind"))
		}
		if outputSpec.MountPath == "" {
			outputSpec.MountPath = "."
		}
		// invoke
		log := log15.New()
		log.SetHandler(log15.StreamHandler(stderr, log15.TerminalFormat()))
		outputResult := scan(outputSpec, log)
		// output
		// FIXME serialization format.
		//  should be especially pretty and human-friendly; deserves custom code.
		//    really, you want that anyway for things like hassle-free syntax in practice for single URIs without an array, etc.
		msg, err := json.Marshal(outputResult)
		if err != nil {
			panic(meep.Meep(
				&meep.ErrProgrammer{},
				meep.Cause(fmt.Errorf("Transcription error: %s", err)),
			))
		}
		fmt.Fprintf(output, "%s\n", string(msg))
		return nil
	}
}
