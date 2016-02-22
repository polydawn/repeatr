package def

/*
	Formula describes `(inputs, computation) -> (outputs)`.

	Values may be mutated during final validation if missing,
	i.e. the special `Output` that describes stdout and stderr is required
	and will be supplied for you if not already specifically configured.
*/
type Formula struct {
	Inputs  InputGroup  `json:"inputs"`  // total set of inputs.  sorted order.  included in the conjecture.
	Action  Action      `json:"action"`  // description of the computation to be performed.  included in the conjecture.
	Outputs OutputGroup `json:"outputs"` // set of expected outputs.  sorted order.  conditionally included in the conjecture (configurable per output).
	//SchedulingInfo interface{} // configures what execution framework is used and impl-specific additional parameters to that (minimum node memory, etc).  not considered part of the conjecture.
}

func (f Formula) Clone() *Formula {
	f.Inputs = f.Inputs.Clone()
	f.Action = f.Action.Clone()
	f.Outputs = f.Outputs.Clone()
	return &f
}
