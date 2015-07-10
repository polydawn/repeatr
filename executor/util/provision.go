package util

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/io"
)

// Run inputs
func ProvisionInputs(transmat integrity.Transmat, assemblerFn integrity.Assembler, inputs []def.Input, rootfs string, journal io.Writer) integrity.Assembly {
	// start having all filesystems
	filesystems := make(map[def.Input]integrity.Arena, len(inputs))
	fsGather := make(chan map[def.Input]materializerReport)
	for _, in := range inputs {
		go func(in def.Input) {
			try.Do(func() {
				fmt.Fprintf(journal, "Starting materialize for %s hash=%s\n", in.Type, in.Hash)
				arena := transmat.Materialize(
					integrity.TransmatKind(in.Type),
					integrity.CommitID(in.Hash),
					[]integrity.SiloURI{integrity.SiloURI(in.URI)},
				)
				fmt.Fprintf(journal, "Finished materialize for %s hash=%s\n", in.Type, in.Hash)
				fsGather <- map[def.Input]materializerReport{
					in: {Arena: arena},
				}
			}).Catch(integrity.Error, func(err *errors.Error) {
				fmt.Fprintf(journal, "Errored during materialize for %s hash=%s -- %s\n", in.Type, in.Hash, err)
				fsGather <- map[def.Input]materializerReport{
					in: {Err: err},
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
	fmt.Fprintf(journal, "All inputs acquired... starting assembly\n")

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
	fmt.Fprintf(journal, "Assembly complete!\n")
	return assembly
}

type materializerReport struct {
	Arena integrity.Arena // if success
	Err   *errors.Error   // subtype of input.Error.  (others are forbidden by contract and treated as fatal.)
}

func ProvisionOutputs(outputs []def.Output, rootfs string, journal io.Writer) {
	// We no longer make output locations by default.
	// Originally, this seemed like a good idea, because it would be a
	//  consistent stance and allow us to use more complex (e.g. mount-powered)
	//   output shuttling concepts later without any fuss.
	// However, practically speaking, as a default, this has turned out not to work well.
	// Other tools in the world expect clear directories.
	// One example that makes this instantaneously game-over in the real
	//  world is git: `git clone [url] .` *won't work* if there are any other
	//   dirs already existing under `.`; and given how frequently projects
	//    put their build and test outputs in gitignored dirs under their
	//     project versioning root, we very frequently see outputs pointed there.
	//      That creates a lot of noise... so we're dropping the behavior.
}

// Run outputs
// TODO: run all simultaneously, waitgroup out the errors
func PreserveOutputs(transmat integrity.Transmat, outputs []def.Output, rootfs string, journal io.Writer) []def.Output {
	// run commit on the outputs
	scanGather := make(chan scanReport)
	for _, out := range outputs {
		go func(out def.Output) {
			scanPath := filepath.Join(rootfs, out.Location)
			fmt.Fprintf(journal, "Starting scan on %q\n", scanPath)
			try.Do(func() {
				// TODO: following is hack; badly need to update config parsing to understand this first-class
				warehouseCoordsList := make([]integrity.SiloURI, 0)
				if out.URI != "" {
					warehouseCoordsList = append(warehouseCoordsList, integrity.SiloURI(out.URI))
				}
				// invoke transmat
				commitID := transmat.Scan(
					integrity.TransmatKind(out.Type),
					scanPath,
					warehouseCoordsList,
				)
				out.Hash = string(commitID)
				fmt.Fprintf(journal, "Finished scan on %q\n", scanPath)
				scanGather <- scanReport{Output: out}
			}).Catch(integrity.Error, func(err *errors.Error) {
				fmt.Fprintf(journal, "Errored scan on %q -- %s\n", scanPath, err)
				scanGather <- scanReport{Err: err}
			}).Done()
		}(out)
	}

	// gather reports
	var results []def.Output
	for range outputs {
		report := <-scanGather
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
