package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// list of versions.  list of names of formulas we're about to run, really.
// this doesn't map to git commits because the formulas do; we're intentionally
// making the formulas standalone as usual (you should be able to pastebin them).
var versions = []string{
	"v0.13",
	"v0.12",
	"v0.11",
	"v0.10",
	"v0.9",
	"v0.8",
	"v0.7",
	"v0.6",
	"v0.5",
}

var exit = 0

func main() {
	if err := os.Chdir("./meta/releases"); err != nil {
		panic(err)
	}
	for _, version := range versions {
		artifacts := BuildRelease(version)
		CheckLinks(version, artifacts)
	}
	os.Exit(exit)
}

func BuildRelease(version string) map[string]string {
	fmt.Fprintf(os.Stderr, "=======> running build for %s =======>\n", version)
	cmd := exec.Command(
		"repeatr", "run",
		"repeatr-release-"+version+".frm",
	)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	fmt.Fprintf(os.Stderr, "<======= completed build for %s <=======\n", version)
	if err != nil {
		panic(err)
	}
	var frm map[string]interface{}
	if err := json.Unmarshal(out, &frm); err != nil {
		panic(err)
	}
	artifacts := make(map[string]string)
	for k, v := range frm["results"].(map[string]interface{}) {
		if k[0] == '$' {
			continue
		}
		artifacts[k] = v.(map[string]interface{})["hash"].(string)
	}
	return artifacts
}

const sigil = "\033[44;32m≡≡≡\033[0m  "

func CheckLinks(version string, artifacts map[string]string) {
	versionPath := "dl/repeatr-" + version
	os.Mkdir("dl", 0755)
	os.Mkdir(versionPath, 0755)
	for name, hash := range artifacts {
		warePath := filepath.Join("../../wares/", hash)
		linkPath := filepath.Join(versionPath, name) + ".tar.gz"
		currentRef, err := os.Readlink(linkPath)
		if os.IsNotExist(err) {
			os.Symlink(warePath, linkPath)
			fmt.Fprintf(os.Stdout, sigil+"%q: linked to %q\n", linkPath, hash)
			continue
		}
		currentRef = currentRef[strings.LastIndex(currentRef, "/")+1:]
		if currentRef == hash {
			fmt.Fprintf(os.Stdout, sigil+"%q: already linked to %q\n", linkPath, hash)
			continue
		}
		fmt.Fprintf(os.Stdout, sigil+"%q: MISMATCH!  recorded as %q; latest result is %q\n", linkPath, currentRef, hash)
		exit = 5
	}
}
