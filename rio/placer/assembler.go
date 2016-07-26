package placer

import (
	"path/filepath"
	"sort"

	"go.polydawn.net/repeatr/lib/fs"
	"go.polydawn.net/repeatr/rio"
)

func NewAssembler(p rio.Placer) rio.Assembler {
	return defaultAssembler{p}.Assemble
}

type defaultAssembler struct {
	Placer rio.Placer
}

var _ rio.Assembler = defaultAssembler{}.Assemble

func (a defaultAssembler) Assemble(basePath string, parts []rio.AssemblyPart) rio.Assembly {
	sort.Sort(rio.AssemblyPartsByPath(parts))
	housekeeping := &defaultAssembly{}
	for _, part := range parts {
		destBasePath := filepath.Join(basePath, part.TargetPath)
		if err := fs.MkdirAll(destBasePath); err != nil {
			panic(Error.Wrap(err)) // REVIEW: not clear if placers and assemblers should get separate error hierarchies.  have yet to think of a useful scenario for it.
		}
		housekeeping.record(a.Placer(part.SourcePath, destBasePath, part.Writable, part.BareMount))
	}
	return housekeeping
}

/*
	Gathers the teardown instructions from all the Placers used to assemble
	the filesystem.  Dispatches teardown to each of them in reverse order.

	It's pretty safe to bet a filesystem can be discommissioned with nothing
	but umount and rm calls if necessary, but this is used so on calm
	shutdowns we can do logging and etc.  Teardown operations are also allowed
	to included steps which are required for correct operation of output gathering.
*/
type defaultAssembly struct {
	emplacements []rio.Emplacement
}

func (a *defaultAssembly) record(r rio.Emplacement) {
	a.emplacements = append(a.emplacements, r)
}

func (a *defaultAssembly) Teardown() {
	for i := len(a.emplacements) - 1; i >= 0; i-- {
		a.emplacements[i].Teardown()
	}
}
