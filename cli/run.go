package cli

import (
	. "fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"github.com/ugorji/go/codec"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/scheduler"
)

func LoadFormulaFromFile(path string) def.Formula {
	filename, _ := filepath.Abs(path)

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		Println(err)
		Println("Could not read file", filename)
		os.Exit(1)
	}

	dec := codec.NewDecoderBytes(content, &codec.JsonHandle{})

	formula := def.Formula{}
	dec.MustDecode(&formula)

	return formula
}

func RunFormulae(s *scheduler.Scheduler, e *executor.Executor, f ...def.Formula) {
	try.Do(func() {

		(*s).Configure(e)
		(*s).Start()

		// Queue each job as the scheduler deigns to read from the channel
		go func() {
			for _, formula := range f {
				(*s).Queue() <- formula
			}
		}()

		exitCode := 0

		// Get job results in order
		// Could obviously be improved by out-of-order
		for x := 0; x < len(f); x++ {
			if len(f) > 1 {
				Println("Running formula", x+1)
			}
			job := <-(*s).Results()
			result := job.Wait()

			Println("Job finished with code", result.ExitCode, "Outputs:", result.Outputs)

			if result.Error != nil {
				Println("Problem executing job:", result.Error)
			}
		}

		// DISCUSS: we could consider any non-zero exit a Error, but having that distinct from execution problems makes sense.
		// This is clearly silly and placeholder.
		os.Exit(exitCode)

	}).Catch(def.ValidationError, func(e *errors.Error) {
		// TODO: I think this is off the goroutine now, whelp
		Println(e.Message())
		os.Exit(2)
	}).Done()
}
