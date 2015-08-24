package util

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/io/filter"
)

// Run inputs
func ProvisionInputs(transmat integrity.Transmat, assemblerFn integrity.Assembler, inputs []def.Input, rootfs string, journal log15.Logger) integrity.Assembly {
	// massage a bunch of maps.  `def.Input` isn't a valid key, sadly, because of the URI slice.
	// so we're keying everything by Location instead, since there's no reason for that to not be unique.
	inputsMap := make(map[string]def.Input, len(inputs))
	filesystems := make(map[string]integrity.Arena, len(inputs))
	fsGather := make(chan map[string]materializerReport)
	for _, in := range inputs {
		if _, ok := inputsMap[in.Location]; ok {
			panic("duplicate Location in input config") // TODO this should be validated much earlier.
		}
		inputsMap[in.Location] = in
	}

	// start having all filesystems
	for _, in := range inputs {
		go func(in def.Input) {
			try.Do(func() {
				warehouseCoordsList := make([]integrity.SiloURI, len(in.URI))
				for i, s := range in.URI {
					warehouseCoordsList[i] = integrity.SiloURI(s)
				}
				journal.Info(fmt.Sprintf("Starting materialize for %s hash=%s", in.Type, in.Hash))
				arena := transmat.Materialize(
					integrity.TransmatKind(in.Type),
					integrity.CommitID(in.Hash),
					warehouseCoordsList,
				)
				journal.Info(fmt.Sprintf("Finished materialize for %s hash=%s", in.Type, in.Hash))
				fsGather <- map[string]materializerReport{
					in.Location: {Arena: arena},
				}
			}).Catch(integrity.Error, func(err *errors.Error) {
				journal.Warn(fmt.Sprintf("Errored during materialize for %s hash=%s", in.Type, in.Hash), "error", err.Message())
				fsGather <- map[string]materializerReport{
					in.Location: {Err: err},
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
	journal.Info("All inputs acquired... starting assembly")

	// assemble them into the final tree
	assemblyParts := make([]integrity.AssemblyPart, 0, len(filesystems))
	for location, arena := range filesystems {
		assemblyParts = append(assemblyParts, integrity.AssemblyPart{
			SourcePath: arena.Path(),
			TargetPath: location,
			Writable:   true, // TODO input config should have a word about this
		})
	}
	assembly := assemblerFn(rootfs, assemblyParts)
	journal.Info("Assembly complete!")
	return assembly
}

type materializerReport struct {
	Arena integrity.Arena // if success
	Err   *errors.Error   // subtype of input.Error.  (others are forbidden by contract and treated as fatal.)
}

func ProvisionOutputs(outputs []def.Output, rootfs string, journal log15.Logger) {
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
func PreserveOutputs(transmat integrity.Transmat, outputs []def.Output, rootfs string, journal log15.Logger) []def.Output {
	// run commit on the outputs
	scanGather := make(chan scanReport)
	for _, out := range outputs {
		go func(out def.Output) {
			filterOptions := make([]integrity.MaterializerConfigurer, 0, 4)
			for _, name := range out.Filters {
				cfg := strings.Fields(name)
				switch cfg[0] {
				case "uid":
					f := filter.UidFilter{}
					if len(cfg) > 1 {
						f.Value, _ = strconv.Atoi(cfg[1])
					}
					filterOptions = append(filterOptions, integrity.UseFilter(f))
				case "gid":
					f := filter.GidFilter{}
					if len(cfg) > 1 {
						f.Value, _ = strconv.Atoi(cfg[1])
					}
					filterOptions = append(filterOptions, integrity.UseFilter(f))
				case "mtime":
					f := filter.MtimeFilter{}
					if len(cfg) > 1 {
						f.Value, _ = time.Parse(time.RFC3339, cfg[1])
					}
					filterOptions = append(filterOptions, integrity.UseFilter(f))
				default:
					continue
				}
			}
			scanPath := filepath.Join(rootfs, out.Location)
			journal.Info(fmt.Sprintf("Starting scan on %q", scanPath))
			try.Do(func() {
				warehouseCoordsList := make([]integrity.SiloURI, len(out.URI))
				for i, s := range out.URI {
					warehouseCoordsList[i] = integrity.SiloURI(s)
				}
				// invoke transmat
				commitID := transmat.Scan(
					integrity.TransmatKind(out.Type),
					scanPath,
					warehouseCoordsList,
					filterOptions...,
				)
				out.Hash = string(commitID)
				journal.Info(fmt.Sprintf("Finished scan on %q", scanPath))
				scanGather <- scanReport{Output: out}
			}).Catch(integrity.Error, func(err *errors.Error) {
				journal.Warn(fmt.Sprintf("Errored scan on %q", scanPath), "error", err.Message())
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
