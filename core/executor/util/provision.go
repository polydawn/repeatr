package util

import (
	"fmt"
	"path/filepath"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/rio"
)

// Run inputs
func ProvisionInputs(transmat rio.Transmat, inputs def.InputGroup, journal log15.Logger) map[string]rio.Arena {
	// start having all filesystems
	// input names are used as keys, so must be unique
	fsGather := make(chan map[string]materializerReport)
	for name, in := range inputs {
		go func(name string, in *def.Input) {
			try.Do(func() {
				journal.Info(fmt.Sprintf("Starting materialize for %s hash=%s", in.Type, in.Hash))
				// todo: create validity checking api for URIs, check them all before launching anything
				warehouses := make([]rio.SiloURI, len(in.Warehouses))
				for i, wh := range in.Warehouses {
					warehouses[i] = rio.SiloURI(wh)
				}
				// invoke transmat (blocking, potentially long time)
				arena := transmat.Materialize(
					rio.TransmatKind(in.Type),
					rio.CommitID(in.Hash),
					warehouses,
					journal,
				)
				// submit report
				journal.Info(fmt.Sprintf("Finished materialize for %s hash=%s", in.Type, in.Hash))
				fsGather <- map[string]materializerReport{
					name: {Arena: arena},
				}
			}).Catch(rio.Error, func(err *errors.Error) {
				journal.Warn(fmt.Sprintf("Errored during materialize for %s hash=%s", in.Type, in.Hash), "error", err.Message())
				fsGather <- map[string]materializerReport{
					name: {Err: err},
				}
			}).Done()
		}(name, in)
	}

	// (we don't have any output setup at this point, but if we do in the future, that'll be here.)

	// gather materialized inputs
	// any errors are re-raised immediately (TODO: this currently doesn't fan out smooth cancellations)
	nInputs := len(inputs)
	filesystems := make(map[string]rio.Arena, nInputs)
	for range inputs {
		for name, report := range <-fsGather {
			if report.Err != nil {
				panic(report.Err)
			}
			journal.Info(fmt.Sprintf("Input %d/%d ready", len(filesystems)+1, nInputs))
			filesystems[name] = report.Arena
		}
	}

	return filesystems
}

func AssembleFilesystem(
	assemblerFn rio.Assembler,
	rootPath string,
	inputs def.InputGroup,
	inputArenas map[string]rio.Arena,
	hostMounts []def.Mount,
	journal log15.Logger,
) rio.Assembly {
	journal.Info("All inputs acquired... starting assembly")
	// process inputs
	assemblyParts := make([]rio.AssemblyPart, 0, len(inputArenas))
	for name, arena := range inputArenas {
		assemblyParts = append(assemblyParts, rio.AssemblyPart{
			SourcePath: arena.Path(),
			TargetPath: inputs[name].MountPath,
			Writable:   true, // TODO input config should have a word about this
		})
	}
	// process mounts
	for _, mount := range hostMounts {
		assemblyParts = append(assemblyParts, rio.AssemblyPart{
			SourcePath: mount.SourcePath,
			TargetPath: mount.TargetPath,
			Writable:   mount.Writable,
			BareMount:  true,
		})
	}
	// assemmmmmmmmblllle
	assembly := assemblerFn(rootPath, assemblyParts)
	journal.Info("Assembly complete!")
	return assembly
}

type materializerReport struct {
	Arena rio.Arena     // if success
	Err   *errors.Error // subtype of input.Error.  (others are forbidden by contract and treated as fatal.)
}

func ProvisionOutputs(outputs def.OutputGroup, rootfs string, journal log15.Logger) {
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
func PreserveOutputs(transmat rio.Transmat, outputs def.OutputGroup, rootfs string, journal log15.Logger) def.OutputGroup {
	// run commit on the outputs
	scanGather := make(chan map[string]scanReport)
	for name, out := range outputs {
		go func(name string, out *def.Output) {
			out.Filters = &def.Filters{}
			out.Filters.InitDefaultsOutput()
			filterOptions := rio.ConvertFilterConfig(*out.Filters)
			scanPath := filepath.Join(rootfs, out.MountPath)
			journal.Info(fmt.Sprintf("Starting scan on %q", scanPath))
			try.Do(func() {
				// todo: create validity checking api for URIs, check them all before launching anything
				warehouses := make([]rio.SiloURI, len(out.Warehouses))
				for i, wh := range out.Warehouses {
					warehouses[i] = rio.SiloURI(wh)
				}
				// invoke transmat (blocking, potentially long time)
				commitID := transmat.Scan(
					rio.TransmatKind(out.Type),
					scanPath,
					warehouses,
					journal,
					filterOptions...,
				)
				out.Hash = string(commitID)
				// submit report
				journal.Info(fmt.Sprintf("Finished scan on %q", scanPath))
				scanGather <- map[string]scanReport{
					name: {Output: out},
				}
			}).Catch(rio.Error, func(err *errors.Error) {
				journal.Warn(fmt.Sprintf("Errored scan on %q", scanPath), "error", err.Message())
				scanGather <- map[string]scanReport{
					name: {Err: err},
				}
			}).Done()
		}(name, out)
	}

	// gather reports
	results := def.OutputGroup{}
	for range outputs {
		for name, report := range <-scanGather {
			if report.Err != nil {
				panic(report.Err)
			}
			results[name] = report.Output
		}
	}

	return results
}

type scanReport struct {
	Output *def.Output   // now including the hash
	Err    *errors.Error // subtype of output.Error.  (others are forbidden by contract and treated as fatal.)
}
