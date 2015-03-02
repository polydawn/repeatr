package main

import (
	"os"

	"polydawn.net/repeatr/cli"
)

func main() {
	cli.GetApp().Run(os.Args)
}
