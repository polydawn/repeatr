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
		hash: "b15dfa15ffb1f70c3655d06ffaeff8a4a9dd1348"
		silo:
			- "./../.."
			- "https://github.com/polydawn/repeatr.git"
action:
	cwd: "/task/repeatr/"
	env:
		"PATH": "/bin/:/usr/bin/:/app/go/go/bin/"
		"GOROOT": "/app/go/go"
		"GITCOMMIT": "b15dfa15ffb1f70c3655d06ffaeff8a4a9dd1348"
		"BUILDDATE": "2015-10-21 23:45:45-05:00"
		"GOOS": "linux"
		"GOARCH": "amd64"
	command: [ "./goad", "install" ]
outputs:
	"repeatr-linux-amd64-v0.7":
		mount: "/task/repeatr/.gopath/bin/"
		type: "tar"
		silo: "file+ca://./wares/"
