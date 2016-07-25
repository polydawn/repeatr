package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/codegangsta/cli"
	"github.com/go-yaml/yaml"
	"github.com/spacemonkeygo/errors"
	"github.com/ugorji/go/codec"
)

func CfgCommandPattern(stdin io.Reader, stdout, stderr io.Writer) cli.Command {
	return cli.Command{
		Name:  "cfg",
		Usage: "Manipulate config and formulas programmatically (parse, validate, etc).",
		Subcommands: []cli.Command{{
			Name:  "parse",
			Usage: "Parse config and re-emit as json; error if any gross syntatic failures.",
			Action: func(ctx *cli.Context) {
				// select input args
				input := func() io.Reader {
					switch l := len(ctx.Args()); {
					case l == 1:
						inputName := ctx.Args()[0]
						if inputName == "-" {
							return stdin
						} else {
							input, err := os.OpenFile(inputName, os.O_RDONLY, 0)
							if err != nil {
								panic(Exit.NewWith(
									fmt.Sprintf("error reading input: %s\n", err),
									SetExitCode(EXIT_USER),
								))
							}
							return input
						}
					default:
						panic(Error.NewWith(
							"`repeatr cfg parse` requires one input file (or '-' to read stdin).",
							SetExitCode(EXIT_BADARGS),
						))
					}
				}()
				// slurp input
				ser, err := ioutil.ReadAll(input)
				if err != nil && err != io.EOF {
					panic(Exit.NewWith(
						fmt.Sprintf("error reading input: %s\n", err),
						SetExitCode(EXIT_USER),
					))
				}
				// bounce serialization
				var raw interface{}
				if err := yaml.Unmarshal(ser, &raw); err != nil {
					panic(Exit.NewWith(
						fmt.Sprintf("Could not parse yaml: %s", err),
						SetExitCode(EXIT_USER),
					))
				}
				if err := codec.NewEncoder(stdout, &codec.JsonHandle{}).Encode(raw); err != nil {
					panic(errors.ProgrammerError.New("Transcription error: %s", errors.GetMessage(err)))
				}
				// i tend to expect a trailing linebreak from shell tools
				stdout.Write([]byte{'\n'})
			},
		}},
	}
}
