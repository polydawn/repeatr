package mixins

import (
	. "github.com/warpfork/go-errcat"

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
func MakeWorkDirs(workspaceFs fs.FS, rr api.FormulaRunRecord) (
	jobFs fs.FS, // New osfs handle where you can put job-lifetime/tmp files.
	chrootFs fs.FS, // New osfs handle for the chroot (conincidentally inside jobPath).
	err error,
) {
	wsPath := workspaceFs.BasePath()
	if err := fsOp.MkdirAll(osfs.New(fs.AbsolutePath{}), wsPath.CoerceRelative(), 0700); err != nil {
		return nil, nil, Errorf(repeatr.ErrLocalCacheProblem, "cannot initialize workspace dirs: %s", err)
	}
	jobPath := fs.MustRelPath(rr.Guid)
	chrootPath := jobPath.Join(fs.MustRelPath("chroot"))
	if err := workspaceFs.Mkdir(jobPath, 0700); err != nil {
		return nil, nil, Recategorize(repeatr.ErrLocalCacheProblem, err)
	}
	if err := workspaceFs.Mkdir(chrootPath, 0755); err != nil {
		return nil, nil, Recategorize(repeatr.ErrLocalCacheProblem, err)
	}
	return osfs.New(wsPath.Join(jobPath)), osfs.New(wsPath.Join(chrootPath)), nil
}
