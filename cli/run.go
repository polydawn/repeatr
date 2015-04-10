package cli

import (
	. "fmt"
	"io"
	"io/ioutil"
	"os"
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
		panic(Error.Wrap(Errorf("Could not read formula file %q: %s", filename, err)))
	}

	dec := codec.NewDecoderBytes(content, &codec.JsonHandle{})

	formula := def.Formula{}
	dec.MustDecode(&formula)

	return formula
}

func RunFormulae(s scheduler.Scheduler, e executor.Executor, f ...def.Formula) {
	s.Configure(e, len(f)) // we know exactly how many forumlae will be enqueued
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

			Println("Job", n, id, "queued")
			job := <-jobChan
			Println("Job", n, id, "starting")

			// Stream job output to terminal in real time
			// TODO: This ruins stdout / stderr split. Job should probably just expose the Mux interface.
			_, err := io.Copy(os.Stdout, job.OutputReader())
			if err != nil {
				// TODO: This is serious, how to handle in CLI context debatable
				Println("Error reading job stream")
				panic(err)
			}

			result := job.Wait()
			if result.Error != nil {
				Println("Job", n, id, "failed with", result.Error.Message())
			} else {
				Println("Job", n, id, "finished with code", result.ExitCode, "and outputs", result.Outputs)
			}
		}()
	}

	wg.Wait()
}
