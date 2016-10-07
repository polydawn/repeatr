package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
	"go.polydawn.net/repeatr/cmd/repeatr/cfg"
	"go.polydawn.net/repeatr/cmd/repeatr/examine"
	"go.polydawn.net/repeatr/cmd/repeatr/scan"
	"go.polydawn.net/repeatr/cmd/repeatr/unpack"
	rcli "go.polydawn.net/repeatr/core/cli"
)

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
		Version:   "v0.13+dev",
		Writer:    stderr,
		Commands: []cli.Command{
			rcli.RunCommandPattern(stdout, stderr),
			rcli.TwerkCommandPattern(stdin, stdout, stderr),
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
				Action: scanCmd.Scan(stdout, stderr),
			},
			{
				Name:  "examine",
				Usage: "examine a ware and the metadata of its contents, or a filesystem",
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
						Usage: "examine a ware from a warehouse",
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
						Usage:  "examine a local filesystem",
						Action: examineCmd.ExamineFile(stdout, stderr),
					},
				},
			},
			{
				Name:   "cfg",
				Usage:  "Manipulate config and formulas programmatically (parse, validate, etc).",
				Action: subcommandHelpThunk,
				Subcommands: []cli.Command{{
					Name:   "parse",
					Usage:  "Parse config and re-emit as json; error if any gross syntatic failures.",
					Action: cfgCmd.Parse(stdin, stdout, stderr),
				}},
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
		fmt.Fprintf(os.Stdout, "git commit %v\n", rcli.GITCOMMIT)
		fmt.Fprintf(os.Stdout, "build date %v\n", rcli.BUILDDATE)
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
