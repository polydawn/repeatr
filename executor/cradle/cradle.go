package cradle

import (
	"fmt"
	"time"

	. "github.com/warpfork/go-errcat"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/rio/fs"
	"go.polydawn.net/rio/fsOp"
)

func FormulaDefaults(frm api.Formula) api.Formula {
	frm = frm.Clone()

	//
	// Some values always need to be filled in if blank:
	//

	// Always fill in a non-nil userinfo.
	if frm.Action.Userinfo == nil {
		frm.Action.Userinfo = &api.FormulaUserinfo{}
	}
	// Always fill in *some* UID and GID and userinfo.
	if frm.Action.Userinfo.Uid == nil {
		frm.Action.Userinfo.Uid = ptrint(1000)
	}
	if frm.Action.Userinfo.Gid == nil {
		frm.Action.Userinfo.Gid = ptrint(1000)
	}

	//
	// Some values vary in their treatment depending on if 'cradle' is enabled:
	//

	// If cradle is disabled, set a zero cwd and skip the rest.
	if frm.Action.Cwd == "" {
		switch frm.Action.Cradle {
		case "disable":
			frm.Action.Cwd = "/"
		default:
			// '/task' is the default when cradle is enabled, because any dir at
			//   all is more sensible than the bare root dir, and since cradle
			//   is enabled, we'll make sure it's writable and ready to go.
			frm.Action.Cwd = "/task"
		}
	}

	//
	// The rest of these values are only set if 'cradle' is enabled:
	//

	switch frm.Action.Cradle {
	case "disable":
		return frm
	default:
		// pass
	}
	// Compute remainder of userinfo.
	//  (These aren't used if cradle=disabled, so we don't set them until now.)
	if frm.Action.Userinfo.Username == "" {
		switch *frm.Action.Userinfo.Uid {
		case 0:
			frm.Action.Userinfo.Username = "root"
		default:
			frm.Action.Userinfo.Username = "reuser"
		}
	}
	if frm.Action.Userinfo.Homedir == "" {
		switch *frm.Action.Userinfo.Uid {
		case 0:
			frm.Action.Userinfo.Homedir = api.AbsPath("/root")
		default:
			frm.Action.Userinfo.Homedir = api.AbsPath(fmt.Sprintf("/home/%s", frm.Action.Userinfo.Username))
		}
	}
	// Fold userinfo values back into env.
	if frm.Action.Env == nil {
		frm.Action.Env = map[string]string{}
	}
	if _, exists := frm.Action.Env["HOME"]; !exists {
		frm.Action.Env["HOME"] = string(frm.Action.Userinfo.Homedir)
	}
	if _, exists := frm.Action.Env["USER"]; !exists {
		frm.Action.Env["USER"] = string(frm.Action.Userinfo.Username)
	}
	if _, exists := frm.Action.Env["PATH"]; !exists {
		frm.Action.Env["PATH"] = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	}
	return frm
}
func ptrint(i int) *int { return &i }

func TidyFilesystem(frm api.Formula, chrootFs fs.FS) error {
	switch frm.Action.Cradle {
	case "disable":
		return nil
	default:
	}
	// Foist usable bits onto cwd and parents.
	if err := fsOp.MkdirUsable(chrootFs, fs.MustAbsolutePath(string(frm.Action.Cwd)).CoerceRelative(), DirpropsForUserinfo(*frm.Action.Userinfo)); err != nil {
		return Errorf(repeatr.ErrJobInvalid, "failed building cradle fs (cwd): %s", err)
	}
	// Foist usable bits onto homedir and parents.
	if err := fsOp.MkdirUsable(chrootFs, fs.MustAbsolutePath(string(frm.Action.Userinfo.Homedir)).CoerceRelative(), DirpropsForUserinfo(*frm.Action.Userinfo)); err != nil {
		return Errorf(repeatr.ErrJobInvalid, "failed building cradle fs (homedir): %s", err)
	}
	// Force standard tempdir bits onto /tmp.
	tmpPath := fs.MustRelPath("tmp")
	defer fsOp.RepairMtime(chrootFs, fs.RelPath{})()
	defer fsOp.RepairMtime(chrootFs, tmpPath)()
	if err := fsOp.MkdirAll(chrootFs, tmpPath, 01777); err != nil {
		return Errorf(repeatr.ErrJobInvalid, "failed building cradle fs (tmp): %s", err)
	}
	if err := chrootFs.Chmod(tmpPath, 01777); err != nil {
		return Errorf(repeatr.ErrJobInvalid, "failed building cradle fs (tmp): %s", err)
	}
	return nil
}

func DirpropsForUserinfo(userinfo api.FormulaUserinfo) fs.Metadata {
	return fs.Metadata{
		Type:  fs.Type_Dir,
		Perms: 0755,
		Uid:   uint32(*userinfo.Uid),
		Gid:   uint32(*userinfo.Gid),
		Mtime: time.Unix(api.DefaultTime, 0),
	}
}
