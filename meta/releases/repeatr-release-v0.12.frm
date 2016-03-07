inputs:
	"/":
		type: "tar"
		hash: "aLMH4qK1EdlPDavdhErOs0BPxqO0i6lUaeRE4DuUmnNMxhHtF56gkoeSulvwWNqT"
		silo: "http+ca://repeatr.s3.amazonaws.com/assets/"
	"/app/go/":
		type: "tar"
		hash: "37KlJTWDLFzke1kOtzKYavFek77EB91AzROQty-S-f50wQ0jDuifrbbqN8_McxQC"
		silo: "https://storage.googleapis.com/golang/go1.6.linux-amd64.tar.gz"
	"/task/repeatr/":
		type: "git"
		hash: "1d280923b7b1e8621375e545bbc206a1c30ddad0"
		silo:
			- "./../.."
			- "https://github.com/polydawn/repeatr.git"
action:
	cwd: "/task/repeatr/"
	env:
		"PATH": "/bin/:/usr/bin/:/app/go/go/bin/"
		"GOROOT": "/app/go/go"
		"GITCOMMIT": "1d280923b7b1e8621375e545bbc206a1c30ddad0"
		"BUILDDATE": "2016-03-07 10:21:03-08:00"
		"GOOS": "linux"
		"GOARCH": "amd64"
	command: [ "./goad", "install" ]
outputs:
	"repeatr-linux-amd64-v0.12":
		mount: "/task/repeatr/.gopath/bin/"
		type: "tar"
		silo: "file+ca://./wares/"
