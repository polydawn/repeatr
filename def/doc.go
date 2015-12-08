/*
	Repeatr is focused on telling a story about formulas: when you put the same
	things in, you should get the same things out.

	Formulas describe a piece of computation, its inputs, and how to collect
	its outputs.  After that, repeatr can help you make sure your formula
	produces the same thing time and time again.

	We call the parts of the formula that should be deterministic the "conjecture".
	We'll use this word consistently throughout the documentation.
	Anything that is part of the conjecture is hashed when processing the formula,
	and any output marked as part of the conjecture is hashed after the formula's task
	is executed.  (You can choose which of the outputs are part of your conjecture!
	But everything about your inputs must be part of your conjecture, because
	if the inputs change, output consistency is impossible -- except stuff like
	the network locations of data is skipped from the conjecture, since
	that can change without changing the meaning of your formula.)

	### Mathwise:

	Given a Formula j, and the []Output v, and some hash h:

	h(j.Inputs||j.Action||filter(j.Outputs, where Conjecture=true)) -> h(v)

	should be an onto relationship.

	In other words, a Formula should define a "pure" function.  And we'll let you know if it doesn't.

	### Misc docs:

	- The root filesystem of your execution engine is just another `Input` with the rest, with MountPath="/".
	Exactly one input with the root location is required at runtime.

	- Formula.SchedulingInfo, since it's *not* included in the 'conjecture',
	is expected not to have a major impact on your execution correctness.
*/
package def
