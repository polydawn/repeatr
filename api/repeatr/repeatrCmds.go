/*
	Interfaces of repeatr commands.

	The real repeatr logic implements these;
	so do the various proxy tools (e.g. r2k8s);
	so do some mocks, which are useful for testing.

	A serial client is included in this package;
	how to launch any remote actions is not in this package.
*/
package repeatr

import (
	"context"
	"time"

	"go.polydawn.net/repeatr/api"
)

type RunFunc func(
	ctx context.Context,
	formula *api.Formula,
	stream chan<- *Event,
) (*api.RunRecord, error)

/*
	A "union" type of all the kinds of event that may be generated in the
	course of running a formula.

	(RunRecords aren't forwarded through 'stream' chans like this, but they're
	defined in the union because it's about where they're sent on the wire.)
*/
type Event struct {
	Log       *LogItem       `refmt:"log,omitempty"`       // Log events are repeatr's logs.
	Journal   string         `refmt:"journal,omitempty"`   // User output strings are called journal events.
	RunRecord *api.RunRecord `refmt:"runRecord,omitempty"` // RunRecords are the final event emitted by a run.
}

/*
	Log events from repeatr as serialized.
*/
type LogItem struct {
	Time  time.Time         `refmt:"t"`
	Level int               `refmt:"lvl"`
	Msg   string            `refmt:"msg"`
	Ctx   map[string]string `refmt:"ctx,omitempty"`
}
