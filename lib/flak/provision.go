package flak

import (
	. "fmt"
	"os"
	"path/filepath"

	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input/dispatch"
	"polydawn.net/repeatr/output/dispatch"
)

// Run inputs
// TODO: run all simultaneously, waitgroup out the errors
func ProvisionInputs(inputs []def.Input, rootfs string) {
	for x, input := range inputs {
		Println("Provisioning input", x+1, input.Type, "to", input.Location)
		path := filepath.Join(rootfs, input.Location)

		// Ensure that the parent folder of this input exists
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			panic(errors.IOError.Wrap(err))
		}

		// Run input
		err = <-inputdispatch.Get(input).Apply(path)
		if err != nil {
			Println("Input", x+1, "failed:", err)
			panic(err)
		}
	}
}

// Output folders should exist
// TODO: discussion
func ProvisionOutputs(outputs []def.Output, rootfs string) {
	for _, output := range outputs {
		path := filepath.Join(rootfs, output.Location)
		err := os.MkdirAll(path, 0755)
		if err != nil {
			panic(errors.IOError.Wrap(err))
		}
	}
}

// Run outputs
// TODO: run all simultaneously, waitgroup out the errors
func PreserveOutputs(outputs []def.Output, rootfs string) []def.Output {
	for x, output := range outputs {
		Println("Persisting output", x+1, output.Type, "from", output.Location)
		// path := filepath.Join(rootfs, output.Location)

		err := <-outputdispatch.Get(output).Apply(rootfs)
		if err != nil {
			Println("Output", x+1, "failed:", err)
			panic(err)
		}

		// TODO: doesn't get hash info, etc
	}

	return outputs
}
