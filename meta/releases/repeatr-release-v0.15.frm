inputs:
	"/":
		type: "tar"
		hash: "aLMH4qK1EdlPDavdhErOs0BPxqO0i6lUaeRE4DuUmnNMxhHtF56gkoeSulvwWNqT"
		silo: "http+ca://repeatr.s3.amazonaws.com/assets/"
	"/app/go/":
		type: "tar"
		hash: "gi0Kpb-VH3TK0UBX6YmpuKsrMAUlxicPrY2YvXPo9sBQm_NsD_hKrn7pmc95zrmM"
		silo: "https://storage.googleapis.com/golang/go1.8.1.linux-amd64.tar.gz"
	"/task/repeatr/":
		type: "git"
		hash: "8fb1a9ebd85eadacc861b0c149221af6808270d4"
		silo:
			- "./../.."
			- "https://github.com/polydawn/repeatr.git"
action:
	cwd: "/task/repeatr/"
	env:
		"PATH": "/bin/:/usr/bin/:/app/go/go/bin/"
		"GOROOT": "/app/go/go"
		"GITCOMMIT": "8fb1a9ebd85eadacc861b0c149221af6808270d4"
		"BUILDDATE": "2017-04-25 21:32:13+02:00"
		"GOOS": "linux"
		"GOARCH": "amd64"
	command: [ "./goad", "install" ]
outputs:
	"repeatr-linux-amd64-v0.15":
		mount: "/task/repeatr/.gopath/bin/"
		type: "tar"
		silo: "file+ca://./wares/"
