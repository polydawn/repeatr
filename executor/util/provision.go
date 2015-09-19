package util

import (
	"fmt"
	"path/filepath"

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
	// input names are used as keys, so must be unique
	inputsByName := make(map[string]def.Input, len(inputs))
	for _, in := range inputs {
		// TODO checks should also be sooner, up in cfg parse
		// but this check is for programmatic access as well (errors down the line can get nonobvious if you skip this).
		if _, ok := inputsByName[in.Name]; ok {
			panic(errors.ProgrammerError.New("duplicate name in input config"))
		}
		inputsByName[in.Name] = in
	}
	filesystems := make(map[string]integrity.Arena, len(inputs))
	fsGather := make(chan map[string]materializerReport)
	for _, in := range inputs {
		go func(in def.Input) {
			try.Do(func() {
				journal.Info(fmt.Sprintf("Starting materialize for %s hash=%s", in.Type, in.Hash))
				// todo: create validity checking api for URIs, check them all before launching anything
				warehouses := make([]integrity.SiloURI, len(in.Warehouses))
				for i, wh := range in.Warehouses {
					warehouses[i] = integrity.SiloURI(wh)
				}
				// invoke transmat (blocking, potentially long time)
				arena := transmat.Materialize(
					integrity.TransmatKind(in.Type),
					integrity.CommitID(in.Hash),
					warehouses,
				)
				// submit report
				journal.Info(fmt.Sprintf("Finished materialize for %s hash=%s", in.Type, in.Hash))
				fsGather <- map[string]materializerReport{
					in.Name: {Arena: arena},
				}
			}).Catch(integrity.Error, func(err *errors.Error) {
				journal.Warn(fmt.Sprintf("Errored during materialize for %s hash=%s", in.Type, in.Hash), "error", err.Message())
				fsGather <- map[string]materializerReport{
					in.Name: {Err: err},
				}
			}).Done()
		}(in)
	}

	// (we don't have any output setup at this point, but if we do in the future, that'll be here.)

	// gather materialized inputs
	for range inputs {
		for name, report := range <-fsGather {
			if report.Err != nil {
				panic(report.Err)
			}
			filesystems[name] = report.Arena
		}
	}
	journal.Info("All inputs acquired... starting assembly")

	// assemble them into the final tree
	assemblyParts := make([]integrity.AssemblyPart, 0, len(filesystems))
	for name, arena := range filesystems {
		assemblyParts = append(assemblyParts, integrity.AssemblyPart{
			SourcePath: arena.Path(),
			TargetPath: inputsByName[name].MountPath,
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
	// We no longer make output MountPaths by default.
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
			filterOptions := make([]integrity.MaterializerConfigurer, 0, 3)
			out.Filters.InitDefaultsOutput()
			switch out.Filters.UidMode {
			case def.FilterKeep: // easy, just no filter.
			case def.FilterUse:
				f := filter.UidFilter{out.Filters.Uid}
				filterOptions = append(filterOptions, integrity.UseFilter(f))
			default:
				panic(errors.ProgrammerError.New("unhandled filter mode %v", out.Filters.UidMode))
			}
			switch out.Filters.GidMode {
			case def.FilterKeep: // easy, just no filter.
			case def.FilterUse:
				f := filter.GidFilter{out.Filters.Gid}
				filterOptions = append(filterOptions, integrity.UseFilter(f))
			default:
				panic(errors.ProgrammerError.New("unhandled filter mode %v", out.Filters.GidMode))
			}
			switch out.Filters.MtimeMode {
			case def.FilterKeep: // easy, just no filter.
			case def.FilterUse:
				f := filter.MtimeFilter{out.Filters.Mtime}
				filterOptions = append(filterOptions, integrity.UseFilter(f))
			default:
				panic(errors.ProgrammerError.New("unhandled filter mode %v", out.Filters.MtimeMode))
			}

			scanPath := filepath.Join(rootfs, out.MountPath)
			journal.Info(fmt.Sprintf("Starting scan on %q", scanPath))
			try.Do(func() {
				// todo: create validity checking api for URIs, check them all before launching anything
				warehouses := make([]integrity.SiloURI, len(out.Warehouses))
				for i, wh := range out.Warehouses {
					warehouses[i] = integrity.SiloURI(wh)
				}
				// invoke transmat (blocking, potentially long time)
				commitID := transmat.Scan(
					integrity.TransmatKind(out.Type),
					scanPath,
					warehouses,
					filterOptions...,
				)
				out.Hash = string(commitID)
				// submit report
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
