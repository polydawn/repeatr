package main

import (
	"context"
	"io"
	"os"
)

func main() {
	ctx := context.Background()
	exitCode := Main(ctx, os.Args, os.Stdin, os.Stdout, os.Stderr)
	os.Exit(int(exitCode))
}

func Main(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return 100
}
