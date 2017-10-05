package cradle

import (
	"fmt"

	. "github.com/polydawn/go-errcat"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/rio/fs"
	"go.polydawn.net/rio/fsOp"
)

func FormulaDefaults(frm api.Formula) api.Formula {
	frm = frm.Clone()
	// Always fill in a zero userinfo.
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
	// If cradle is disabled, set a zero cwd and skip the rest.
	switch frm.Action.Cradle {
	case "disable":
		frm.Action.Cwd = "/"
		return frm
	default:
		// '/task' is the default when cradle is enabled, because any dir at
		//   all is more sensible than the bare root dir, and since cradle
		//   is enabled, we'll make sure it's writable and ready to go.
		frm.Action.Cwd = "/task"
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
	if err := fsOp.MkdirAll(chrootFs, fs.MustAbsolutePath(string(frm.Action.Cwd)).CoerceRelative(), 0755); err != nil {
		return Errorf(repeatr.ErrJobInvalid, "failed building cradle fs (cwd): %s", err)
	}
	// TODO and ensure fixed perms
	// Foist usable bits onto homedir and parents.
	if err := fsOp.MkdirAll(chrootFs, fs.MustAbsolutePath(string(frm.Action.Userinfo.Homedir)).CoerceRelative(), 0755); err != nil {
		return Errorf(repeatr.ErrJobInvalid, "failed building cradle fs (homedir): %s", err)
	}
	// TODO and ensure fixed perms
	// Force standard tempdir bits onto /tmp.
	defer fsOp.RepairMtime(chrootFs, fs.MustRelPath("."))()
	if err := fsOp.MkdirAll(chrootFs, fs.MustRelPath("./tmp"), 01777); err != nil {
		return Errorf(repeatr.ErrJobInvalid, "failed building cradle fs (tmp): %s", err)
	}
	// TODO and ensure fixed perms
	return nil
}

// TODO func DirpropsForUserinfo(api.FormulaUserinfo) fs.Metadata
