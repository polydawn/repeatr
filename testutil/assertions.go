package testutil

import (
	"fmt"
	"os"
	//	"path/filepath"

	//	"github.com/smartystreets/goconvey/convey"
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
