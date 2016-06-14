package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"

	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/api/hitch"
	"polydawn.net/repeatr/core/executor"
	"polydawn.net/repeatr/lib/guid"
)

func LoadFormulaFromFile(path string) (frm def.Formula) {
	filename, _ := filepath.Abs(path)

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(Error.Wrap(fmt.Errorf("Could not read formula file %q: %s", filename, err)))
	}

	try.Do(func() {
		frm = *hitch.ParseYaml(content)
	}).Catch(def.ConfigError, func(err *errors.Error) {
		panic(Error.Wrap(err))
	}).Done()
	return
}

func RunFormula(e executor.Executor, formula def.Formula, journal io.Writer) executor.JobResult {
	// Set up a logger.
	log := log15.New()
	log.SetHandler(log15.StreamHandler(journal, log15.TerminalFormat()))

	jobID := executor.JobID(guid.New())
	log = log.New(log15.Ctx{"runID": jobID})
	log.Info("Job queued")
	job := e.Start(formula, jobID, nil, log)

	// Stream job output to terminal in real time
	_, err := io.Copy(journal, job.OutputReader())
	if err != nil {
		log.Error("Error reading job stream", "error", err)
		panic(err)
	}

	result := job.Wait()
	if result.Error != nil {
		log.Error("Job execution errored", "error", result.Error.Message())
	} else {
		log.Info("Job finished", log15.Ctx{
			"exit":    result.ExitCode,
			"outputs": result.Outputs,
		})
	}
	return result
}
