{
	"imports": {
		"base": "catalog:early.polydawn.io/monolith/busybash:v1:linux-amd64",
		"go": "catalog:early.hyphae.polydawn.io/go:v1.10:linux-amd64",
		"src": "ingest:git:.:HEAD"
	},
	"steps": {
		"test": {
			"operation": {
				"inputs": {
					"/": "base",
					"/app/go": "go",
					"/task": "src"
				},
				"action": {
					"exec": [
						"/bin/bash", "-c",
						"export PATH=$PATH:/app/go/go/bin && export GOPATH=$PWD/.gopath && go test ./..."
					]
				}
			}
		},
		"build": {
			"operation": {
				"inputs": {
					"/": "base",
					"/app/go": "go",
					"/task": "src"
				},
				"action": {
					"exec": [
						"/bin/bash", "-c",
						"export PATH=$PATH:/app/go/go/bin && export GOPATH=$PWD/.gopath && export GOBIN=$PWD/out/bin && go install ./cmd/*"
					]
				},
				"outputs": {
					"bin": "/task/out"
				}
			}
		},
		"smoke-test": {
			"operation": {
				"inputs": {
					"/": "base",
					"/task": "build.bin"
				},
				"action": {
					"exec": [
						"/bin/bash", "-c",
						"./bin/repeatr -h"
					]
				}
			}
		}
	},
	"exports": {
		"bin-linux-amd64": "build.bin",
		"src": "src"
	}
}
