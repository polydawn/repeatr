package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
	"go.polydawn.net/repeatr/cmd/repeatr/examine"
	"go.polydawn.net/repeatr/cmd/repeatr/formula"
	"go.polydawn.net/repeatr/cmd/repeatr/pack"
	"go.polydawn.net/repeatr/cmd/repeatr/run"
	"go.polydawn.net/repeatr/cmd/repeatr/twerk"
	"go.polydawn.net/repeatr/cmd/repeatr/unpack"
)

const appVersion = "v0.15+dev"

func main() {
	os.Exit(Main(os.Args, os.Stdin, os.Stdout, os.Stderr))
}
func Main(
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) (exitcode int) {
	subcommandHelpThunk := func(ctx *cli.Context) error {
		exitcode = cmdbhv.EXIT_BADARGS
		if ctx.Args().Present() {
			cli.ShowCommandHelp(ctx, ctx.Args().First())
		} else {
			cli.ShowCommandHelp(ctx, "")
		}
		return nil
	}
	app := &cli.App{
		Name:      "repeatr",
		Usage:     "Run it. Run it again.",
		UsageText: "Repeatr runs processes in containers, provisioning their inputs and saving their outputs using reliable, immutable, content-addressable goodness.",
		Version:   appVersion,
		Writer:    stderr,
		Commands: []cli.Command{
			{
				Name:  "run",
				Usage: "Run a formula",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "executor",
						Value: "runc",
						Usage: "Which executor to use",
					},
					cli.BoolFlag{
						Name:  "ignore-job-exit",
						Usage: "If true, repeatr will exit with 0/success even if the job exited nonzero.",
					},
					cli.StringSliceFlag{
						Name:  "patch, p",
						Usage: "Files with additional pieces of formula to apply before launch",
					},
					cli.StringSliceFlag{
						Name:  "env, e",
						Usage: "Apply additional environment vars to formula before launch (overrides 'patch').  Format like '-e KEY=val'",
					},
					cli.BoolFlag{
						Name:  "serialize, s",
						Usage: "Serialize output onto stdout",
					},
				},
				Action: runCmd.Run(stdout, stderr),
			},
			{
				Name:  "twerk",
				Usage: "Run one-time-use interactive (thus nonrepeatable!) command.  All the defaults are filled in for you.  Great for experimentation.",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "executor",
						Value: "runc",
						Usage: "Which executor to use. Using anything other than runc at this time is not particularly useful or functional.",
					},
					cli.StringSliceFlag{
						Name:  "patch, p",
						Usage: "Files with additional pieces of formula to apply before launch",
					},
					cli.StringSliceFlag{
						Name:  "env, e",
						Usage: "Apply additional environment vars to formula before launch (overrides 'patch').  Format like '-e KEY=val'",
					},
					cli.StringFlag{
						Name:  "policy, P",
						Value: string(def.PolicyRoutine),
						Usage: "Which capabilities policy to use",
					},
				},
				Action: twerkCmd.Twerk(stdin, stdout, stderr),
			},
			{
				Name:  "unpack",
				Usage: "fetch a ware and unpack it to a local filesystem",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "place",
						Usage: "Optional.  Where to place the filesystem.  Defaults to \"./unpack-[hash]\"",
					},
					cli.StringFlag{
						Name:  "kind",
						Usage: "What kind of data storage format to work with.",
					},
					cli.StringFlag{
						Name:  "hash",
						Usage: "The ID of the object to explore.",
					},
					cli.StringFlag{
						Name:  "where",
						Usage: "A URL giving coordinates to a warehouse where repeatr should find the object to explore.",
					},
					cli.BoolFlag{
						Name: "skip-exists",
						Usage: "If a file already exists at at '--place=%s', assume it's correct and exit immediately.  If this flag is not provided, the default behavior is to do the whole unpack, rolling over the existing files." +
							"  BE WARY of using this: it's effectively caching with no cachebusting rule.  Caveat emptor.",
					},
				},
				Action: unpackCmd.Unpack(stderr),
			},
			{
				Name:  "pack",
				Usage: "Scan a local filesystem reporting the hash, and (optionally) packing the data into snapshot form to save in a warehouse",
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
				Action: packCmd.Pack(stdout, stderr),
			},
			{
				Name:  "examine",
				Usage: "Examine a ware and the metadata of its contents, or a filesystem",
				Description: strings.Join([]string{
					"`repeatr examine` produces a human-readable manifest of every file in the named item",
					"(either wares or local filesystems may be examined), their properties, and their hashes.",
					"\n\n  ",
					"Output is structed as tab-delimited values -- you may feed it to an external `diff` program",
					"to compare one item with another; or, for easier reading, try piping it to `column -t`",
				}, " "),
				Subcommands: []cli.Command{
					{
						Name:  "ware",
						Usage: "Examine a ware from a warehouse",
						Flags: []cli.Flag{
							cli.StringFlag{
								Name:  "kind",
								Usage: "What kind of data storage format to work with.",
							},
							cli.StringFlag{
								Name:  "hash",
								Usage: "The ID of the object to examine.",
							},
							cli.StringFlag{
								Name:  "where",
								Usage: "A URL giving coordinates to a warehouse where repeatr should find the object to examine.",
							},
						},
						Action: examineCmd.ExamineWare(stdout, stderr),
					},
					{
						Name:   "file",
						Usage:  "Examine a local filesystem",
						Action: examineCmd.ExamineFile(stdout, stderr),
					},
				},
			},
			{
				Name:   "formula",
				Usage:  "Manipulate formulas programmatically (parse, validate, etc).",
				Action: subcommandHelpThunk,
				Subcommands: []cli.Command{
					{
						Name:   "parse",
						Usage:  "Parse formula and re-emit as json; error if any gross syntatic failures.",
						Action: formulaCmd.Parse(stdin, stdout, stderr),
					},
					{
						Name:   "setuphash",
						Usage:  "Print the setup hash of a formula",
						Action: formulaCmd.SetupHash(stdin, stdout, stderr),
					},
				},
			},
			{
				Name:  "version",
				Usage: "Shows the version of repeatr",
				Action: func(ctx *cli.Context) {
					cli.ShowVersion(ctx)
				},
			},
		},
		CommandNotFound: func(ctx *cli.Context, command string) {
			exitcode = cmdbhv.EXIT_BADARGS
			fmt.Fprintf(stderr, "Incorrect usage: '%s' is not a %s subcommand\n", command, ctx.App.Name)
		},
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Present() {
				cli.ShowCommandHelp(ctx, ctx.Args().First())
			} else {
				cli.ShowAppHelp(ctx)
			}
			return nil
		},
	}
	cli.VersionPrinter = func(ctx *cli.Context) {
		// Put some more info in our version printer.
		// Also, version goes to stdout.
		fmt.Fprintf(os.Stdout, "%v %v\n", ctx.App.Name, ctx.App.Version)
	}
	meep.Try(func() {
		if err := app.Run(args); err != nil {
			exitcode = cmdbhv.EXIT_BADARGS
			fmt.Fprintf(stderr, "Incorrect usage: %s", err)
		}
	}, meep.TryPlan{
		{ByType: &cmdbhv.ErrExit{},
			Handler: func(e error) {
				exitcode = e.(*cmdbhv.ErrExit).Code
				fmt.Fprintf(stderr, "%s\n", e)
			}},
		{ByType: &cmdbhv.ErrBadArgs{},
			Handler: func(e error) {
				exitcode = cmdbhv.EXIT_BADARGS
				fmt.Fprintf(stderr, "%s\n", e)
			}},
	})
	return
}
