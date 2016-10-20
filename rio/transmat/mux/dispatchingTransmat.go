package dispatch

import (
	"fmt"

	"github.com/inconshreveable/log15"

	"go.polydawn.net/repeatr/api/def"
	"go.polydawn.net/repeatr/rio"
)

var _ rio.Transmat = &Transmat{}

/*
	`dispatch.Transmat` gathers a bunch of Transmats under one entrypoint,
	so that any kind of data specification can be fed into this one `Materialize`
	function, and it will DTRT.
*/
type Transmat struct {
	dispatch map[rio.TransmatKind]rio.Transmat
}

func New(transmats map[rio.TransmatKind]rio.Transmat) *Transmat {
	dt := &Transmat{
		dispatch: transmats,
	}
	return dt
}

/*
	Dispatches the materialize call to one of this dispatcher's configured transmats.

	May panic with:

	  - `*def.ErrConfigValidation` -- if there's no transmat set up to handle the requested kind.
	  - whatever else the dispatched transmat may panic with.
*/
func (dt *Transmat) Materialize(kind rio.TransmatKind, dataHash rio.CommitID, siloURIs []rio.SiloURI, log log15.Logger, options ...rio.MaterializerConfigurer) rio.Arena {
	transmat := dt.dispatch[kind]
	if transmat == nil {
		panic(&def.ErrConfigValidation{
			Msg: fmt.Sprintf("no transmat of kind %q available to satisfy request", kind),
		})
	}
	return transmat.Materialize(kind, dataHash, siloURIs, log, options...)
}

/*
	Dispatches the scan call to one of this dispatcher's configured transmats.

	May panic with:

	  - `*def.ErrConfigValidation` -- if there's no transmat set up to handle the requested kind.
	  - whatever else the dispatched transmat may panic with.
*/
func (dt *Transmat) Scan(kind rio.TransmatKind, subjectPath string, siloURIs []rio.SiloURI, log log15.Logger, options ...rio.MaterializerConfigurer) rio.CommitID {
	transmat := dt.dispatch[kind]
	if transmat == nil {
		panic(&def.ErrConfigValidation{
			Msg: fmt.Sprintf("no transmat of kind %q available to satisfy request", kind),
		})
	}
	return transmat.Scan(kind, subjectPath, siloURIs, log, options...)
}
