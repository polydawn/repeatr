package runc

import (
	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/executor"
	"go.polydawn.net/repeatr/core/executor/cradle"
)

func EmitRuncConfigStruct(frm def.Formula, job executor.Job, rootPath string, tty bool) interface{} {
	userinfo := cradle.UserinfoForPolicy(frm.Action.Policy)
	hostname := frm.Action.Hostname
	if hostname == "" {
		hostname = string(job.Id())
	}
	return map[string]interface{}{
		"ociVersion": "1.0.0-rc5",
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

			"capabilities": map[string]interface{}{
				"bounding":    cradle.CapsForPolicy(frm.Action.Policy),
				"effective":   cradle.CapsForPolicy(frm.Action.Policy),
				"inheritable": cradle.CapsForPolicy(frm.Action.Policy),
				"permitted":   cradle.CapsForPolicy(frm.Action.Policy),
				"ambient":     cradle.CapsForPolicy(frm.Action.Policy),
			},
			"rlimits": []interface{}{
				map[string]interface{}{
					"type": "RLIMIT_NOFILE",
					"hard": 1024,
					"soft": 1024,
				},
			},
			"noNewPrivileges": true,
		},
		"root": map[string]interface{}{
			"path":     "rootfs",
			"readonly": false,
		},
		"hostname": hostname,
		"mounts": []interface{}{
			map[string]interface{}{
				"destination": "/proc",
				"type":        "proc",
				"source":      "proc",
			},
		},
		"linux": map[string]interface{}{
			"resources": map[string]interface{}{
				"devices": []interface{}{
					map[string]interface{}{
						"allow":  false,
						"access": "rwm",
					},
				},
			},
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
			"maskedPaths": []string{
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug",
				"/sys/firmware",
			},
			"readonlyPaths": []string{
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
		},
	}
}
