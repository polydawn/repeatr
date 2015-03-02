package cli

import (
	. "fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/ugorji/go/codec"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/null"
)

func GetApp() *cli.App {
	app := cli.NewApp()

	app.Name = "repeatr"
	app.Usage = "Run it. Run it again."
	app.Version = "0.0.1"

	app.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "Run a formula",
			Action: Run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "executor, e",
					Value: "null",
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

	return app
}

func Run(c *cli.Context) {

	desiredExecutor := c.String("executor")
	filename, _ := filepath.Abs(c.String("input"))

	var executor executor.Executor

	switch desiredExecutor {
	case "null":
		executor = &null.Executor{}
	default:
		Println("No such executor", desiredExecutor)
		os.Exit(1)
	}

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		Println(err)
		Println("Could not read file", filename)
		os.Exit(1)
	}

	dec := codec.NewDecoderBytes(content, &codec.JsonHandle{})

	formula := def.Formula{}
	dec.MustDecode(&formula)

	executor.Run(formula)
}
