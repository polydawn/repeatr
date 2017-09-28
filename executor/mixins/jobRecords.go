package mixins

import (
	"os"
	"time"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/rio/lib/guid"
)

/*
	Initialize all the fields in a RunRecord for a new job.

	Includes setting the UID (so you can turn around and use that for
	tempfiles and such!), and several host-specific things,
	like the current time and the hostname.
*/
func InitRunRecord(rr *api.RunRecord, frm api.Formula) {
	rr.Guid = guid.New()
	rr.Time = time.Now().Unix()
	rr.FormulaID = frm.SetupHash()
	rr.Results = map[api.AbsPath]api.WareID{}
	rr.ExitCode = -1
	rr.Hostname, _ = os.Hostname()
}
