package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input/dispatch"
	"polydawn.net/repeatr/output/dispatch"
)

// Run inputs
// TODO: run all simultaneously, waitgroup out the errors
func ProvisionInputs(inputs []def.Input, rootfs string, journal io.Writer) {
	for x, input := range inputs {
		fmt.Fprintln(journal, "Provisioning input", x+1, input.Type, "to", input.Location)
		path := filepath.Join(rootfs, input.Location)

		// Ensure that the parent folder of this input exists
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			panic(errors.IOError.Wrap(err))
		}

		// Run input
		err = <-inputdispatch.Get(input).Apply(path)
		if err != nil {
			panic(err)
		}
	}
}

// Output folders should exist
// TODO: discussion
func ProvisionOutputs(outputs []def.Output, rootfs string, journal io.Writer) {
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
func PreserveOutputs(outputs []def.Output, rootfs string, journal io.Writer) []def.Output {
	for x, output := range outputs {
		fmt.Fprintln(journal, "Persisting output", x+1, output.Type, "from", output.Location)
		// path := filepath.Join(rootfs, output.Location)

		report := <-outputdispatch.Get(output).Apply(rootfs)
		if report.Err != nil {
			panic(report.Err)
		}
		fmt.Fprintln(journal, "Output", x+1, "hash:", report.Output.Hash)
	}

	return outputs
}
