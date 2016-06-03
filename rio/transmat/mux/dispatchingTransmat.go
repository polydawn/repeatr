package dispatch

import (
	"github.com/inconshreveable/log15"

	"polydawn.net/repeatr/rio"
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

func (dt *Transmat) Materialize(kind rio.TransmatKind, dataHash rio.CommitID, siloURIs []rio.SiloURI, log log15.Logger, options ...rio.MaterializerConfigurer) rio.Arena {
	transmat := dt.dispatch[kind]
	if transmat == nil {
		panic(rio.ConfigError.New("no transmat of kind %q available to satisfy request", kind))
	}
	return transmat.Materialize(kind, dataHash, siloURIs, log, options...)
}

func (dt *Transmat) Scan(kind rio.TransmatKind, subjectPath string, siloURIs []rio.SiloURI, log log15.Logger, options ...rio.MaterializerConfigurer) rio.CommitID {
	transmat := dt.dispatch[kind]
	if transmat == nil {
		panic(rio.ConfigError.New("no transmat of kind %q available to satisfy request", kind))
	}
	return transmat.Scan(kind, subjectPath, siloURIs, log, options...)
}
