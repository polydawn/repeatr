package errcat

import "fmt"

/*
	This file contains some helpers that are useful for users of the
	GoConvey testing system.
	(It's easy to include it in this package because the GoConvey
	assertion functions don't require any GoConvey *types*, so we don't
	have to add any dependencies in order to provide these helpers.)
*/

// ShouldErrorWith is a helper function for GoConvey-style testing:
// it takes an "actual" value (which should be an error), and a description
// of what errcat "Category" it should have, and returns
// either an empty string if the error matches the category,
// or a string describing how it's out of line if there's no match.
// (In other words: return of empty string means "success!",
// and any other string means "failure: because %s".)
func ShouldErrorWith(actual interface{}, expectedClause ...interface{}) string {
	if len(expectedClause) != 1 {
		return "Misuse: ShouldHaveCategory predicate needs exactly one item in the \"expected\" clause"
	}
	expected := expectedClause[0]
	if actual == nil && expected == nil {
		return "" // good!
	}
	if actual == nil {
		return fmt.Sprintf("Actual: nil\nExpected category: %q", expected)
	}
	_, ok := actual.(error)
	if !ok {
		return fmt.Sprintf("Actual: %v\nExpected category: %q\nShould have error interface type!  (This is probaby a misuse of the ShouldHaveCategory predicate...)", actual, expected)
	}
	e2, ok := actual.(*Error)
	if !ok {
		return fmt.Sprintf("Actual: %v\nExpected category: %q\nShould have an errcat error!  Was type %T.", actual, expected, actual)
	}
	if e2.Category != expected {
		return fmt.Sprintf("Actual category: %q\nExpected category: %q\n(Full error: %v)", e2.Category, expected, actual)
	}
	return "" // couldn't find grounds to reject it; must be good!
}
