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
