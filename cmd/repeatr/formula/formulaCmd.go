package formulaCmd

import (
	"fmt"
	"io"
	"os"

	"github.com/ugorji/go/codec"
	"github.com/urfave/cli"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/api/hitch"
	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
)

func Parse(stdin io.Reader, stdout, stderr io.Writer) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// select input args
		input := func() io.Reader {
			switch l := len(ctx.Args()); {
			case l == 1:
				inputName := ctx.Args()[0]
				if inputName == "-" {
					return stdin
				} else {
					// Check if it exists first for polite error message
					_, err := os.Lstat(inputName)
					if os.IsNotExist(err) {
						panic(meep.Meep(&cmdbhv.ErrBadArgs{
							Message: "that path does not exist"}))
					}

					input, err := os.OpenFile(inputName, os.O_RDONLY, 0)
					if err != nil {
						panic(meep.Meep(
							&meep.ErrUnderspecified{},
							meep.Cause(fmt.Errorf("error reading input: %s", err)),
						))
					}
					return input
				}
			default:
				panic(meep.Meep(&cmdbhv.ErrBadArgs{
					Message: "`repeatr formula parse` requires one input file (or '-' to read stdin).",
				}))
			}
		}()
		var frm def.Formula
		hitch.DecodeYaml(input, &frm)
		if err := codec.NewEncoder(stdout, &codec.JsonHandle{}).Encode(&frm); err != nil {
			panic(meep.Meep(
				&meep.ErrProgrammer{},
				meep.Cause(fmt.Errorf("Transcription error: %s", err)),
			))
		}
		// i tend to expect a trailing linebreak from shell tools
		stdout.Write([]byte{'\n'})
		return nil
	}
}

func SetupHash(stdin io.Reader, stdout, stderr io.Writer) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// select input args
		input := func() io.Reader {
			switch l := len(ctx.Args()); {
			case l == 1:
				inputName := ctx.Args()[0]
				if inputName == "-" {
					return stdin
				} else {
					// Check if it exists first for polite error message
					_, err := os.Lstat(inputName)
					if os.IsNotExist(err) {
						panic(meep.Meep(&cmdbhv.ErrBadArgs{
							Message: "that path does not exist"}))
					}
					// read file
					input, err := os.OpenFile(inputName, os.O_RDONLY, 0)
					if err != nil {
						panic(meep.Meep(
							&meep.ErrUnderspecified{},
							meep.Cause(fmt.Errorf("error reading input: %s", err)),
						))
					}
					return input
				}
			default:
				panic(meep.Meep(&cmdbhv.ErrBadArgs{
					Message: "`repeatr formula parse` requires one input file (or '-' to read stdin).",
				}))
			}
		}()

		var frm def.Formula
		hitch.DecodeYaml(input, &frm)
		hash := frm.SetupHash()
		fmt.Fprintf(stdout, "%s\n", hash)
		return nil
	}
}
