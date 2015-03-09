package nsinit

import (
	. "fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/input/dispatch"
	"polydawn.net/repeatr/lib/flak"
	"polydawn.net/repeatr/output/dispatch"
)

// interface assertion
var _ executor.Executor = &Executor{}

type Executor struct {
}

// Execute a forumla in a specified directory.
// Directory is assumed to exist.
func (*Executor) Execute(job def.Formula, d string) (def.Job, []def.Output) {

	// Dedicated rootfs folder to distinguish container from nsinit noise
	rootfs := filepath.Join(d, "rootfs")
	err := os.MkdirAll(rootfs, 0777)
	if err != nil {
		panic(errors.IOError.Wrap(err))
	}

	// nsinit wants to have a legferl
	logFile := filepath.Join(d, "nsinit-debug.log")

	// Prep command
	args := []string{}

	// Global options:
	// --root will place the 'nsinit' folder (holding a state.json file) in d
	// --log-file does much the same with a log file (unsure if care?)
	// --debug allegedly enables debug output in the log file
	args = append(args, "--root", d, "--log-file", logFile, "--debug")

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
	// TODO: replace with mounts
	for x, input := range job.Inputs {
		Println("Provisioning input", x+1, input.Type, "to", input.Location)
		path := filepath.Join(rootfs, input.Location)

		err = <-inputs.Get(input).Apply(path)
		if err != nil {
			Println("Input", x+1, "failed:", err)
			panic(err)
		}
	}

	// Output folders should exist
	// TODO: replace with mounts
	for _, output := range job.Outputs {
		path := filepath.Join(rootfs, output.Location)
		err := os.MkdirAll(path, 0777)
		if err != nil {
			panic(errors.IOError.Wrap(err))
		}
	}

	Println("Running formula...")
	cmd.Run()

	// Run outputs
	for x, output := range job.Outputs {
		Println("Persisting output", x+1, output.Type, "from", output.Location)
		// path := filepath.Join(rootfs, output.Location)

		err = <-outputs.Get(output).Apply(rootfs)
		if err != nil {
			Println("Output", x+1, "failed:", err)
			panic(err)
		}
	}

	// Done... ish. No outputs. Womp womp!
	return job, nil
}

func (e *Executor) Run(job def.Formula) (def.Job, []def.Output) {
	// Prepare the forumla for execution on this host
	def.ValidateAll(&job)

	var resultJob def.Job
	var outputs []def.Output

	flak.WithTempDir(func(d string) {
		resultJob, outputs = e.Execute(job, d)
	}, "nsinit")

	Println("Done!")
	return resultJob, outputs
}
