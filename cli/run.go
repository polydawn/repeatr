package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/ugorji/go/codec"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/scheduler"
)

func LoadFormulaFromFile(path string) def.Formula {
	filename, _ := filepath.Abs(path)

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(Error.Wrap(fmt.Errorf("Could not read formula file %q: %s", filename, err)))
	}

	dec := codec.NewDecoderBytes(content, &codec.JsonHandle{})

	formula := def.Formula{}
	if err := dec.Decode(&formula); err != nil {
		panic(Error.New("Could not parse formula file %q: %s", filename, err))
	}

	return formula
}

func RunFormulae(s scheduler.Scheduler, e executor.Executor, journal io.Writer, f ...def.Formula) (allclear bool) {
	allclear = true // set to false on the first instance of a problem

	jobLoggerFactory := func(_ def.JobID) io.Writer {
		// All job progress reporting actually still copy to our shared journal stream.
		// This should be replaced with a real logging framework,
		//  at which point we can add tags for jobID before handing out specialized loggers.
		return journal
	}

	s.Configure(e, len(f), jobLoggerFactory) // we know exactly how many forumlae will be enqueued
	s.Start()

	var wg sync.WaitGroup

	// Queue each job as the scheduler deigns to read from the channel
	for x, formula := range f {
		wg.Add(1)

		// gofunc + range = race condition, whoops!
		n := x + 1
		id, jobChan := s.Schedule(formula)

		go func() {
			defer wg.Done()

			fmt.Fprintln(journal, "Job", n, id, "queued")
			job := <-jobChan
			// TODO need better lifecycle events here.  "starting" here means we might still be in provisioning stage.
			fmt.Fprintln(journal, "Job", n, id, "starting")

			// Stream job output to terminal in real time
			// TODO: This ruins stdout / stderr split. Job should probably just expose the Mux interface.
			_, err := io.Copy(journal, job.OutputReader())
			if err != nil {
				// TODO: This is serious, how to handle in CLI context debatable
				fmt.Fprintln(journal, "Error reading job stream")
				panic(err)
			}

			result := job.Wait()
			if result.Error != nil {
				fmt.Fprintln(journal, "Job", n, id, "failed with", result.Error.Message())
				allclear = false
			} else {
				fmt.Fprintln(journal, "Job", n, id, "finished with code", result.ExitCode, "and outputs", result.Outputs)
			}
		}()
	}

	wg.Wait()
	return
}
