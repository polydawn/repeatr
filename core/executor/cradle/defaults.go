package cradle

import (
	"polydawn.net/repeatr/api/def"
)

/*
	Apply default configuration, returning a new formula with the changes.
*/
func ApplyDefaults(frm *def.Formula) *def.Formula {
	frm = frm.Clone()
	if frm.Action.Policy == "" {
		frm.Action.Policy = def.PolicyRoutine
	}
	frm.Action.Env.Merge(DefaultEnv(frm.Action.Policy))
	if frm.Action.Cwd == "" {
		frm.Action.Cwd = DefaultCwd()
	}
	return frm
}

func DefaultEnv(p def.Policy) def.Env {
	return def.Env{
		"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME": UserinfoForPolicy(p).Home,
	}
}

func DefaultCwd() string {
	return "/task"
}
