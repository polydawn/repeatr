package def

/*
	Formula describes `action(inputs) -> (outputs)`.
*/
type Formula struct {
	Inputs  InputGroup  `json:"inputs"`
	Action  Action      `json:"action"`
	Outputs OutputGroup `json:"outputs"`
}
