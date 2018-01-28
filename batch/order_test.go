package batch

import (
	"testing"

	"go.polydawn.net/go-timeless-api"
	. "go.polydawn.net/repeatr/testutil"
)

func TestNilRelationLexicalOrdering(t *testing.T) {
	basting := api.Basting{Steps: map[string]api.BastingStep{
		"stepD": {},
		"stepB": {},
		"stepA": {},
		"stepC": {},
	}}
	order, err := orderSteps(basting)
	WantEqual(t, err, nil)
	WantEqual(t, order, []string{
		"stepA",
		"stepB",
		"stepC",
		"stepD",
	})
}

func TestFanoutLexicalOrdering(t *testing.T) {
	basting := api.Basting{Steps: map[string]api.BastingStep{
		"stepD": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "step0", "/out"},
		}},
		"stepB": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "step0", "/out"},
		}},
		"stepA": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "step0", "/out"},
		}},
		"stepC": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "step0", "/out"},
		}},
		"step0": {},
	}}
	order, err := orderSteps(basting)
	WantEqual(t, err, nil)
	WantEqual(t, order, []string{
		"step0",
		"stepA",
		"stepB",
		"stepC",
		"stepD",
	})
}

func TestFanInLexicalOrdering(t *testing.T) {
	basting := api.Basting{Steps: map[string]api.BastingStep{
		"stepD": {},
		"stepB": {},
		"stepA": {},
		"stepC": {},
		"step9": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/":  {"wire", "stepA", "/out"},
			"/1": {"wire", "stepB", "/out"},
			"/2": {"wire", "stepC", "/out"},
			"/3": {"wire", "stepD", "/out"},
		}},
	}}
	order, err := orderSteps(basting)
	WantEqual(t, err, nil)
	WantEqual(t, order, []string{
		"stepA",
		"stepB",
		"stepC",
		"stepD",
		"step9",
	})
}

func TestSimpleLinearOrdering(t *testing.T) {
	basting := api.Basting{Steps: map[string]api.BastingStep{
		"stepA": {},
		"stepB": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "stepA", "/out"},
		}},
		"stepC": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "stepB", "/out"},
		}},
		"stepD": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "stepC", "/out"},
		}},
	}}
	order, err := orderSteps(basting)
	WantEqual(t, err, nil)
	WantEqual(t, order, []string{
		"stepA",
		"stepB",
		"stepC",
		"stepD",
	})
}

func TestComplexOrdering(t *testing.T) {
	/*
		             /------------> K --\
		             |                   \
		A --> B -----E ------> H --------> L
		            /                     /
		  C --> D -----F --> G ----------/
		               |
		               \------> I----------> M
		               |                    /
		               \--------> J -------/
	*/
	basting := api.Basting{Steps: map[string]api.BastingStep{
		"stepA": {},
		"stepB": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "stepA", "/out"},
		}},
		"stepC": {},
		"stepD": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "stepC", "/out"},
		}},
		"stepE": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/":  {"wire", "stepB", "/out"},
			"/1": {"wire", "stepD", "/out"},
		}},
		"stepF": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "stepD", "/out"},
		}},
		"stepG": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "stepF", "/out"},
		}},
		"stepH": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "stepE", "/out"},
		}},
		"stepI": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "stepF", "/out"},
		}},
		"stepJ": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "stepF", "/out"},
		}},
		"stepK": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/": {"wire", "stepE", "/out"},
		}},
		"stepL": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/":  {"wire", "stepG", "/out"},
			"/1": {"wire", "stepK", "/out"},
			"/2": {"wire", "stepH", "/out"},
		}},
		"stepM": {Imports: map[api.AbsPath]api.ReleaseItemID{
			"/":  {"wire", "stepI", "/out"},
			"/1": {"wire", "stepJ", "/out"},
		}},
	}}
	order, err := orderSteps(basting)
	WantEqual(t, err, nil)
	WantEqual(t, order, []string{
		"stepA", "stepB", "stepC", "stepD", "stepE",
		"stepF", "stepG", "stepH", "stepI", "stepJ",
		"stepK", "stepL", "stepM",
	})
}
