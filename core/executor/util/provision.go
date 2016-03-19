package util

import (
	"fmt"
	"path/filepath"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/io"
)

// Run inputs
func ProvisionInputs(transmat integrity.Transmat, inputs def.InputGroup, journal log15.Logger) map[string]integrity.Arena {
	// start having all filesystems
	// input names are used as keys, so must be unique
	fsGather := make(chan map[string]materializerReport)
	for name, in := range inputs {
		go func(name string, in *def.Input) {
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
					journal,
				)
				// submit report
				journal.Info(fmt.Sprintf("Finished materialize for %s hash=%s", in.Type, in.Hash))
				fsGather <- map[string]materializerReport{
					name: {Arena: arena},
				}
			}).Catch(integrity.Error, func(err *errors.Error) {
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
	filesystems := make(map[string]integrity.Arena, len(inputs))
	for range inputs {
		for name, report := range <-fsGather {
			if report.Err != nil {
				panic(report.Err)
			}
			filesystems[name] = report.Arena
		}
	}

	return filesystems
}

func AssembleFilesystem(
	assemblerFn integrity.Assembler,
	rootPath string,
	inputs def.InputGroup,
	inputArenas map[string]integrity.Arena,
	hostMounts []def.Mount,
	journal log15.Logger,
) integrity.Assembly {
	journal.Info("All inputs acquired... starting assembly")
	// process inputs
	assemblyParts := make([]integrity.AssemblyPart, 0, len(inputArenas))
	for name, arena := range inputArenas {
		assemblyParts = append(assemblyParts, integrity.AssemblyPart{
			SourcePath: arena.Path(),
			TargetPath: inputs[name].MountPath,
			Writable:   true, // TODO input config should have a word about this
		})
	}
	// process mounts
	for _, mount := range hostMounts {
		assemblyParts = append(assemblyParts, integrity.AssemblyPart{
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
	Arena integrity.Arena // if success
	Err   *errors.Error   // subtype of input.Error.  (others are forbidden by contract and treated as fatal.)
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
func PreserveOutputs(transmat integrity.Transmat, outputs def.OutputGroup, rootfs string, journal log15.Logger) def.OutputGroup {
	// run commit on the outputs
	scanGather := make(chan map[string]scanReport)
	for name, out := range outputs {
		go func(name string, out *def.Output) {
			out.Filters = &def.Filters{}
			out.Filters.InitDefaultsOutput()
			filterOptions := integrity.ConvertFilterConfig(*out.Filters)
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
					journal,
					filterOptions...,
				)
				out.Hash = string(commitID)
				// submit report
				journal.Info(fmt.Sprintf("Finished scan on %q", scanPath))
				scanGather <- map[string]scanReport{
					name: {Output: out},
				}
			}).Catch(integrity.Error, func(err *errors.Error) {
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
