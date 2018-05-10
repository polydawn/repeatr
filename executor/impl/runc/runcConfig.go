package runc

import (
	"io/ioutil"

	"github.com/polydawn/refmt"
	"github.com/polydawn/refmt/json"
	. "github.com/warpfork/go-errcat"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	"go.polydawn.net/repeatr/executor/policy"
)

func templateRuncConfig(jobID string, action api.FormulaAction, rootPath string, tty bool) (interface{}, error) {
	caps, err := policy.GetCapsForPolicy(action.Policy)
	if err != nil {
		return nil, err
	}
	capsStrs := policy.CapsToStrings(caps)
	hostname := action.Hostname
	if hostname == "" {
		hostname = string(jobID)
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
				"uid":            *action.Userinfo.Uid,
				"gid":            *action.Userinfo.Gid,
				"additionalGids": nil,
			},
			"args": action.Exec,
			"env": func() (env []string) {
				for k, v := range action.Env {
					env = append(env, k+"="+v)
				}
				return
			}(),
			"cwd": action.Cwd,

			"capabilities": map[string]interface{}{
				"bounding":    capsStrs,
				"effective":   capsStrs,
				"inheritable": capsStrs,
				"permitted":   capsStrs,
				"ambient":     capsStrs,
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
			"path":     rootPath,
			"readonly": false,
		},
		"hostname": hostname,
		"mounts": []interface{}{
			map[string]interface{}{
				"destination": "/proc",
				"type":        "proc",
				"source":      "proc",
			},
			map[string]interface{}{
				// Note that this mount causes a LOT of magic to be implied.
				// Runc takes the existence of this as an instruction
				// to populate it with a bunch of device nodes and symlink.
				//
				// Somewhat wildly, the only way to opt *out* of this
				// is *not* in fact to refrain from making this mount,
				// but actually to bind *something* into this position:
				// https://github.com/opencontainers/runc/blob/94cfb7955b8460e0f4943e3a18a6fe6b45d9d8d3/libcontainer/rootfs_linux.go#L30
				"destination": "/dev",
				"type":        "tmpfs",
				"source":      "tmpfs",
				"options": []string{
					"nosuid",
					"strictatime",
					"mode=755",
					"size=65536k",
				},
			},
			map[string]interface{}{
				// This, together with /dev, is an implicit requirement
				// for interactive mode to work: one of the first things
				// runc does when setting up a terminal is attempt to
				// open /dev/ptmx, which is a symlink pointing into here.
				"destination": "/dev/pts",
				"type":        "devpts",
				"source":      "devpts",
				"options": []string{
					"nosuid",
					"noexec",
					"newinstance",
					"ptmxmode=0666",
					"mode=0620",
					"gid=5", // alarming magic number
				},
			},
			map[string]interface{}{
				// "/dev/shm" is not a requirement of posix or anything,
				// but good luck running a wide variety of desktop
				// applications without it; it's a defacto standard.
				"destination": "/dev/shm",
				"type":        "tmpfs",
				"source":      "shm",
				"options": []string{
					"nosuid",
					"noexec",
					"nodev",
					"mode=1777",
					"size=65536k",
				},
			},
			map[string]interface{}{
				"destination": "/dev/mqueue",
				"type":        "mqueue",
				"source":      "mqueue",
				"options": []string{
					"nosuid",
					"noexec",
					"nodev",
				},
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
	}, nil
}

func writeConfigToFile(path string, runcCfg interface{}) error {
	runcCfgBytes, err := refmt.Marshal(json.EncodeOptions{}, runcCfg)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(path, runcCfgBytes, 0600); err != nil {
		return Recategorize(repeatr.ErrLocalCacheProblem, err)
	}
	return nil
}
