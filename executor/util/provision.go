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
	// start having all filesystems
	filesystems := make(map[def.Input]integrity.Arena, len(inputs))
	fsGather := make(chan map[def.Input]materializerReport)
	for _, in := range inputs {
		go func(in def.Input) {
			try.Do(func() {
				journal.Info(fmt.Sprintf("Starting materialize for %s hash=%s", in.Type, in.Hash))
				arena := transmat.Materialize(
					integrity.TransmatKind(in.Type),
					integrity.CommitID(in.Hash),
					[]integrity.SiloURI{integrity.SiloURI(in.URI)},
				)
				journal.Info(fmt.Sprintf("Finished materialize for %s hash=%s", in.Type, in.Hash))
				fsGather <- map[def.Input]materializerReport{
					in: {Arena: arena},
				}
			}).Catch(integrity.Error, func(err *errors.Error) {
				journal.Warn(fmt.Sprintf("Errored during materialize for %s hash=%s", in.Type, in.Hash), "error", err.Message())
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
	journal.Info("All inputs acquired... starting assembly")

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
			filters := filter.FilterSet{}
			for _, name := range out.Filters {
				cfg := strings.Fields(name)
				switch cfg[0] {
				case "uid":
					f := filter.UidFilter{}
					if len(cfg) > 1 {
						f.Value, _ = strconv.Atoi(cfg[1])
					}
					filters.Put(f)
				case "gid":
					f := filter.GidFilter{}
					if len(cfg) > 1 {
						f.Value, _ = strconv.Atoi(cfg[1])
					}
					filters.Put(f)
				case "mtime":
					f := filter.MtimeFilter{}
					if len(cfg) > 1 {
						f.Value, _ = time.Parse(time.RFC3339, cfg[1])
					}
					filters.Put(f)
				default:
					continue
				}
			}
			scanPath := filepath.Join(rootfs, out.Location)
			journal.Info(fmt.Sprintf("Starting scan on %q", scanPath))
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
