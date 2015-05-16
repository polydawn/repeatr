package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/output/dispatch"
)

// Run inputs
func ProvisionInputs(transmat integrity.Transmat, assemblerFn integrity.Assembler, inputs []def.Input, rootfs string, journal io.Writer) integrity.Assembly {
	// start having all filesystems
	filesystems := make(map[def.Input]integrity.Arena, len(inputs))
	fsGather := make(chan map[def.Input]materializerReport)
	for _, in := range inputs {
		go func(in def.Input) {
			try.Do(func() {
				fsGather <- map[def.Input]materializerReport{
					in: materializerReport{Arena: transmat.Materialize(
						integrity.TransmatKind(in.Type),
						integrity.CommitID(in.Hash),
						[]integrity.SiloURI{integrity.SiloURI(in.URI)},
					)},
				}
			}).Catch(input.Error, func(err *errors.Error) {
				fsGather <- map[def.Input]materializerReport{
					in: materializerReport{Err: err},
				}
			}).Done()
		}(in)
	}

	// (we don't have any output setup at this point, but if we do in the future, that'll be here.)

	// gather materialized inputs
	for range inputs {
		for in, report := range <-fsGather {
			if report.Err != nil {
				panic(report.Err)
			}
			filesystems[in] = report.Arena
		}
	}

	// assemble them into the final tree
	assemblyParts := make([]integrity.AssemblyPart, 0, len(filesystems))
	for input, arena := range filesystems {
		assemblyParts = append(assemblyParts, integrity.AssemblyPart{
			SourcePath: arena.Path(),
			TargetPath: input.Location,
			Writable:   true, // TODO input config should have a word about this
		})
	}
	assembly := assemblerFn(rootfs, assemblyParts)
	return assembly
}

type materializerReport struct {
	Arena integrity.Arena // if success
	Err   *errors.Error   // subtype of input.Error.  (others are forbidden by contract and treated as fatal.)
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
		path := filepath.Join(rootfs, output.Location)

		report := <-outputdispatch.Get(output).Apply(path)
		if report.Err != nil {
			panic(report.Err)
		}
		fmt.Fprintln(journal, "Output", x+1, "hash:", report.Output.Hash)
	}

	return outputs
}
