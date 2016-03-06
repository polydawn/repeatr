inputs:
	"/":
		type: "tar"
		hash: "aLMH4qK1EdlPDavdhErOs0BPxqO0i6lUaeRE4DuUmnNMxhHtF56gkoeSulvwWNqT"
		silo: "http+ca://repeatr.s3.amazonaws.com/assets/"
	"/app/go/":
		type: "tar"
		hash: "vbl0TwPjBrjoph65IaWxOy-Yl0MZXtXEDKcxodzY0_-inUDq7rPVTEDvqugYpJAH"
		silo: "https://storage.googleapis.com/golang/go1.5.linux-amd64.tar.gz"
	"/task/repeatr/":
		type: "git"
		hash: "11c6ee9e4daadc29019959d1a7a70de142924744"
		silo:
			- "./../.."
			- "https://github.com/polydawn/repeatr.git"
action:
	cwd: "/task/repeatr/"
	env:
		"PATH": "/bin/:/usr/bin/:/app/go/go/bin/"
		"GOROOT": "/app/go/go"
		"GITCOMMIT": "11c6ee9e4daadc29019959d1a7a70de142924744"
		"BUILDDATE": "2015-08-02 21:12:40-05:00"
		"GOOS": "linux"
		"GOARCH": "amd64"
	command: [ "./goad", "install" ]
outputs:
	"repeatr-linux-amd64-v0.5":
		mount: "/task/repeatr/.gopath/bin/"
		type: "tar"
		silo: "file+ca://./wares/"
