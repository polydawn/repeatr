package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/inconshreveable/log15"
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
		// All job progress reporting, still copy to our shared journal stream.
		// This func might now be outdated; but we haven't decided what any of this
		//  should look like if take a lurch toward supporting cluster farming.
		//  (It might make sense to have a structural comms layer?  Or, maybe plain
		//  byte streams are best for sanity conservation.  Either way: not today.)
		return journal
	}

	s.Configure(e, len(f), jobLoggerFactory) // we know exactly how many forumlae will be enqueued
	s.Start()

	// Set up a logger.
	logger := log15.New()
	logger.SetHandler(log15.StreamHandler(journal, log15.TerminalFormat()))

	// Queue each job as the scheduler deigns to read from the channel
	var wg sync.WaitGroup
	for _, formula := range f {
		wg.Add(1)

		id, jobChan := s.Schedule(formula)

		go func() {
			defer wg.Done()

			log := log15.New(log15.Ctx{"JobID": id})

			log.Info("Job queued")
			job := <-jobChan
			// TODO need better lifecycle events here.  "starting" here means we might still be in provisioning stage.
			log.Info("Job starting")

			// Stream job output to terminal in real time
			_, err := io.Copy(journal, job.OutputReader())
			if err != nil {
				log.Error("Error reading job stream", "error", err)
				panic(err)
			}

			result := job.Wait()
			if result.Error != nil {
				log.Error("Job execution errored", "error", result.Error.Message())
				allclear = false
			} else {
				log.Info("Job finished", log15.Ctx{
					"exit":    result.ExitCode,
					"outputs": result.Outputs,
				})
			}
		}()
	}

	wg.Wait()
	return
}
