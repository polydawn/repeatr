package cli

import (
	. "fmt"
	"os"

	"github.com/codegangsta/cli"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/null"
)

func GetApp() *cli.App {
	app := cli.NewApp()

	app.Name = "repeatr"
	app.Usage = "Run it. Run it again."
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "executor, e",
			Value: "null",
			Usage: "Which executor to use",
		},
	}

	app.Action = func(c *cli.Context) {

		desiredExecutor := c.String("executor")

		var executor executor.Executor

		switch desiredExecutor {
		case "null":
			executor = &null.Executor{}
		default:
			Println("No such executor", desiredExecutor)
			os.Exit(1)
		}

		executor.Run(def.Formula{})
	}

	return app
}
