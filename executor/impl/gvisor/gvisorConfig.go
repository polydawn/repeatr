package gvisor

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
		},
		"root": map[string]interface{}{
			"path":     rootPath,
			"readonly": false,
		},
		"hostname": hostname,
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
