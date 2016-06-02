package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/core/executor/util"
	"polydawn.net/repeatr/rio"
)

func ScanCommandPattern(output, stderr io.Writer) cli.Command {
	return cli.Command{
		Name:  "scan",
		Usage: "Scan a local filesystem, optionally packing the data into a warehouse",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "place",
				Value: ".",
				Usage: "Optional.  The local filesystem path to scan.  Defaults to your current directory.",
			},
			cli.StringFlag{
				Name:  "kind",
				Usage: "What kind of data storage format to work with.",
			},
			cli.StringFlag{
				Name:  "where",
				Usage: "Optional.  A URL giving coordinates to a warehouse where repeatr should store the scanned data.",
			},
			cli.StringSliceFlag{
				Name:  "filter",
				Usage: "Optional.  Filters to apply when scanning.  If not provided, reasonable defaults (flattening uid, gid, and mtime) will be used.",
			},
		},
		Action: func(ctx *cli.Context) {
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
				panic(Error.NewWith("Malformed argument: filters could not parse: "+err.Message(), SetExitCode(EXIT_BADARGS)))
			}).Done()
			filters.InitDefaultsOutput()
			outputSpec := def.Output{
				Type:       ctx.String("kind"),
				Warehouses: warehouses,
				Filters:    filters,
				MountPath:  ctx.String("place"),
			}
			if outputSpec.Type == "" {
				panic(Error.NewWith("Missing argument: \"kind\" is a required parameter for scan", SetExitCode(EXIT_BADARGS)))
			}
			if outputSpec.MountPath == "" {
				outputSpec.MountPath = "."
			}
			// invoke
			log := log15.New()
			log.SetHandler(log15.StreamHandler(stderr, log15.TerminalFormat()))
			outputResult := Scan(outputSpec, log)
			// output
			// FIXME serialization format.
			//  should be especially pretty and human-friendly; deserves custom code.
			//    really, you want that anyway for things like hassle-free syntax in practice for single URIs without an array, etc.
			msg, err := json.Marshal(outputResult)
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(output, "%s\n", string(msg))
		},
	}
}

/*
	Spits out a chunk of json on stdout that can be used as
	a `Input` specification in a `Formula`.
*/
func Scan(outputSpec def.Output, log log15.Logger) def.Output {
	// TODO validate MountPath exists, give nice errors

	// todo: create validity checking api for URIs, check them all before launching anything
	warehouses := make([]rio.SiloURI, len(outputSpec.Warehouses))
	for i, wh := range outputSpec.Warehouses {
		warehouses[i] = rio.SiloURI(wh)
	}

	// So, this CLI command is *not* in its rights to change the subject area,
	//  so take that as a pretty strong hint that filters are going to have to pass down *into* transmats.
	commitID := util.DefaultTransmat().Scan(
		// All of this stuff that's type-coercing?
		//  Yeah these are hints that this stuff should be facing data validation.
		rio.TransmatKind(outputSpec.Type),
		outputSpec.MountPath,
		warehouses,
		log,
		rio.ConvertFilterConfig(*outputSpec.Filters)...,
	)

	outputSpec.Hash = string(commitID)
	return outputSpec
}
