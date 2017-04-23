package null

import (
	"crypto/sha512"
	"encoding/base64"
	"io"
	"sort"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/ugorji/go/codec"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/core/executor"
	"go.polydawn.net/repeatr/core/executor/basicjob"
	"go.polydawn.net/repeatr/lib/guid"
)

var _ executor.Executor = &Executor{}

type Mode int

const (
	Deterministic Mode = iota
	Nondeterministic
	SadExit
	Erroring
)

type Executor struct {
	Mode Mode
}

func (*Executor) Configure(workspacePath string) {
}

func (e *Executor) Start(f def.Formula, id executor.JobID, stdin io.Reader, _ log15.Logger) executor.Job {
	job := basicjob.New(id)
	job.Result = executor.JobResult{
		ID:       id,
		ExitCode: -1,
	}

	go func() {
		switch e.Mode {
		case Deterministic:
			// seed hash with action and sorted input hashes.
			// (a real formula would behave a little differently around names v paths, but eh.)
			// ... actually this is basically the same as the stage2 identity.  yeah.  tis.
			// which arguably makes it kind of a dangerously cyclic for some tests.  but there's
			// not much to be done about that aside from using real executors in your tests then.
			hasher := sha512.New384()
			hasher.Write([]byte(formulaHash(f)))

			// aside: fuck you golang for making me write this goddamned sort AGAIN.
			// my kingdom for a goddamn sorted map.
			keys := make([]string, len(f.Outputs))
			var i int
			for k := range f.Outputs {
				keys[i] = k
				i++
			}
			sort.Strings(keys)

			// emit outputs, using their names in sorted order to predictably advance
			//  the hash state, while drawing their ids back.
			job.Result.Outputs = def.OutputGroup{}
			for _, name := range keys {
				hasher.Write([]byte(name))
				job.Result.Outputs[name] = f.Outputs[name].Clone()
				job.Result.Outputs[name].Hash = base64.URLEncoding.EncodeToString(hasher.Sum(nil))
			}
		case Nondeterministic:
			job.Result.Outputs = def.OutputGroup{}
			for name, spec := range f.Outputs {
				job.Result.Outputs[name] = spec.Clone()
				job.Result.Outputs[name].Hash = guid.New()
			}
		case SadExit:
			job.Result.ExitCode = 4
		case Erroring:
			job.Result.Error = executor.TaskExecError.New("mock error").(*errors.Error)
		default:
			panic("no")
		}

		close(job.WaitChan)
	}()

	return job
}

/*
	Computes something like the CA hash of the formula.
	Not exported because format not final, etc.
*/
func formulaHash(f def.Formula) string {
	// San check empty values.  programmer error if set.
	for _, spec := range f.Outputs {
		if spec.Hash != "" {
			panic("stage2 formula with output hash set")
		}
	}
	// Copy and zero other things that we don't want to include in canonical IDs.
	// This is working around lack of useful ways to pass encoding style hints down
	//  with out current libraries.
	f2 := def.Formula(f).Clone()
	for _, spec := range f2.Inputs {
		spec.Warehouses = nil
	}
	for _, spec := range f2.Outputs {
		spec.Warehouses = nil
	}
	// hash the rest, and thar we be
	hasher := sha512.New384()
	codec.NewEncoder(hasher, &codec.CborHandle{}).MustEncode(f2)
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}
