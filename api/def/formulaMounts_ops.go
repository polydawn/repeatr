package def

import (
	"bytes"

	"github.com/ugorji/go/codec"
)

func (g InputGroup) Clone() InputGroup {
	r := make(InputGroup, len(g))
	for k, v := range g {
		r[k] = v.Clone()
	}
	return r
}

func (g OutputGroup) Clone() OutputGroup {
	r := make(OutputGroup, len(g))
	for k, v := range g {
		r[k] = v.Clone()
	}
	return r
}

func (i Input) Clone() *Input {
	w2 := make(WarehouseCoords, len(i.Warehouses))
	copy(w2, i.Warehouses)
	i.Warehouses = w2
	return &i
}

func (i Input) String() string {
	var buf bytes.Buffer
	codec.NewEncoder(&buf, &codec.JsonHandle{}).Encode(i)
	return buf.String()
}

func (o Output) Clone() *Output {
	w2 := make(WarehouseCoords, len(o.Warehouses))
	copy(w2, o.Warehouses)
	o.Warehouses = w2
	// filters are complex but also all immutable, so ignorable
	return &o
}

func (o Output) String() string {
	var buf bytes.Buffer
	codec.NewEncoder(&buf, &codec.JsonHandle{}).Encode(o)
	return buf.String()
}

// Default filters for input are to respect everything.
func (f *Filters) InitDefaultsInput() {
	if f.UidMode == FilterUninitialized {
		f.UidMode = FilterKeep
	}
	if f.GidMode == FilterUninitialized {
		f.GidMode = FilterKeep
	}
	if f.MtimeMode == FilterUninitialized {
		f.MtimeMode = FilterKeep
	}
}

// Default filters for output are to flatten uid, gid, and mtime.
func (f *Filters) InitDefaultsOutput() {
	if f.UidMode == FilterUninitialized {
		f.UidMode = FilterUse
		f.Uid = FilterDefaultUid
	}
	if f.GidMode == FilterUninitialized {
		f.GidMode = FilterUse
		f.Gid = FilterDefaultGid
	}
	if f.MtimeMode == FilterUninitialized {
		f.MtimeMode = FilterUse
		f.Mtime = FilterDefaultMtime
	}
}
