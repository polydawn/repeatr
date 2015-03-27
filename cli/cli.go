package cli

import (
	. "fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"github.com/ugorji/go/codec"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor/dispatch"
)


var App *cli.App

func init() {
	App = cli.NewApp()

	App.Name = "repeatr"
	App.Usage = "Run it. Run it again."
	App.Version = "0.0.1"

	App.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "Run a formula",
			Action: Run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "executor, e",
					Value: "nsinit",
					Usage: "Which executor to use",
				},
				cli.StringFlag{
					Name:  "input, i",
					Value: "formula.json",
					Usage: "Location of input formula (json format)",
				},
			},
		},
	}
}

func Run(c *cli.Context) {

	try.Do(func() {
		executor := *executordispatch.Get(c.String("executor"))
		filename, _ := filepath.Abs(c.String("input"))

		content, err := ioutil.ReadFile(filename)
		if err != nil {
			Println(err)
			Println("Could not read file", filename)
			os.Exit(1)
		}

		dec := codec.NewDecoderBytes(content, &codec.JsonHandle{})

		formula := def.Formula{}
		dec.MustDecode(&formula)

		job := executor.Start(formula)
		Println("Job starting...")

		result := job.Wait()
		Println("Job finished with code", result.ExitCode)
		Println("Outputs:", result.Outputs)

		if result.Error != nil {
			Println("Problem executing job:", result.Error)
			os.Exit(3)
		}

		// DISCUSS: we could consider any non-zero exit a Error, but having that distinct from execution problems makes sense.
		// This is clearly silly and placeholder.
		os.Exit(result.ExitCode)

	}).Catch(def.ValidationError, func(e *errors.Error) {
		Println(e.Message())
		os.Exit(2)
	}).Done()
}
