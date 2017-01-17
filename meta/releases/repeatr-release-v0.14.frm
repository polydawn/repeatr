inputs:
	"/":
		type: "tar"
		hash: "aLMH4qK1EdlPDavdhErOs0BPxqO0i6lUaeRE4DuUmnNMxhHtF56gkoeSulvwWNqT"
		silo: "http+ca://repeatr.s3.amazonaws.com/assets/"
	"/app/go/":
		type: "tar"
		hash: "UZlcQvHU5Qg5gXquDW2QTc-v6l-LzR1-8SRmJ2aWQjiqFcPggIETikoEta_DJZQc"
		silo: "https://storage.googleapis.com/golang/go1.7.4.linux-amd64.tar.gz"
	"/task/repeatr/":
		type: "git"
		hash: "72b19be26bbac56c9d1dc418a01bfe0b09a9363c"
		silo:
			- "./../.."
			- "https://github.com/polydawn/repeatr.git"
action:
	cwd: "/task/repeatr/"
	env:
		"PATH": "/bin/:/usr/bin/:/app/go/go/bin/"
		"GOROOT": "/app/go/go"
		"GITCOMMIT": "72b19be26bbac56c9d1dc418a01bfe0b09a9363c"
		"BUILDDATE": "2017-01-17 11:23:48-06:00"
		"GOOS": "linux"
		"GOARCH": "amd64"
	command: [ "./goad", "install" ]
outputs:
	"repeatr-linux-amd64-v0.14":
		mount: "/task/repeatr/.gopath/bin/"
		type: "tar"
		silo: "file+ca://./wares/"
