package testutil

import (
	"os"

	"github.com/smartystreets/goconvey/convey"
)

func Convey_IfHaveRoot(items ...interface{}) {
	if os.Getuid() == 0 {
		convey.Convey(items...)
	} else {
		convey.SkipConvey(items...)
	}
}
