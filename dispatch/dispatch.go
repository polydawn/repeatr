package dispatch

import (
	. "fmt"
	"os"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/executor/null"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/input/dir"
	"polydawn.net/repeatr/input/tar"
	"polydawn.net/repeatr/output"
)

// TODO: This should not require a global string -> class map :|
// Should attempt to reflect-find, trying main package name first.
// Will make simpler to use extended transports, etc.

func GetExecutor(desire string) *executor.Executor {
	var executor executor.Executor

	switch desire {
	case "null":
		executor = &null.Executor{}
	default:
		Println("No such executor", desire)
		os.Exit(1)
	}

	return &executor
}

func GetInput(desire def.Input) *input.Input {
	var input input.Input

	switch desire.Type {
	case "dir":
		input = dir.New(desire)
	case "tar":
		input = tar.New(desire)
	default:
		Println("No such input", desire)
		os.Exit(1)
	}

	return &input
}

func GetOutput(desire def.Output) *output.Output {
	var output output.Output

	switch desire.Type {
	default:
		Println("No such output", desire)
		os.Exit(1)
	}

	return &output
}
