package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
	"polydawn.net/repeatr/cli"
)

func main() {
	try.Do(func() {
		cli.Main(os.Args, os.Stderr)
	}).Catch(cli.Error, func(err *errors.Error) {
		// Errors marked as valid user-facing issues get a nice
		// pretty-printed route out, and may include specified exit codes.
		if isDebugMode() {
			// in debug-mode, repanic all the way to death so that we get all of golang's built in log features.
			panic(err)
		} else {
			// print nicely.
			fmt.Fprintf(os.Stderr,
				"Repeatr was unable to complete your request!\n"+
					"%s\n",
				err)
			os.Exit(int(cli.EXIT_USER))
		}
	}).CatchAll(func(err error) {
		// Errors that aren't marked as valid user-facing issues should be
		// logged in preparation for a bug report.
		if isDebugMode() {
			// in debug-mode, repanic all the way to death so that we get all of golang's built in log features.
			panic(err)
		} else {
			// save the error to a file.  we want to keep the stacks, but not scare away the user.
			logPath, saveErr := saveErrorReport(err)
			var saveMsg string
			if saveErr == nil {
				saveMsg = fmt.Sprintf("We've logged the full error to a file: %q.  Please include this in the report.", logPath)
			} else {
				saveMsg = fmt.Sprintf("Additionally, we were unable to save a full log of the problem (\"%s\").", saveErr)
			}
			fmt.Fprintf(os.Stderr,
				"Repeatr encountered a serious issue and was unable to complete your request!\n"+
					"Please file an issue to help us fix it.\n"+
					saveMsg+"\n"+
					"\n"+
					"This is the short version of the problem:\n"+
					"%s\n",
				err)
			os.Exit(int(cli.EXIT_UNKNOWNPANIC))
		}
	})
}

func isDebugMode() bool {
	// if either "DEBUG" or "REPEATR_DEBUG" env vars are set, we're in debug mode.
	return len(os.Getenv("DEBUG")) != 0 || len(os.Getenv("REPEATR_DEBUG")) != 0
}

func saveErrorReport(caught error) (string, error) {
	logFile, err := ioutil.TempFile(os.TempDir(), "repeatr-error-report-")
	if err != nil {
		return "", err
	}
	defer logFile.Close()
	fmt.Fprintf(logFile, "Repeatr error report\n")
	fmt.Fprintf(logFile, "====================\n")
	fmt.Fprintf(logFile, "Date: %s\n", time.Now())
	fmt.Fprintf(logFile, "\n")
	fmt.Fprintf(logFile, "Full error:\n")
	fmt.Fprintf(logFile, "-----------\n")
	fmt.Fprintf(logFile, "%s\n", caught)
	fmt.Fprintf(logFile, "\n")
	// TODO full stack viewed from here.  yank the formatting stuff from spacemonkey errors
	return logFile.Name(), nil
}
