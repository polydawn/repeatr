/*
	The `def` package outlines definitions of all the major configurations
	for repeatr components.  Use the sibling `act` package to see the verbs
	that relate them.

	Shortlist:

		- `Ware`s name a piece of data

		- `Warehouse`s describe where you can get pieces of data from

		- `Formula`s describe `action(input)->(output)`
		  They're broken down into...
		    - Inputs (a list of Wares and mount path instructions)
		    - Outputs (a list of gather path instructions)
		    - Actions (a command, environment, and policies)

		- `RunRecord`s describe what happen when a Formula was executed
		  (in other words, the actual outputs -- as Wares!)

		- `Catalog`s, like in shopping, map a human-readable description
		  onto the machine-friendly `Ware` ID numbers.

		- `Commission`s are like Formulas, but can point to Catalogs instead
		  of directly at Wares (they provide an update system).
*/
package def
