package nsinit

import (
	. "fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/lib/flak"
)

// interface assertion
var _ executor.Executor = &Executor{}

type Executor struct {
}

// Can generalize & relocate

func (*Executor) Run(job def.Formula) (def.Job, []def.Output) {
	// Where we'll put the rootfs
	base := flak.GetTempDir("nsinit")
	rootfs := filepath.Join(base, "rootfs")

	err := os.MkdirAll(rootfs, 0777)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}

	// nsinit wants to have a legferl
	logFile := filepath.Join(base, "nsinit-debug.log")

	// DISCUSS: consider doing this in the CLI before the executor gets it?
	// Probably not; future executors may want to do specific subsets of validation at specific times (see def/validate.go)
	def.ValidateAll(&job)

	// Prep command
	args := []string{}

	// Global options:
	// --root will place the 'nsinit' folder (holding a state.json file) in base
	// --log-file does much the same with a log file (unsure if care?)
	// --debug allegedly enables debug output in the log file
	args = append(args, "--root", base, "--log-file", logFile, "--debug")

	// Subcommand, and tell nsinit to not desire a JSON file (instead just use many flergs)
	args = append(args, "exec", "--create")

	// Lol-networking, a giant glorious TODO.
	args = append(args, "--veth-bridge", "docker0", "--veth-address", "172.17.0.101/16", "--veth-gateway", "172.17.42.1", "--veth-mtu", "1500")

	// For now, interactive attach. Debuggery.
	// Eventually, replace with uh... siphon... vodoo... and an Accent?
	args = append(args, "--tty")

	// Where our system image exists
	args = append(args, "--rootfs", rootfs)

	// Add all desired environment variables
	for k, v := range job.Accents.Env {
		args = append(args, "--env", k+"="+v)
	}

	// Unroll command args
	args = append(args, job.Accents.Entrypoint...)

	// For now, run in this terminal
	cmd := exec.Command("nsinit", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run inputs
	// ( discussion: replace with mounts? )
	Println("Provisioning inputs...")
	for _, input := range job.Inputs {
		path := filepath.Join(rootfs, input.Location)
		err := os.MkdirAll(path, 0777)
		if err != nil {
			panic(errors.IOError.Wrap(err))
		}

		tar := exec.Command("tar", "-xf", input.URI, "-C", path)
		tar.Stdin = os.Stdin
		tar.Stdout = os.Stdout
		tar.Stderr = os.Stderr
		tar.Run()

		// Eventually:
		// err := <- dispatch.GetInput(input).Apply(path)
	}

	// Output folders should exist
	// ( discussion: replace with mounts? )
	for _, output := range job.Outputs {
		path := filepath.Join(rootfs, output.Location)
		err := os.MkdirAll(path, 0777)
		if err != nil {
			panic(errors.IOError.Wrap(err))
		}
	}

	Println("Running formula...")
	cmd.Run()

	Println("Persisting outputs...")
	for _, output := range job.Outputs {
		path := filepath.Join(rootfs, output.Location)

		// Assumes output is a folder. Output transport impls should obviously be more robust
		tar := exec.Command("tar", "-cf", output.URI, "--xform", "s,"+strings.TrimLeft(rootfs, "/")+",,", path)
		tar.Stdin = os.Stdin
		tar.Stdout = os.Stdout
		tar.Stderr = os.Stderr
		tar.Run()

		// Eventually:
		// err := <- dispatch.GetOutput(output).Dream()
	}

	Println("Cleaning up...")
	err = os.RemoveAll(base)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}

	// Done... ish. No outputs or job result. Womp womp!
	Println("Done!")
	return nil, nil
}
