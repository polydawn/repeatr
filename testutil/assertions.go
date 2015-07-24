package testutil

import (
	"fmt"
	"os"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/try"
)

/*
	'actual' should be path; 'expected' may be empty (in which case it checks
	that anything with an inode exists) or a filemode (in which case it will
	check for that file type, ignoring all bits outside of the `os.ModeType`
	range; and '0' must be used for plain file (there is no const)).
*/
func ShouldBeFile(actual interface{}, expected ...interface{}) string {
	filename, ok := actual.(string)
	if !ok {
		return "You must provide a filename as the first argument to this assertion."
	}

	info, err := os.Stat(filename)
	if err != nil {
		// includes if os.IsNotExist(err)
		return err.Error()
	}

	switch len(expected) {
	case 0:
		return "" // not picky about mode?  okay, you pass already.
	case 1:
		mode, ok := expected[0].(os.FileMode)
		if !ok {
			return "You must provide a FileMode as the second argument to this assertion, if any."
		}
		modeType := info.Mode() & os.ModeType
		if modeType != mode {
			return fmt.Sprintf("Expected file to have mode %v but it had %v instead!", mode, modeType)
		}
		return ""
	default:
		return "You must provide zero or one parameters as expectations to this assertion."
	}
}

/*
	'actual' should be an `*errors.Error`; 'expected' should be an `*errors.ErrorClass`;
	we'll check that the error is under the umbrella of the error class.
*/
func ShouldBeErrorClass(actual interface{}, expected ...interface{}) string {
	err, ok := actual.(error)
	if !ok {
		return fmt.Sprintf("You must provide an `error` as the first argument to this assertion; got `%T`", actual)
	}

	var class *errors.ErrorClass
	switch len(expected) {
	case 0:
		return "You must provide a spacemonkey `ErrorClass` as the expectation parameter to this assertion."
	case 1:
		cls, ok := expected[0].(*errors.ErrorClass)
		if !ok {
			return "You must provide a spacemonkey `ErrorClass` as the expectation parameter to this assertion."
		}
		class = cls
	default:
		return "You must provide one parameter as an expectation to this assertion."
	}

	spaceClass := errors.GetClass(err)
	if spaceClass.Is(class) {
		return ""
	}
	return fmt.Sprintf("Expected error to be of class %q but it had %q instead!  (Full message: %s)", class.String(), spaceClass.String(), err.Error())
}

/*
	'actual' should be a `func()`; 'expected' should be an `*errors.ErrorClass`;
	we'll run the function, and check that it panics, and that the error is under the umbrella of the error class.
*/
func ShouldPanicWith(actual interface{}, expected ...interface{}) string {
	fn, ok := actual.(func())
	if !ok {
		return fmt.Sprintf("You must provide a `func()` as the first argument to this assertion; got `%T`", actual)
	}

	var errClass *errors.ErrorClass
	switch len(expected) {
	case 0:
		return "You must provide a spacemonkey `ErrorClass` as the expectation parameter to this assertion."
	case 1:
		cls, ok := expected[0].(*errors.ErrorClass)
		if !ok {
			return "You must provide a spacemonkey `ErrorClass` as the expectation parameter to this assertion."
		}
		errClass = cls
	default:
		return "You must provide one parameter as an expectation to this assertion."
	}

	var caught error
	try.Do(
		fn,
	).CatchAll(func(err error) {
		caught = err
	}).Done()

	if caught == nil {
		return fmt.Sprintf("Expected error to be of class %q but no error was raised!", errClass.String())
	}
	spaceClass := errors.GetClass(caught)
	if spaceClass.Is(errClass) {
		return ""
	}
	return fmt.Sprintf("Expected error to be of class %q but it had %q instead!  (Full message: %s)", errClass.String(), spaceClass.String(), caught.Error())
}
