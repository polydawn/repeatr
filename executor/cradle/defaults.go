package cradle

import (
	"polydawn.net/repeatr/def"
)

func ApplyDefaults(frm def.Formula) def.Formula {
	// TODO frm = *frm.Clone()
	// set cwd if blank
	if frm.Action.Cwd == "" {
		frm.Action.Cwd = DefaultCwd()
	}
	// merge env
	for k, v := range Env() {
		if _, ok := frm.Action.Env[k]; !ok {
			frm.Action.Env[k] = v
		}
	}
	// done
	return frm
}

func Env() def.Env {
	return def.Env{
		"PATH": "",
		"HOME": "",
	}
}

func DefaultCwd() string {
	return "/task"
}
