package packCmd

import (
	"fmt"
	"io"

	"github.com/inconshreveable/log15"
	"github.com/ugorji/go/codec"
	"github.com/urfave/cli"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
)

func Pack(stdout, stderr io.Writer) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// args parse
		var warehouses def.WarehouseCoords
		if ctx.IsSet("where") {
			warehouses = def.WarehouseCoords{
				def.WarehouseCoord(ctx.String("where")),
			}
		}
		filters := &def.Filters{}
		meep.Try(func() {
			filters.FromStringSlice(ctx.StringSlice("filter"))
		}, meep.TryPlan{
			{ByType: &def.ErrConfigValidation{}, Handler: func(e error) {
				panic(meep.Meep(&cmdbhv.ErrBadArgs{Message: "malformed filter argument: could not parse: " + e.Error()}))
			}},
		})
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
		// set up logging.
		log := log15.New()
		log.SetHandler(log15.StreamHandler(stderr, log15.TerminalFormat()))
		// invoke
		var output def.Output
		meep.Try(func() {
			output = pack(outputSpec, log)
		}, cmdbhv.TryPlanToExit)
		// output
		if err := codec.NewEncoder(stdout, &codec.JsonHandle{Indent: -1}).Encode(output); err != nil {
			panic(meep.Meep(
				&meep.ErrProgrammer{},
				meep.Cause(fmt.Errorf("Transcription error: %s", err)),
			))
		}
		stdout.Write([]byte{'\n'})
		return nil
	}
}
