package def

import (
	"time"
)

var Somewhen time.Time = time.Date(2000, time.January, 15, 0, 0, 0, 0, time.UTC)
var SomewhenNano int64 = Somewhen.UnixNano()
