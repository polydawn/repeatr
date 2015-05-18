package util

import (
	"io"
	"os"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/output"
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
func PreserveOutputs(transmat integrity.Transmat, outputs []def.Output, rootfs string, journal io.Writer) []def.Output {
	// run commit on the outputs
	scanGather := make(chan scanReport)
	for _, out := range outputs {
		go func() {
			try.Do(func() {
				commitID := transmat.Scan(
					integrity.TransmatKind(out.Type),
					filepath.Join(rootfs, out.Location),
					[]integrity.SiloURI{integrity.SiloURI(out.URI)},
				)
				out.Hash = string(commitID)
				scanGather <- scanReport{Output: out}
			}).Catch(output.Error, func(err *errors.Error) {
				scanGather <- scanReport{Err: err}
			}).Done()
		}()
	}

	// gather reports
	var results []def.Output
	for report := range scanGather {
		if report.Err != nil {
			panic(report.Err)
		}
		results = append(results, report.Output)
	}

	return results
}

type scanReport struct {
	Output def.Output    // now including the hash
	Err    *errors.Error // subtype of output.Error.  (others are forbidden by contract and treated as fatal.)
}
