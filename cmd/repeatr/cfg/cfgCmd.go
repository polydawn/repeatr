package cfgCmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/codegangsta/cli"
	"github.com/go-yaml/yaml"
	"github.com/ugorji/go/codec"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/cmd/repeatr/bhv"
	"go.polydawn.net/repeatr/lib/cereal"
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
					Message: "`repeatr cfg parse` requires one input file (or '-' to read stdin).",
				}))
			}
		}()
		// slurp input
		ser, err := ioutil.ReadAll(input)
		if err != nil && err != io.EOF {
			panic(meep.Meep(
				&meep.ErrUnderspecified{},
				meep.Cause(fmt.Errorf("error reading input: %s", err)),
			))
		}
		// bounce serialization
		ser = cereal.Tab2space(ser)
		var raw interface{}
		if err := yaml.Unmarshal(ser, &raw); err != nil {
			panic(meep.Meep(
				&meep.ErrUnderspecified{},
				meep.Cause(fmt.Errorf("Could not parse yaml: %s", err)),
			))
		}
		if err := codec.NewEncoder(stdout, &codec.JsonHandle{}).Encode(raw); err != nil {
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
