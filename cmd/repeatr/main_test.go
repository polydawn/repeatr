package main

import (
	"bytes"
	"context"
	"testing"
)

// Returns the behavior from an invocation of Main.
func determineBehavior(args ...string) behavior {
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return Main(context.Background(), args, stdin, stdout, stderr)
}

func TestCLIParse(t *testing.T) {
	bhv := determineBehavior("repeatr", "wow")
	t.Logf("%#v\n", bhv.parsedArgs)

	bhv = determineBehavior("repeatr", "run")
	t.Logf("%#v\n", bhv.parsedArgs)

	bhv = determineBehavior("repeatr", "run", "file.frm")
	t.Logf("%#v\n", bhv.parsedArgs)
}
