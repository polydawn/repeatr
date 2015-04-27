package placer

import (
	"path/filepath"
	"sort"

	"polydawn.net/repeatr/io"
	"polydawn.net/repeatr/lib/fs"
)

type defaultAssembler struct {
	Placer integrity.Placer
}

var _ integrity.Assembler = defaultAssembler{}.Assemble

func (a defaultAssembler) Assemble(basePath string, mounts []integrity.AssemblyPart) integrity.Assembly {
	sort.Sort(integrity.AssemblyPartsByPath(mounts))
	housekeeping := &defaultAssembly{}
	for _, mount := range mounts {
		destBasePath := filepath.Join(basePath, mount.TargetPath)
		fs.MkdirAll(destBasePath)
		housekeeping.record(a.Placer(mount.SourcePath, destBasePath, mount.Writable))
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
	emplacements []integrity.Emplacement
}

func (a *defaultAssembly) record(r integrity.Emplacement) {
	a.emplacements = append(a.emplacements, r)
}

func (a *defaultAssembly) Teardown() {
	for i := len(a.emplacements) - 1; i >= 0; i-- {
		a.emplacements[i].Teardown()
	}
}
