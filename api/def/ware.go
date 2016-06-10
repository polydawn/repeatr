package def

/*
	Ware is an identifier for an object -- but not necessarily instructions on
	how to get it.  You can think of Wares a lot like SKUs
	(https://en.wikipedia.org/wiki/Stock-keeping_unit): they're great for
	keeping stock of things, but not actually a direct shipping plan for when
	you're out of stock.  (`Warehouse`s describe that part.)
*/
type Ware struct {
	Type string `json:"type"`
	Hash string `json:"hash"`
}
