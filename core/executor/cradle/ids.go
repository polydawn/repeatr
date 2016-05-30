package cradle

import (
	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/api/def"
)

type Userinfo struct {
	Uid  int
	Gid  int
	Home string
}

/*
	Maps a policy to the user info that policy should drop into
	before the contained process is execed.
*/
func UserinfoForPolicy(p def.Policy) Userinfo {
	switch p {
	case def.PolicyRoutine:
		return Userinfo{
			Uid:  1000,
			Gid:  1000,
			Home: "/home/luser",
		}
	case def.PolicyUidZero,
		def.PolicyGovernor,
		def.PolicySysad:
		return Userinfo{
			Uid:  0,
			Gid:  0,
			Home: "/root",
		}
	default:
		panic(errors.ProgrammerError.New("missing switch case"))
	}
}
