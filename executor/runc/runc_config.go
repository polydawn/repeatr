package runc

import (
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor/cradle"
)

func EmitRuncConfigStruct(frm def.Formula, job def.Job, rootPath string, tty bool) interface{} {
	userinfo := cradle.UserinfoForPolicy(frm.Action.Policy)
	return map[string]interface{}{
		"version": "0.2.0",
		"platform": map[string]interface{}{
			"os":   "linux",
			"arch": "amd64",
		},
		"process": map[string]interface{}{
			"terminal": tty,
			"user": map[string]interface{}{
				"uid":            userinfo.Uid,
				"gid":            userinfo.Gid,
				"additionalGids": nil,
			},
			"args": frm.Action.Entrypoint,
			"env": func() (env []string) {
				for k, v := range frm.Action.Env {
					env = append(env, k+"="+v)
				}
				return
			}(),
			"cwd": frm.Action.Cwd,
		},
		"root": map[string]interface{}{
			"path":     rootPath,
			"readonly": false,
		},
		"hostname": string(job.Id()),
		"mounts": []interface{}{
			map[string]interface{}{
				"name": "proc",
				"path": "/proc",
			},
		},
		"linux": map[string]interface{}{
			"capabilities": []interface{}{
				"CAP_AUDIT_WRITE",
				"CAP_KILL",
			},
		},
	}
}

func EmitRuncRuntimeStruct(_ def.Formula) interface{} {
	return map[string]interface{}{
		"mounts": map[string]interface{}{
			"proc": map[string]interface{}{
				"type":    "proc",
				"source":  "proc",
				"options": nil,
			},
		},
		"linux": map[string]interface{}{
			"resources": map[string]interface{}{},
			"namespaces": []interface{}{
				map[string]interface{}{
					"type": "pid",
					"path": "",
				},
				map[string]interface{}{
					"type": "ipc",
					"path": "",
				},
				map[string]interface{}{
					"type": "uts",
					"path": "",
				},
				map[string]interface{}{
					"type": "mount",
					"path": "",
				},
			},
			"devices": []interface{}{
				map[string]interface{}{
					"path":        "/dev/null",
					"type":        99,
					"major":       1,
					"minor":       3,
					"permissions": "rwm",
					"fileMode":    438,
					"uid":         0,
					"gid":         0,
				},
				map[string]interface{}{
					"path":        "/dev/random",
					"type":        99,
					"major":       1,
					"minor":       8,
					"permissions": "rwm",
					"fileMode":    438,
					"uid":         0,
					"gid":         0,
				},
				map[string]interface{}{
					"path":        "/dev/full",
					"type":        99,
					"major":       1,
					"minor":       7,
					"permissions": "rwm",
					"fileMode":    438,
					"uid":         0,
					"gid":         0,
				},
				map[string]interface{}{
					"path":        "/dev/tty",
					"type":        99,
					"major":       5,
					"minor":       0,
					"permissions": "rwm",
					"fileMode":    438,
					"uid":         0,
					"gid":         0,
				},
				map[string]interface{}{
					"path":        "/dev/zero",
					"type":        99,
					"major":       1,
					"minor":       5,
					"permissions": "rwm",
					"fileMode":    438,
					"uid":         0,
					"gid":         0,
				},
				map[string]interface{}{
					"path":        "/dev/urandom",
					"type":        99,
					"major":       1,
					"minor":       9,
					"permissions": "rwm",
					"fileMode":    438,
					"uid":         0,
					"gid":         0,
				},
			},
		},
	}
}
