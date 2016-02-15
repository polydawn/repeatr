package cradle

import (
	"polydawn.net/repeatr/def"
)

/*
	Apply default configuration, returning a new formula with the changes.
*/
func ApplyDefaults(frm *def.Formula) *def.Formula {
	frm = frm.Clone()
	frm.Action.Env.Merge(DefaultEnv())
	if frm.Action.Cwd == "" {
		frm.Action.Cwd = DefaultCwd()
	}
	return frm
}

func DefaultEnv() def.Env {
	return def.Env{
		"PATH": "",
		"HOME": "",
	}
}

func DefaultCwd() string {
	return "/task"
}
