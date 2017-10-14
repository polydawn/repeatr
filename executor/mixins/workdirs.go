package mixins

import (
	. "github.com/polydawn/go-errcat"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/rio/fs"
	"go.polydawn.net/rio/fs/osfs"
	"go.polydawn.net/rio/fsOp"
)

// Make work dirs.
//  Including whole workspace dir and parents, if necessary.
//
// The runrecord need only have gotten past `mixins.InitRunRecord` so far
// (we use it for its guid).
func MakeWorkDirs(workspaceFs fs.FS, rr api.RunRecord) (
	jobPath fs.RelPath, // Relative to workspaceFs.
	chrootFs fs.FS, // New osfs handle for the chroot (conincidentally inside jobPath).
	err error,
) {
	if err := fsOp.MkdirAll(osfs.New(fs.AbsolutePath{}), workspaceFs.BasePath().CoerceRelative(), 0700); err != nil {
		return fs.RelPath{}, nil, Errorf(repeatr.ErrLocalCacheProblem, "cannot initialize workspace dirs: %s", err)
	}
	jobPath = fs.MustRelPath(rr.Guid)
	chrootPath := jobPath.Join(fs.MustRelPath("chroot"))
	if err := workspaceFs.Mkdir(jobPath, 0700); err != nil {
		return fs.RelPath{}, nil, Recategorize(repeatr.ErrLocalCacheProblem, err)
	}
	if err := workspaceFs.Mkdir(chrootPath, 0755); err != nil {
		return fs.RelPath{}, nil, Recategorize(repeatr.ErrLocalCacheProblem, err)
	}
	chrootFs = osfs.New(workspaceFs.BasePath().Join(chrootPath))
	return
}
