package cli

import (
	. "fmt"
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
	(*s).Configure(e)
	(*s).Start()

	var wg sync.WaitGroup

	// Queue each job as the scheduler deigns to read from the channel
	for x, formula := range f {
		wg.Add(1)

		go func() {
			defer wg.Done()
			job := <- (*s).Schedule(formula)

			Println("Job", x, job.Id(), "queued")
			result := job.Wait()

			Println("Job finished with code", result.ExitCode, "Outputs:", result.Outputs)

			if result.Error != nil {
				Println("Problem executing job:", result.Error)
			}
		}()
	}

	wg.Wait()
}
