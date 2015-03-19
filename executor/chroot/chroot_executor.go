package chroot

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/input"
	"polydawn.net/repeatr/input/dispatch"
	"polydawn.net/repeatr/lib/guid"
)

type Executor struct {
	workspacePath string // default: something like '/var/lib/repeatr/executors/chroot/'.
}

func (x *Executor) Run(formula def.Formula) (job def.Job, outs []def.Output) {
	try.Do(func() {
		job, outs = x.run(formula)
	}).Catch(input.InputError, func(e *errors.Error) {
		// REVIEW: also directly pass input/output system errors up?  or, since we may have to gather several, put them in a group and wrap them in a "prereqs failed" executor error?
		panic(e)
	}).Catch(Error, func(e *errors.Error) {
		panic(e)
	}).CatchAll(func(err error) {
		panic(UnknownError.Wrap(err))
	}).Done()
	return
}

func (x *Executor) run(formula def.Formula) (def.Job, []def.Output) {
	// Prepare the forumla for execution on this host
	def.ValidateAll(&formula)

	// make up a job id
	jobID := def.JobID(guid.New())

	// make a rootfs in our workspace using the jobID
	rootfsPath := filepath.Join(x.workspacePath, string(jobID))
	if err := os.Mkdir(rootfsPath, 0755); err != nil {
		panic(Error.Wrap(errors.IOError.Wrap(err))) // REVIEW: WorkspaceIOError?  or a flag that indicates "wow, super hosed"?
	}

	// prep inputs
	x.prepareInputs(rootfsPath, formula.Inputs)

	// prep outputs
	// TODO implement some outputs!

	// sandbox up and invoke the real job
	x.invokeTask(rootfsPath, formula)

	// commit outputs
	// TODO implement some outputs!

	return nil, nil // FIXME implement def.Job
}

func (x *Executor) prepareInputs(rootfsPath string, inputs []def.Input) {
	for _, input := range inputs {
		path := filepath.Join(rootfsPath, input.Location)

		// Ensure that the parent folder of this input exists
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			panic(Error.Wrap(errors.IOError.Wrap(err)))
		}

		// Run input
		// TODO: all of them, asynchronously.
		err = <-inputdispatch.Get(input).Apply(path)
		if err != nil {
			panic(err)
		}
	}
}

func (x *Executor) invokeTask(rootfsPath string, formula def.Formula) {
	// REVIEW: method sig kind of hints we should gather all the "task" related parts together so we don't have to pass the whole formula here

	// chroot's are pretty easy.
	cmdName := formula.Accents.Entrypoint[0]
	cmd := exec.Command(cmdName, formula.Accents.Entrypoint[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:    rootfsPath,
		Pdeathsig: syscall.SIGKILL,
	}

	// god have mercy FIXME
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		// TODO: i'd love to report executable-not-found (very) differently from other major blowups, syscall fails, etc.
		panic(TaskExecError.Wrap(err))
	}
	if err := cmd.Wait(); err != nil {
		// FIXME: do the whole integer error code unwrapping shenanigans
		panic(err)
	}
}
