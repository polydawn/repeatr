package cli

import (
	"io"

	"github.com/codegangsta/cli"
)

func Main(args []string, journal, output io.Writer) {
	App := cli.NewApp()

	App.Name = "repeatr"
	App.Usage = "Run it. Run it again."
	App.Version = "0.0.1"

	App.Writer = journal

	App.Commands = []cli.Command{
		RunCommandPattern(journal),
		ScanCommandPattern(journal, output),
	}

	App.Run(args)
}
