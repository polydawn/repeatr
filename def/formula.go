package def

/*
	Formula describes `(inputs, computation) -> (outputs)`.

	Values may be mutated during final validation if missing,
	i.e. the special `Output` that describes stdout and stderr is required
	and will be supplied for you if not already specifically configured.
*/
type Formula struct {
	Inputs  []Input  `json:"inputs"`  // total set of inputs.  sorted order.  included in the conjecture.
	Action  Action   `json:"action"`  // description of the computation to be performed.  included in the conjecture.
	Outputs []Output `json:"outputs"` // set of expected outputs.  sorted order.  conditionally included in the conjecture (configurable per output).
	//SchedulingInfo interface{} // configures what execution framework is used and impl-specific additional parameters to that (minimum node memory, etc).  not considered part of the conjecture.
}
